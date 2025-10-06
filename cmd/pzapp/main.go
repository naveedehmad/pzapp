package main

import (
	"log"
	"os"

	"portkiller/internal/ports"
	"portkiller/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	var provider ports.Provider
	if os.Getenv("PZAPP_USE_MOCK") == "1" {
		provider = ports.NewMockProvider()
	} else {
		provider = ports.NewSystemProvider()
	}

	program := tea.NewProgram(ui.New(provider))

	if err := program.Start(); err != nil {
		log.Fatalf("failed to start pzapp: %v", err)
	}
}
