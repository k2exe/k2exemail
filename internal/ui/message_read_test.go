package ui

import (
	"errors"
	"testing"
	"time"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

type messageReadStoreStub struct {
	saved mailbox.Message
	calls int
	err   error
}

func (s *messageReadStoreStub) Save(
	msg mailbox.Message,
) error {
	s.calls++
	s.saved = msg
	return s.err
}

func TestSetMessageUnread(t *testing.T) {
	tests := []struct {
		name    string
		initial bool
		want    bool
	}{
		{
			name:    "mark unread",
			initial: false,
			want:    true,
		},
		{
			name:    "mark read",
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
				Unread:    tt.initial,
				UpdatedAt: updatedAt,
			}

			store := &messageReadStoreStub{}

			got, err := setMessageUnread(
				store,
				msg,
				tt.want,
			)
			if err != nil {
				t.Fatalf(
					"setMessageUnread() error = %v",
					err,
				)
			}

			if store.calls != 1 {
				t.Fatalf(
					"Save() calls = %d, want 1",
					store.calls,
				)
			}

			if got.Unread != tt.want {
				t.Fatalf(
					"Unread = %v, want %v",
					got.Unread,
					tt.want,
				)
			}

			if store.saved.Unread != tt.want {
				t.Fatalf(
					"saved Unread = %v, want %v",
					store.saved.Unread,
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

func TestSetMessageUnreadSkipsUnchangedState(
	t *testing.T,
) {
	msg := mailbox.Message{
		ID:     "message-2",
		Folder: mailbox.FolderInbox,
		Unread: true,
	}

	store := &messageReadStoreStub{}

	got, err := setMessageUnread(
		store,
		msg,
		true,
	)
	if err != nil {
		t.Fatalf(
			"setMessageUnread() error = %v",
			err,
		)
	}

	if store.calls != 0 {
		t.Fatalf(
			"Save() calls = %d, want 0",
			store.calls,
		)
	}

	if !got.Unread {
		t.Fatal("Unread = false, want true")
	}
}

func TestSetMessageUnreadWrapsSaveError(
	t *testing.T,
) {
	saveErr := errors.New("save failed")

	msg := mailbox.Message{
		ID:     "message-3",
		Folder: mailbox.FolderArchive,
		Unread: true,
	}

	store := &messageReadStoreStub{
		err: saveErr,
	}

	got, err := setMessageUnread(
		store,
		msg,
		false,
	)
	if err == nil {
		t.Fatal(
			"setMessageUnread() expected save error",
		)
	}

	if !errors.Is(err, saveErr) {
		t.Fatalf(
			"setMessageUnread() error = %v, want wrapped %v",
			err,
			saveErr,
		)
	}

	if got.Unread != msg.Unread {
		t.Fatalf(
			"returned Unread = %v, want original %v",
			got.Unread,
			msg.Unread,
		)
	}
}
