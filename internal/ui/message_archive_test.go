package ui

import (
	"errors"
	"testing"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

type messageArchiveStoreStub struct {
	from mailbox.Folder
	to   mailbox.Folder
	id   string
	err  error
}

func (s *messageArchiveStoreStub) Move(
	from,
	to mailbox.Folder,
	id string,
) error {
	s.from = from
	s.to = to
	s.id = id
	return s.err
}

func TestArchiveOrRestoreMessage(t *testing.T) {
	tests := []struct {
		name     string
		folder   mailbox.Folder
		wantFrom mailbox.Folder
		wantTo   mailbox.Folder
	}{
		{
			name:     "archive inbox message",
			folder:   mailbox.FolderInbox,
			wantFrom: mailbox.FolderInbox,
			wantTo:   mailbox.FolderArchive,
		},
		{
			name:     "restore archived message",
			folder:   mailbox.FolderArchive,
			wantFrom: mailbox.FolderArchive,
			wantTo:   mailbox.FolderInbox,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &messageArchiveStoreStub{}

			err := archiveOrRestoreMessage(
				store,
				tt.folder,
				"message-1",
			)
			if err != nil {
				t.Fatalf(
					"archiveOrRestoreMessage() error = %v",
					err,
				)
			}

			if store.from != tt.wantFrom ||
				store.to != tt.wantTo ||
				store.id != "message-1" {
				t.Fatalf(
					"Move() = %q -> %q, %q; want %q -> %q, %q",
					store.from,
					store.to,
					store.id,
					tt.wantFrom,
					tt.wantTo,
					"message-1",
				)
			}
		})
	}
}

func TestArchiveOrRestoreMessageRejectsUnsupportedFolder(
	t *testing.T,
) {
	store := &messageArchiveStoreStub{}

	err := archiveOrRestoreMessage(
		store,
		mailbox.FolderSent,
		"message-2",
	)
	if err == nil {
		t.Fatal(
			"archiveOrRestoreMessage() expected unsupported folder error",
		)
	}

	if store.id != "" {
		t.Fatalf(
			"Move() unexpectedly called for %q",
			store.id,
		)
	}
}

func TestArchiveOrRestoreMessageWrapsMoveError(t *testing.T) {
	moveErr := errors.New("move failed")
	store := &messageArchiveStoreStub{
		err: moveErr,
	}

	err := archiveOrRestoreMessage(
		store,
		mailbox.FolderInbox,
		"message-3",
	)
	if err == nil {
		t.Fatal(
			"archiveOrRestoreMessage() expected move error",
		)
	}

	if !errors.Is(err, moveErr) {
		t.Fatalf(
			"archiveOrRestoreMessage() error = %v, want wrapped %v",
			err,
			moveErr,
		)
	}
}
