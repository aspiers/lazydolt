package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aspiers/lazydolt/internal/dolt"
	"github.com/aspiers/lazydolt/internal/ui"
)

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	var runner dolt.Runner
	sqlRunner, err := dolt.NewSQLRunner(dir)
	if err != nil {
		// Fall back to CLI runner if sql-server can't start.
		fmt.Fprintf(os.Stderr, "Warning: sql-server unavailable (%v), falling back to CLI\n", err)
		cliRunner, cliErr := dolt.NewCLIRunner(dir)
		if cliErr != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", cliErr)
			os.Exit(1)
		}
		runner = cliRunner
	} else {
		runner = sqlRunner
	}
	defer runner.Close()

	app := ui.NewApp(runner)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
