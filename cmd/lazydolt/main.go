package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aspiers/lazydolt/internal/dolt"
	"github.com/aspiers/lazydolt/internal/ui"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [directory]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A TUI for Dolt databases, inspired by lazygit.\n\n")
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  directory    Path to a Dolt repository (default: current directory)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
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
