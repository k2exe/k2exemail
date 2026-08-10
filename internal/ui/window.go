package ui

import (
	"fyne.io/fyne/v2"
)

func NewMainWindow(a fyne.App, title string) fyne.Window {
	w := a.NewWindow(title)
	w.SetContent(newMailShell())
	w.Resize(fyne.NewSize(1200, 800))
	return w
}
