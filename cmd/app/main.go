package main

import (
	"fmt"
	"os"

	"tui-ssh-docker/internal/app"
	"tui-ssh-docker/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	a, err := app.NewApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize app: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(ui.InitialModel(a), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
