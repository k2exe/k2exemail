package main

import (
	"fyne.io/fyne/v2/app"

	"github.com/k2exe/k2exemail/internal/ui"
)

const (
	appID   = "com.k2exe.k2exemail"
	appName = "K2EXEmail"
)

func main() {
	a := app.NewWithID(appID)
	w := ui.NewMainWindow(a, appName)

	w.ShowAndRun()
}
