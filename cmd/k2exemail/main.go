package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"
)

const (
	appID   = "com.k2exe.k2exemail"
	appName = "K2EXEmail"
)

func main() {
	a := app.NewWithID(appID)
	w := a.NewWindow(appName)

	w.SetContent(widget.NewLabel("K2EXEmail"))
	w.Resize(fyne.NewSize(1200, 800))

	w.ShowAndRun()
}
