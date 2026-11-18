package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"env-editor/internal/envfile"
	"env-editor/internal/i18n"
	"env-editor/internal/tui"
)

func main() {
	i18n.Init()
	path, masked := ".env", false
	for _, arg := range os.Args[1:] {
		switch arg {
		case "-h", "--help":
			fmt.Print(i18n.Usage.String() + tui.KeyHelp())
			return
		case "-m", "--mask":
			masked = true
		default:
			path = arg
		}
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}

	doc, err := envfile.Load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, i18n.ErrPrefix.String(), err)
		os.Exit(1)
	}
	if _, err := tea.NewProgram(tui.New(path, doc, masked), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, i18n.ErrPrefix.String(), err)
		os.Exit(1)
	}
}
