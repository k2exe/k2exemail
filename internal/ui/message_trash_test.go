package ui

import (
	"errors"
	"testing"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

type messageTrashStoreStub struct {
	moveFrom  mailbox.Folder
	moveTo    mailbox.Folder
	moveID    string
	deleteAt  mailbox.Folder
	deleteID  string
	moveErr   error
	deleteErr error
}

func (s *messageTrashStoreStub) Move(
	from,
	to mailbox.Folder,
	id string,
) error {
	s.moveFrom = from
	s.moveTo = to
	s.moveID = id
	return s.moveErr
}

func (s *messageTrashStoreStub) Delete(
	folder mailbox.Folder,
	id string,
) error {
	s.deleteAt = folder
	s.deleteID = id
	return s.deleteErr
}

func TestTrashOrDeleteMessageMovesToTrash(t *testing.T) {
	store := &messageTrashStoreStub{}

	err := trashOrDeleteMessage(
		store,
		mailbox.FolderOutbox,
		"message-1",
	)
	if err != nil {
		t.Fatalf(
			"trashOrDeleteMessage() error = %v",
			err,
		)
	}

	if store.moveFrom != mailbox.FolderOutbox ||
		store.moveTo != mailbox.FolderTrash ||
		store.moveID != "message-1" {
		t.Fatalf(
			"Move() = %q -> %q, %q",
			store.moveFrom,
			store.moveTo,
			store.moveID,
		)
	}

	if store.deleteID != "" {
		t.Fatalf(
			"Delete() unexpectedly called for %q",
			store.deleteID,
		)
	}
}

func TestTrashOrDeleteMessageDeletesFromTrash(t *testing.T) {
	store := &messageTrashStoreStub{}

	err := trashOrDeleteMessage(
		store,
		mailbox.FolderTrash,
		"message-2",
	)
	if err != nil {
		t.Fatalf(
			"trashOrDeleteMessage() error = %v",
			err,
		)
	}

	if store.deleteAt != mailbox.FolderTrash ||
		store.deleteID != "message-2" {
		t.Fatalf(
			"Delete() = %q, %q",
			store.deleteAt,
			store.deleteID,
		)
	}

	if store.moveID != "" {
		t.Fatalf(
			"Move() unexpectedly called for %q",
			store.moveID,
		)
	}
}

func TestTrashOrDeleteMessageWrapsMoveError(t *testing.T) {
	store := &messageTrashStoreStub{
		moveErr: errors.New("move failed"),
	}

	err := trashOrDeleteMessage(
		store,
		mailbox.FolderInbox,
		"message-3",
	)
	if err == nil {
		t.Fatal(
			"trashOrDeleteMessage() expected move error",
		)
	}
}

func TestTrashOrDeleteMessageWrapsDeleteError(t *testing.T) {
	store := &messageTrashStoreStub{
		deleteErr: errors.New("delete failed"),
	}

	err := trashOrDeleteMessage(
		store,
		mailbox.FolderTrash,
		"message-4",
	)
	if err == nil {
		t.Fatal(
			"trashOrDeleteMessage() expected delete error",
		)
	}
}

func TestMailboxActivityGateExcludesCMSAndMutation(
	t *testing.T,
) {
	gate := &mailboxActivityGate{}

	if !gate.beginMutation() {
		t.Fatal("beginMutation() unexpectedly failed")
	}

	if gate.beginCMS() {
		t.Fatal(
			"beginCMS() succeeded during mailbox mutation",
		)
	}

	gate.endMutation()

	if !gate.beginCMS() {
		t.Fatal("beginCMS() unexpectedly failed")
	}

	if gate.beginMutation() {
		t.Fatal(
			"beginMutation() succeeded during CMS session",
		)
	}

	gate.endCMS()

	if !gate.beginMutation() {
		t.Fatal(
			"beginMutation() failed after CMS session ended",
		)
	}

	gate.endMutation()
}
