package ui

import (
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	fyneTest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

type blockingSelectionMutationStore struct {
	*mailbox.Store

	blockSaveID string
	blockMoveID string

	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (s *blockingSelectionMutationStore) block(
	id string,
	blockedID string,
) {
	if blockedID == "" || id != blockedID {
		return
	}

	s.once.Do(func() {
		close(s.started)
	})

	<-s.release
}

func (s *blockingSelectionMutationStore) Save(
	msg mailbox.Message,
) error {
	s.block(msg.ID, s.blockSaveID)
	return s.Store.Save(msg)
}

func (s *blockingSelectionMutationStore) Move(
	from mailbox.Folder,
	to mailbox.Folder,
	id string,
) error {
	s.block(id, s.blockMoveID)
	return s.Store.Move(from, to, id)
}

func readerToolbarActionAt(
	t *testing.T,
	reader fyne.CanvasObject,
	index int,
) *widget.ToolbarAction {
	t.Helper()

	toolbar := findReaderToolbar(reader)
	if toolbar == nil {
		t.Fatal("reader toolbar not found")
	}

	if index < 0 || index >= len(toolbar.Items) {
		t.Fatalf(
			"toolbar item %d unavailable; item count = %d",
			index,
			len(toolbar.Items),
		)
	}

	action, ok := toolbar.Items[index].(*widget.ToolbarAction)
	if !ok {
		t.Fatalf(
			"toolbar item %d is not an action",
			index,
		)
	}

	return action
}

func waitForSelectionMutation(
	t *testing.T,
	started <-chan struct{},
) {
	t.Helper()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for mailbox mutation")
	}
}

func expectReplyForMessage(
	t *testing.T,
	replied <-chan mailbox.Message,
	wantID string,
) {
	t.Helper()

	select {
	case got := <-replied:
		if got.ID != wantID {
			t.Fatalf(
				"Reply selected message %q, want %q",
				got.ID,
				wantID,
			)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reply callback")
	}
}

func TestReaderStarCompletionPreservesNewSelection(
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
		Subject: "Message A",
	}
	messageB := mailbox.Message{
		ID:      "message-b",
		Folder:  mailbox.FolderInbox,
		Subject: "Message B",
	}

	if err := base.Save(messageA); err != nil {
		t.Fatal(err)
	}
	if err := base.Save(messageB); err != nil {
		t.Fatal(err)
	}

	store := &blockingSelectionMutationStore{
		Store:       base,
		blockSaveID: messageA.ID,
		started:     make(chan struct{}),
		release:     make(chan struct{}),
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
		func(updated mailbox.Message) {
			if updated.ID == messageA.ID &&
				replyAction != nil {
				replyAction.OnActivated()
			}
		},
		nil,
	)

	replyAction = readerToolbarActionAt(t, reader, 0)
	starAction := readerToolbarActionAt(t, reader, 4)

	showMessage(messageA)
	starAction.OnActivated()

	waitForSelectionMutation(t, store.started)

	showMessage(messageB)
	close(store.release)

	expectReplyForMessage(t, replied, messageB.ID)

	persisted, err := base.Load(
		mailbox.FolderInbox,
		messageA.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !persisted.Starred {
		t.Fatal("message A was not persisted as starred")
	}
}

func TestReaderArchiveCompletionPreservesNewSelection(
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
		Subject: "Message A",
	}
	messageB := mailbox.Message{
		ID:      "message-b",
		Folder:  mailbox.FolderInbox,
		Subject: "Message B",
	}

	if err := base.Save(messageA); err != nil {
		t.Fatal(err)
	}
	if err := base.Save(messageB); err != nil {
		t.Fatal(err)
	}

	store := &blockingSelectionMutationStore{
		Store:       base,
		blockMoveID: messageA.ID,
		started:     make(chan struct{}),
		release:     make(chan struct{}),
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
		nil,
		func(removed mailbox.Message) {
			if removed.ID == messageA.ID &&
				replyAction != nil {
				replyAction.OnActivated()
			}
		},
	)

	replyAction = readerToolbarActionAt(t, reader, 0)
	archiveAction := readerToolbarActionAt(t, reader, 6)

	showMessage(messageA)
	archiveAction.OnActivated()

	waitForSelectionMutation(t, store.started)

	showMessage(messageB)
	close(store.release)

	expectReplyForMessage(t, replied, messageB.ID)

	if _, err := base.Load(
		mailbox.FolderArchive,
		messageA.ID,
	); err != nil {
		t.Fatalf("load archived message A: %v", err)
	}
}

func TestReaderTrashCompletionPreservesNewSelection(
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
		Subject: "Message A",
	}
	messageB := mailbox.Message{
		ID:      "message-b",
		Folder:  mailbox.FolderInbox,
		Subject: "Message B",
	}

	if err := base.Save(messageA); err != nil {
		t.Fatal(err)
	}
	if err := base.Save(messageB); err != nil {
		t.Fatal(err)
	}

	store := &blockingSelectionMutationStore{
		Store:       base,
		blockMoveID: messageA.ID,
		started:     make(chan struct{}),
		release:     make(chan struct{}),
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
		nil,
		func(removed mailbox.Message) {
			if removed.ID == messageA.ID &&
				replyAction != nil {
				replyAction.OnActivated()
			}
		},
	)

	replyAction = readerToolbarActionAt(t, reader, 0)
	deleteAction := readerToolbarActionAt(t, reader, 8)

	showMessage(messageA)
	deleteAction.OnActivated()

	waitForSelectionMutation(t, store.started)

	showMessage(messageB)
	close(store.release)

	expectReplyForMessage(t, replied, messageB.ID)

	if _, err := base.Load(
		mailbox.FolderTrash,
		messageA.ID,
	); err != nil {
		t.Fatalf("load trashed message A: %v", err)
	}
}

func TestReaderStarredUnstarCompletionPreservesNewSelection(
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
		Subject: "Message A",
		Starred: true,
	}
	messageB := mailbox.Message{
		ID:      "message-b",
		Folder:  mailbox.FolderInbox,
		Subject: "Message B",
		Starred: true,
	}

	if err := base.Save(messageA); err != nil {
		t.Fatal(err)
	}
	if err := base.Save(messageB); err != nil {
		t.Fatal(err)
	}

	store := &blockingSelectionMutationStore{
		Store:       base,
		blockSaveID: messageA.ID,
		started:     make(chan struct{}),
		release:     make(chan struct{}),
	}

	replied := make(chan mailbox.Message, 1)
	removed := make(chan mailbox.Message, 1)
	var replyAction *widget.ToolbarAction

	reader, showMessage, _ := newReaderPane(
		window,
		store,
		starredMailView(),
		nil,
		func(msg mailbox.Message, _ bool) {
			replied <- msg
		},
		nil,
		nil,
		nil,
		func(msg mailbox.Message) {
			removed <- msg

			if replyAction != nil {
				replyAction.OnActivated()
			}
		},
	)

	replyAction = readerToolbarActionAt(t, reader, 0)
	starAction := readerToolbarActionAt(t, reader, 4)

	showMessage(messageA)
	starAction.OnActivated()

	waitForSelectionMutation(t, store.started)

	showMessage(messageB)
	close(store.release)

	select {
	case got := <-removed:
		if got.ID != messageA.ID {
			t.Fatalf(
				"removed message = %q, want %q",
				got.ID,
				messageA.ID,
			)
		}
		if got.Starred {
			t.Fatal("removed message remained starred")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for unstar removal callback")
	}

	expectReplyForMessage(t, replied, messageB.ID)

	persisted, err := base.Load(
		mailbox.FolderInbox,
		messageA.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Starred {
		t.Fatal("message A remained starred on disk")
	}
}
