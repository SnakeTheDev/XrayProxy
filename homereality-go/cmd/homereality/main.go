package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2/app"

	"github.com/SnakeTheDev/XrayProxy/homereality-go/internal/ui"
)

func main() {
	// Resolve data dir early so we can log next to the binary.
	dir, err := ui.DataDir()
	if err != nil {
		// Last resort before logging is set up.
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Always log to app.log next to the binary (works with -H windowsgui / no console).
	// If HOMEREALITY_DEBUG=1 is set, also mirror to stderr (useful when built with a console).
	logPath := filepath.Join(dir, "app.log")
	lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err == nil {
		defer lf.Close()
		if os.Getenv("HOMEREALITY_DEBUG") == "1" {
			log.SetOutput(io.MultiWriter(lf, os.Stderr))
		} else {
			log.SetOutput(lf)
		}
	}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Home-Reality starting, data dir:", dir)

	a := app.NewWithID("com.homereality.client")

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

	// Tray-only UI: window is created but never shown unless user picks "Show panel".
	_ = ui.NewPanel(a, dir)
	a.Run()
}
