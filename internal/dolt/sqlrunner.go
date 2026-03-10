package dolt

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
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
// be started or connected to, it returns an error.
func NewSQLRunner(repoDir string) (*SQLRunner, error) {
	cli, err := NewCLIRunner(repoDir)
	if err != nil {
		return nil, err
	}

	dbName := filepath.Base(cli.repoDir)

	r := &SQLRunner{
		CLIRunner: cli,
		dbName:    dbName,
	}

	// Try connecting to an existing sql-server first.
	if info, err := r.readServerInfo(); err == nil {
		if db, err := r.connectToServer(info.port); err == nil {
			r.db = db
			r.serverPort = info.port
			return r, nil
		}
	}

	// No existing server — start our own.
	port, err := r.findFreePort()
	if err != nil {
		return nil, fmt.Errorf("finding free port for sql-server: %w", err)
	}

	if err := r.startServer(port); err != nil {
		return nil, fmt.Errorf("starting dolt sql-server: %w", err)
	}

	db, err := r.connectToServer(port)
	if err != nil {
		// Clean up the server we just started.
		r.stopServer()
		return nil, fmt.Errorf("connecting to dolt sql-server: %w", err)
	}

	r.db = db
	r.serverPort = port
	return r, nil
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
