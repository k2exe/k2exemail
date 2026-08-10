package ui

import (
	"fyne.io/fyne/v2"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

type mailboxReader interface {
	List(folder mailbox.Folder) ([]mailbox.Message, error)
}

func NewMainWindow(
	a fyne.App,
	title string,
	messages mailboxReader,
) (fyne.Window, error) {
	content, err := newMailShell(messages)
	if err != nil {
		return nil, err
	}

	w := a.NewWindow(title)
	w.SetContent(content)
	w.Resize(fyne.NewSize(1200, 800))

	return w, nil
}
