package ui

import (
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	fyneTest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

type countingMailboxStore struct {
	*mailbox.Store
	saves atomic.Int32
}

func (s *countingMailboxStore) Save(msg mailbox.Message) error {
	s.saves.Add(1)
	return s.Store.Save(msg)
}

func TestReaderMarksUnreadMessageReadOnOpen(t *testing.T) {
	app := fyneTest.NewApp()
	window := app.NewWindow("test")

	base := mailbox.NewStore(t.TempDir())
	if err := base.Prepare(); err != nil {
		t.Fatal(err)
	}

	msg := mailbox.Message{
		ID:      "message-1",
		Folder:  mailbox.FolderInbox,
		From:    "W2ABC",
		Subject: "Unread",
		Unread:  true,
	}
	if err := base.Save(msg); err != nil {
		t.Fatal(err)
	}

	store := &countingMailboxStore{Store: base}
	updated := make(chan mailbox.Message, 1)

	_, showMessage, _ := newReaderPane(
		window,
		store,
		folderMailView(mailbox.FolderInbox),
		nil,
		nil,
		nil,
		nil,
		func(msg mailbox.Message) {
			updated <- msg
		},
		nil,
	)

	showMessage(msg)

	select {
	case got := <-updated:
		if got.Unread {
			t.Fatal("updated message remained unread")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for automatic mark-read")
	}

	if got := store.saves.Load(); got != 1 {
		t.Fatalf("Save calls = %d, want 1", got)
	}

	persisted, err := base.Load(
		mailbox.FolderInbox,
		"message-1",
	)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Unread {
		t.Fatal("persisted message remained unread")
	}
}

func TestReaderDoesNotSaveAlreadyReadMessageOnOpen(t *testing.T) {
	app := fyneTest.NewApp()
	window := app.NewWindow("test")

	base := mailbox.NewStore(t.TempDir())
	if err := base.Prepare(); err != nil {
		t.Fatal(err)
	}

	store := &countingMailboxStore{Store: base}

	_, showMessage, _ := newReaderPane(
		window,
		store,
		folderMailView(mailbox.FolderInbox),
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	showMessage(mailbox.Message{
		ID:     "message-1",
		Folder: mailbox.FolderInbox,
		Unread: false,
	})

	if got := store.saves.Load(); got != 0 {
		t.Fatalf("Save calls = %d, want 0", got)
	}
}

func TestReaderDoesNotMarkDraftReadOnOpen(t *testing.T) {
	app := fyneTest.NewApp()
	window := app.NewWindow("test")

	base := mailbox.NewStore(t.TempDir())
	if err := base.Prepare(); err != nil {
		t.Fatal(err)
	}

	store := &countingMailboxStore{Store: base}

	_, showMessage, _ := newReaderPane(
		window,
		store,
		folderMailView(mailbox.FolderDrafts),
		func(mailbox.Message) {},
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	showMessage(mailbox.Message{
		ID:     "draft-1",
		Folder: mailbox.FolderDrafts,
		Unread: true,
	})

	if got := store.saves.Load(); got != 0 {
		t.Fatalf("Save calls = %d, want 0", got)
	}
}

type blockingMailboxStore struct {
	*mailbox.Store
	saveStarted chan struct{}
	releaseSave chan struct{}
}

func (s *blockingMailboxStore) Save(msg mailbox.Message) error {
	if msg.ID == "message-a" {
		close(s.saveStarted)
		<-s.releaseSave
	}

	return s.Store.Save(msg)
}

func TestReaderAutomaticReadCompletionPreservesNewSelection(
	t *testing.T,
) {
	app := fyneTest.NewApp()
	window := app.NewWindow("test")

	base := mailbox.NewStore(t.TempDir())
	if err := base.Prepare(); err != nil {
		t.Fatal(err)
	}

	messageA := mailbox.Message{
		ID:      "message-a",
		Folder:  mailbox.FolderInbox,
		From:    "W2AAA",
		Subject: "Message A",
		Unread:  true,
	}
	if err := base.Save(messageA); err != nil {
		t.Fatal(err)
	}

	messageB := mailbox.Message{
		ID:      "message-b",
		Folder:  mailbox.FolderInbox,
		From:    "W2BBB",
		Subject: "Message B",
		Unread:  false,
	}

	store := &blockingMailboxStore{
		Store:       base,
		saveStarted: make(chan struct{}),
		releaseSave: make(chan struct{}),
	}

	replied := make(chan mailbox.Message, 1)
	var replyAction *widget.ToolbarAction

	reader, showMessage, _ := newReaderPane(
		window,
		store,
		folderMailView(mailbox.FolderInbox),
		nil,
		func(msg mailbox.Message, _ bool) {
			replied <- msg
		},
		nil,
		nil,
		func(mailbox.Message) {
			if replyAction != nil {
				replyAction.OnActivated()
			}
		},
		nil,
	)

	toolbar := findReaderToolbar(reader)
	if toolbar == nil {
		t.Fatal("reader toolbar not found")
	}

	action, ok := toolbar.Items[0].(*widget.ToolbarAction)
	if !ok {
		t.Fatal("first reader toolbar item is not an action")
	}
	replyAction = action

	showMessage(messageA)

	select {
	case <-store.saveStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message A save")
	}

	showMessage(messageB)
	close(store.releaseSave)

	select {
	case got := <-replied:
		if got.ID != messageB.ID {
			t.Fatalf(
				"Reply selected message %q, want %q",
				got.ID,
				messageB.ID,
			)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reply callback")
	}
}

func findReaderToolbar(
	object fyne.CanvasObject,
) *widget.Toolbar {
	if toolbar, ok := object.(*widget.Toolbar); ok {
		return toolbar
	}

	container, ok := object.(*fyne.Container)
	if !ok {
		return nil
	}

	for _, child := range container.Objects {
		if toolbar := findReaderToolbar(child); toolbar != nil {
			return toolbar
		}
	}

	return nil
}

func TestReaderMarksPendingUnreadSelectionRead(
	t *testing.T,
) {
	app := fyneTest.NewApp()
	window := app.NewWindow("test")

	base := mailbox.NewStore(t.TempDir())
	if err := base.Prepare(); err != nil {
		t.Fatal(err)
	}

	messageA := mailbox.Message{
		ID:      "message-a",
		Folder:  mailbox.FolderInbox,
		From:    "W2AAA",
		Subject: "Message A",
		Unread:  true,
	}
	messageB := mailbox.Message{
		ID:      "message-b",
		Folder:  mailbox.FolderInbox,
		From:    "W2BBB",
		Subject: "Message B",
		Unread:  true,
	}

	if err := base.Save(messageA); err != nil {
		t.Fatal(err)
	}
	if err := base.Save(messageB); err != nil {
		t.Fatal(err)
	}

	store := &blockingMailboxStore{
		Store:       base,
		saveStarted: make(chan struct{}),
		releaseSave: make(chan struct{}),
	}

	updated := make(chan mailbox.Message, 2)

	_, showMessage, _ := newReaderPane(
		window,
		store,
		folderMailView(mailbox.FolderInbox),
		nil,
		nil,
		nil,
		nil,
		func(msg mailbox.Message) {
			updated <- msg
		},
		nil,
	)

	showMessage(messageA)

	select {
	case <-store.saveStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message A save")
	}

	// B is selected while A is still being persisted.
	// Its initial automatic read attempt is deferred by mutating=true.
	showMessage(messageB)

	close(store.releaseSave)

	deadline := time.After(time.Second)
	sawBRead := false

	for !sawBRead {
		select {
		case msg := <-updated:
			if msg.ID == messageB.ID && !msg.Unread {
				sawBRead = true
			}
		case <-deadline:
			t.Fatal(
				"timed out waiting for pending message B to be marked read",
			)
		}
	}

	persistedA, err := base.Load(
		mailbox.FolderInbox,
		messageA.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if persistedA.Unread {
		t.Fatal("message A remained unread")
	}

	persistedB, err := base.Load(
		mailbox.FolderInbox,
		messageB.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if persistedB.Unread {
		t.Fatal("message B remained unread")
	}
}

func TestReaderExplicitReadToggleResumesPendingAutomaticRead(
	t *testing.T,
) {
	app := fyneTest.NewApp()
	window := app.NewWindow("test")

	base := mailbox.NewStore(t.TempDir())
	if err := base.Prepare(); err != nil {
		t.Fatal(err)
	}

	// A starts read so opening it does not trigger automatic read.
	// The explicit toolbar action will mark it unread and block.
	messageA := mailbox.Message{
		ID:      "message-a",
		Folder:  mailbox.FolderInbox,
		From:    "W2AAA",
		Subject: "Message A",
		Unread:  false,
	}
	messageB := mailbox.Message{
		ID:      "message-b",
		Folder:  mailbox.FolderInbox,
		From:    "W2BBB",
		Subject: "Message B",
		Unread:  true,
	}

	if err := base.Save(messageA); err != nil {
		t.Fatal(err)
	}
	if err := base.Save(messageB); err != nil {
		t.Fatal(err)
	}

	store := &blockingMailboxStore{
		Store:       base,
		saveStarted: make(chan struct{}),
		releaseSave: make(chan struct{}),
	}

	updated := make(chan mailbox.Message, 2)

	reader, showMessage, _ := newReaderPane(
		window,
		store,
		folderMailView(mailbox.FolderInbox),
		nil,
		nil,
		nil,
		nil,
		func(msg mailbox.Message) {
			updated <- msg
		},
		nil,
	)

	toolbar := findReaderToolbar(reader)
	if toolbar == nil {
		t.Fatal("reader toolbar not found")
	}

	if len(toolbar.Items) <= 5 {
		t.Fatalf(
			"reader toolbar has %d items, want at least 6",
			len(toolbar.Items),
		)
	}

	readAction, ok := toolbar.Items[5].(*widget.ToolbarAction)
	if !ok {
		t.Fatal("reader read item is not a toolbar action")
	}

	showMessage(messageA)

	// Explicitly toggle A from read to unread.
	readAction.OnActivated()

	select {
	case <-store.saveStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for explicit message A save")
	}

	// Select unread B while A's explicit mutation is still active.
	showMessage(messageB)

	close(store.releaseSave)

	deadline := time.After(time.Second)
	sawBRead := false

	for !sawBRead {
		select {
		case msg := <-updated:
			if msg.ID == messageB.ID && !msg.Unread {
				sawBRead = true
			}
		case <-deadline:
			t.Fatal(
				"timed out waiting for message B automatic read",
			)
		}
	}

	persistedA, err := base.Load(
		mailbox.FolderInbox,
		messageA.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !persistedA.Unread {
		t.Fatal("explicit message A toggle was not persisted")
	}

	persistedB, err := base.Load(
		mailbox.FolderInbox,
		messageB.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if persistedB.Unread {
		t.Fatal("pending message B remained unread")
	}
}
