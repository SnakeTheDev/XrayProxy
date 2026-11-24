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

	// NewPanel installs the native system-tray icon and menu. Do not show
	// the Fyne window: the Windows notification-area menu is the interface.
	_ = ui.NewPanel(a, dir)
	a.Run()
}
