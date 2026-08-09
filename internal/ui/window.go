package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

func NewMainWindow(a fyne.App, title string) fyne.Window {
	w := a.NewWindow(title)

	w.SetContent(widget.NewLabel(title))
	w.Resize(fyne.NewSize(1200, 800))

	return w
}
