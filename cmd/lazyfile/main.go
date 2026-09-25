package main

import (
	"fmt"
	"os"

	"lazyfile/internal/ui"

	"github.com/gdamore/tcell/v3"
)

func main() {
	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "create screen: %v\n", err)
		os.Exit(1)
	}
	if err := screen.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "initialize screen: %v\n", err)
		os.Exit(1)
	}
	defer screen.Fini()

	initialDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "get current directory: %v\n", err)
		os.Exit(1)
	}
	app, err := ui.New(screen, initialDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load current directory: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "run application: %v\n", err)
		os.Exit(1)
	}
}
