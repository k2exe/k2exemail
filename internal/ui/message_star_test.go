package ui

import (
	"errors"
	"testing"
	"time"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

type messageStarStoreStub struct {
	saved mailbox.Message
	calls int
	err   error
}

func (s *messageStarStoreStub) Save(msg mailbox.Message) error {
	s.calls++
	s.saved = msg
	return s.err
}

func TestSetMessageStarred(t *testing.T) {
	tests := []struct {
		name    string
		initial bool
		want    bool
	}{
		{
			name:    "star message",
			initial: false,
			want:    true,
		},
		{
			name:    "unstar message",
			initial: true,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updatedAt := time.Date(
				2026, 8, 10, 12, 0, 0, 0, time.UTC,
			)

			msg := mailbox.Message{
				ID:        "message-1",
				Folder:    mailbox.FolderInbox,
				Subject:   "Test",
				Starred:   tt.initial,
				UpdatedAt: updatedAt,
			}

			store := &messageStarStoreStub{}

			got, err := setMessageStarred(
				store,
				msg,
				tt.want,
			)
			if err != nil {
				t.Fatalf("setMessageStarred() error = %v", err)
			}

			if store.calls != 1 {
				t.Fatalf(
					"Save() calls = %d, want 1",
					store.calls,
				)
			}

			if got.Starred != tt.want {
				t.Fatalf(
					"Starred = %v, want %v",
					got.Starred,
					tt.want,
				)
			}

			if store.saved.Starred != tt.want {
				t.Fatalf(
					"saved Starred = %v, want %v",
					store.saved.Starred,
					tt.want,
				)
			}

			if !got.UpdatedAt.Equal(updatedAt) {
				t.Fatalf(
					"UpdatedAt = %v, want unchanged %v",
					got.UpdatedAt,
					updatedAt,
				)
			}
		})
	}
}

func TestSetMessageStarredSkipsUnchangedState(t *testing.T) {
	msg := mailbox.Message{
		ID:      "message-2",
		Folder:  mailbox.FolderSent,
		Starred: true,
	}

	store := &messageStarStoreStub{}

	got, err := setMessageStarred(store, msg, true)
	if err != nil {
		t.Fatalf("setMessageStarred() error = %v", err)
	}

	if store.calls != 0 {
		t.Fatalf(
			"Save() calls = %d, want 0",
			store.calls,
		)
	}

	if !got.Starred {
		t.Fatal("Starred = false, want true")
	}
}

func TestSetMessageStarredWrapsSaveError(t *testing.T) {
	saveErr := errors.New("save failed")

	msg := mailbox.Message{
		ID:      "message-3",
		Folder:  mailbox.FolderArchive,
		Starred: false,
	}

	store := &messageStarStoreStub{
		err: saveErr,
	}

	got, err := setMessageStarred(store, msg, true)
	if err == nil {
		t.Fatal("setMessageStarred() expected save error")
	}

	if !errors.Is(err, saveErr) {
		t.Fatalf(
			"setMessageStarred() error = %v, want wrapped %v",
			err,
			saveErr,
		)
	}

	if got.Starred != msg.Starred {
		t.Fatalf(
			"returned Starred = %v, want original %v",
			got.Starred,
			msg.Starred,
		)
	}
}
