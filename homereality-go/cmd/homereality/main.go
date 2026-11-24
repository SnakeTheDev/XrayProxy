package main

import (
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2/app"

	"github.com/SnakeTheDev/XrayProxy/homereality-go/internal/ui"
)

func main() {
	a := app.NewWithID("com.homereality.client")

	dir, err := ui.DataDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	candidates := []string{
		filepath.Join(dir, "sing-box-config.json"),
		"sing-box-config.json",
	}
	var cfgSrc string
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			cfgSrc = c
			break
		}
	}
	if cfgSrc != "" {
		_ = ui.EnsureConfig(dir, cfgSrc)
	}

	panel := ui.NewPanel(a, dir)
	// Show once so you can find it; X / close hides to system tray.
	panel.Show()
	a.Run()
}
