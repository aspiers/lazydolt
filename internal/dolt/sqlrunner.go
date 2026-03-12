package dolt

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/aspiers/lazydolt/internal/domain"
)

// Compile-time check that SQLRunner implements Runner.
var _ Runner = (*SQLRunner)(nil)

// SQLRunner executes dolt operations via a persistent sql-server process
// connected over MySQL protocol. Operations not yet migrated to SQL
// fall back to the embedded CLIRunner.
type SQLRunner struct {
	*CLIRunner           // embedded for CLI fallback
	db         *sql.DB   // MySQL connection pool
	serverCmd  *exec.Cmd // nil if we connected to an existing server
	serverPort int
	dbName     string // database name (directory basename)
}

// serverInfo holds parsed .dolt/sql-server.info data.
type serverInfo struct {
	pid  int
	port int
}

// NewSQLRunner creates a Runner backed by a dolt sql-server.
// It first tries to connect to an existing sql-server for the repo,
// then falls back to starting a new one. If the sql-server cannot
// be started or connected to, it returns an error that includes
// diagnostics from each failed attempt, one per line.
func NewSQLRunner(repoDir string) (*SQLRunner, error) {
	cli, err := NewCLIRunner(repoDir)
	if err != nil {
		return nil, err
	}
	return NewSQLRunnerFrom(cli)
}

// NewSQLRunnerFrom creates a Runner backed by a dolt sql-server,
// using an existing CLIRunner. This avoids redundant repo validation
// when the caller already has a valid CLIRunner to fall back to.
func NewSQLRunnerFrom(cli *CLIRunner) (*SQLRunner, error) {
	dbName := filepath.Base(cli.repoDir)

	r := &SQLRunner{
		CLIRunner: cli,
		dbName:    dbName,
	}

	// Collect diagnostic messages from each failed attempt.
	var diagnostics []string

	// Try connecting to an existing sql-server first.
	info, err := r.readServerInfo()
	if err != nil {
		diagnostics = append(diagnostics,
			fmt.Sprintf("no existing sql-server: %v", err))
	} else {
		db, err := r.connectToServer(info.port)
		if err != nil {
			diagnostics = append(diagnostics,
				fmt.Sprintf("existing sql-server (pid %d, port %d) not reachable: %v",
					info.pid, info.port, err))
		} else {
			r.db = db
			r.serverPort = info.port
			r.injectSQLFunc()
			return r, nil
		}
	}

	// No existing server — start our own.
	port, err := r.findFreePort()
	if err != nil {
		diagnostics = append(diagnostics,
			fmt.Sprintf("finding free port: %v", err))
		return nil, formatSQLRunnerError(diagnostics)
	}

	if err := r.startServer(port); err != nil {
		diagnostics = append(diagnostics,
			fmt.Sprintf("starting new sql-server: %v", err))
		return nil, formatSQLRunnerError(diagnostics)
	}

	db, err := r.connectToServer(port)
	if err != nil {
		// Clean up the server we just started.
		r.stopServer()
		diagnostics = append(diagnostics,
			fmt.Sprintf("started new sql-server on port %d but cannot connect: %v",
				port, err))
		return nil, formatSQLRunnerError(diagnostics)
	}

	r.db = db
	r.serverPort = port
	r.injectSQLFunc()
	return r, nil
}

// formatSQLRunnerError builds a multi-line error from diagnostic steps.
func formatSQLRunnerError(diagnostics []string) error {
	var b strings.Builder
	b.WriteString("sql-server unavailable:")
	for i, d := range diagnostics {
		b.WriteString(fmt.Sprintf("\n  %d. %s", i+1, d))
	}
	return fmt.Errorf("%s", b.String())
}

// injectSQLFunc replaces the embedded CLIRunner's sqlFunc with the
// SQLRunner's server-based implementation. This ensures that all
// CLIRunner methods calling sqlFunc (Status, Branches, Log, etc.)
// use the persistent connection instead of spawning subprocesses.
func (r *SQLRunner) injectSQLFunc() {
	r.CLIRunner.sqlFunc = r.serverSQL
}

// serverSQL executes a SQL query via the persistent sql-server connection.
// This is the strategy function injected into CLIRunner.sqlFunc.
func (r *SQLRunner) serverSQL(query string) ([]map[string]interface{}, error) {
	cmdStr := "dolt sql -r json -q " + query
	rows, err := r.db.Query(query)
	if err != nil {
		r.logCommand(cmdStr, err.Error(), true)
		return nil, fmt.Errorf("%s: %w", cmdStr, err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("getting columns: %w", err)
	}

	var result []map[string]interface{}
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}

		row := make(map[string]interface{}, len(cols))
		for i, col := range cols {
			row[col] = normalizeValue(vals[i])
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		r.logCommand(cmdStr, err.Error(), true)
		return nil, err
	}

	r.logCommand(cmdStr, fmt.Sprintf("%d rows", len(result)), false)
	return result, nil
}

// Close stops the sql-server (if we started it) and closes connections.
func (r *SQLRunner) Close() error {
	var errs []string

	if r.db != nil {
		if err := r.db.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("closing db: %v", err))
		}
		r.db = nil
	}

	if r.serverCmd != nil {
		r.stopServer()
	}

	if len(errs) > 0 {
		return fmt.Errorf("SQLRunner.Close: %s", strings.Join(errs, "; "))
	}
	return nil
}

// DB returns the underlying *sql.DB for direct access (useful for
// methods migrated to SQL in phases 3a/3b).
func (r *SQLRunner) DB() *sql.DB {
	return r.db
}

// normalizeValue converts SQL driver values to the types that callers
// expect (matching the JSON parsing behavior of CLIRunner.SQL).
// JSON numbers come back as float64, strings as string, etc.
func normalizeValue(v interface{}) interface{} {
	switch val := v.(type) {
	case []byte:
		return string(val)
	case int64:
		return float64(val) // match JSON number behavior
	case uint64:
		return float64(val)
	case int32:
		return float64(val)
	case float32:
		return float64(val)
	default:
		return val
	}
}

// SQLRaw overrides CLIRunner.SQLRaw. For now, this delegates to the
// CLI because the tabular format is hard to replicate from raw SQL rows.
// The performance impact is minimal since SQLRaw is only used for
// user-initiated SQL queries, not for data loading.
func (r *SQLRunner) SQLRaw(query string) (string, error) {
	return r.CLIRunner.SQLRaw(query)
}

// Tables overrides CLIRunner.Tables to use SQL queries instead of
// "dolt ls" + "dolt sql ... dolt_status" CLI calls.
func (r *SQLRunner) Tables() ([]domain.Table, error) {
	// Get table names via SHOW TABLES.
	tableRows, err := r.db.Query("SHOW TABLES")
	if err != nil {
		return nil, fmt.Errorf("SHOW TABLES: %w", err)
	}
	defer tableRows.Close()

	var names []string
	for tableRows.Next() {
		var name string
		if err := tableRows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	if err := tableRows.Err(); err != nil {
		return nil, err
	}

	// Get status via SQL() (which now uses the sql-server connection).
	statusEntries, err := r.Status()
	if err != nil {
		return nil, err
	}

	statusMap := make(map[string]*domain.StatusEntry)
	for i := range statusEntries {
		statusMap[statusEntries[i].TableName] = &statusEntries[i]
	}

	// Include tables from status that aren't in SHOW TABLES (e.g. deleted).
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		seen[name] = true
	}
	for _, e := range statusEntries {
		if !seen[e.TableName] {
			names = append(names, e.TableName)
		}
	}

	sort.Strings(names)

	tables := make([]domain.Table, 0, len(names))
	for _, name := range names {
		t := domain.Table{Name: name}
		if st, ok := statusMap[name]; ok {
			t.Status = st
		}
		tables = append(tables, t)
	}

	return tables, nil
}

// readServerInfo parses .dolt/sql-server.info if it exists.
// Format: "pid:port:uuid"
func (r *SQLRunner) readServerInfo() (*serverInfo, error) {
	infoPath := filepath.Join(r.repoDir, ".dolt", "sql-server.info")
	data, err := os.ReadFile(infoPath)
	if err != nil {
		return nil, err
	}

	parts := strings.SplitN(strings.TrimSpace(string(data)), ":", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid sql-server.info format")
	}

	pid, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("parsing pid from sql-server.info: %w", err)
	}

	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("parsing port from sql-server.info: %w", err)
	}

	// Verify the process is still running.
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("process %d not found: %w", pid, err)
	}
	// On Unix, FindProcess always succeeds. Send signal 0 to check.
	if err := proc.Signal(os.Signal(nil)); err != nil {
		return nil, fmt.Errorf("process %d not running: %w", pid, err)
	}

	return &serverInfo{pid: pid, port: port}, nil
}

// findFreePort asks the OS for an available TCP port.
func (r *SQLRunner) findFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}

// startServer launches a dolt sql-server process in the background.
func (r *SQLRunner) startServer(port int) error {
	cmd := exec.Command(r.doltPath, "sql-server",
		"-P", strconv.Itoa(port),
		"-H", "127.0.0.1",
		"--loglevel", "warning",
	)
	cmd.Dir = r.repoDir
	// Discard stdout/stderr — server logs to its own log.
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting sql-server on port %d: %w", port, err)
	}

	r.serverCmd = cmd
	r.logCommand(fmt.Sprintf("dolt sql-server -P %d (pid %d)", port, cmd.Process.Pid), "", false)
	return nil
}

// stopServer sends SIGTERM to the sql-server process and waits for it.
func (r *SQLRunner) stopServer() {
	if r.serverCmd == nil || r.serverCmd.Process == nil {
		return
	}
	_ = r.serverCmd.Process.Signal(os.Interrupt)

	// Wait with a timeout.
	done := make(chan error, 1)
	go func() { done <- r.serverCmd.Wait() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = r.serverCmd.Process.Kill()
		<-done
	}
	r.serverCmd = nil
}

// connectToServer attempts to connect to a dolt sql-server with retries.
func (r *SQLRunner) connectToServer(port int) (*sql.DB, error) {
	dsn := fmt.Sprintf("root@tcp(127.0.0.1:%d)/%s?timeout=2s", port, r.dbName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// Configure connection pool.
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(0) // connections live as long as the server

	// Retry connection with backoff — server may still be starting.
	var lastErr error
	for i := 0; i < 20; i++ {
		if err := db.Ping(); err == nil {
			return db, nil
		} else {
			lastErr = err
		}
		time.Sleep(100 * time.Millisecond)
	}

	db.Close()
	return nil, fmt.Errorf("sql-server on port %d not ready after 2s: %w", port, lastErr)
}
