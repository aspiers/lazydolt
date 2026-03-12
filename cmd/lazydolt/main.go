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

	// Validate the repo first — no point trying sql-server or CLI
	// fallback if it's not a dolt repository.
	cliRunner, err := dolt.NewCLIRunner(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var runner dolt.Runner
	sqlRunner, err := dolt.NewSQLRunnerFrom(cliRunner)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		fmt.Fprintf(os.Stderr, "Falling back to CLI runner.\n")
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
