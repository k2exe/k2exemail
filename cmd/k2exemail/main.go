package main

import (
	"log"

	"fyne.io/fyne/v2/app"

	"github.com/k2exe/k2exemail/internal/appdirs"
	"github.com/k2exe/k2exemail/internal/mailbox"
	"github.com/k2exe/k2exemail/internal/ui"
)

const (
	appID   = "com.k2exe.k2exemail"
	appName = "K2EXEmail"
)

func main() {
	dirs, err := appdirs.Default()
	if err != nil {
		log.Fatalf("determine application directories: %v", err)
	}

	store := mailbox.NewStore(dirs.Data)
	if err := store.Prepare(); err != nil {
		log.Fatalf("prepare mailbox: %v", err)
	}

	a := app.NewWithID(appID)

	w, err := ui.NewMainWindow(a, appName, store)
	if err != nil {
		log.Fatalf("create main window: %v", err)
	}

	w.ShowAndRun()
}
