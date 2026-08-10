package ui

import (
	"io"

	"fyne.io/fyne/v2"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

type mailboxStore interface {
	List(folder mailbox.Folder) ([]mailbox.Message, error)
	Save(msg mailbox.Message) error
	Move(from, to mailbox.Folder, id string) error

	AddAttachmentReader(
		folder mailbox.Folder,
		messageID string,
		name string,
		source io.Reader,
	) (mailbox.Attachment, error)

	RemoveAttachment(
		folder mailbox.Folder,
		messageID string,
		attachmentID string,
	) error

	OpenAttachmentReader(
		folder mailbox.Folder,
		messageID string,
		attachmentID string,
	) (io.ReadCloser, mailbox.Attachment, error)
}

type IdentityFunc func() (
	callsign string,
	locator string,
)

func NewMainWindow(
	a fyne.App,
	title string,
	store mailboxStore,
	identity IdentityFunc,
	updateIdentity IdentityUpdateFunc,
	connectCMS CMSConnectFunc,
) (fyne.Window, error) {
	w := a.NewWindow(title)

	content, err := newMailShell(
		a,
		w,
		store,
		identity,
		updateIdentity,
		connectCMS,
	)
	if err != nil {
		return nil, err
	}

	w.SetContent(content)
	w.Resize(fyne.NewSize(1200, 800))

	return w, nil
}
