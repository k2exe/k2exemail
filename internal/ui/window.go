package ui

import (
	"fyne.io/fyne/v2"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

type mailboxStore interface {
	List(folder mailbox.Folder) ([]mailbox.Message, error)
	Save(msg mailbox.Message) error
	Move(from, to mailbox.Folder, id string) error
}

func NewMainWindow(
	a fyne.App,
	title string,
	store mailboxStore,
	callsign string,
	locator string,
	connectCMS CMSConnectFunc,
) (fyne.Window, error) {
	w := a.NewWindow(title)

	content, err := newMailShell(
		a,
		w,
		store,
		callsign,
		locator,
		connectCMS,
	)
	if err != nil {
		return nil, err
	}

	w.SetContent(content)
	w.Resize(fyne.NewSize(1200, 800))

	return w, nil
}
