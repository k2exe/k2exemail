package ui

import (
	"testing"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

func TestSameMessageIdentity(t *testing.T) {
	tests := []struct {
		name string
		a    mailbox.Message
		b    mailbox.Message
		want bool
	}{
		{
			name: "same physical message",
			a: mailbox.Message{
				ID:     "message-1",
				Folder: mailbox.FolderInbox,
			},
			b: mailbox.Message{
				ID:     "message-1",
				Folder: mailbox.FolderInbox,
			},
			want: true,
		},
		{
			name: "different message ID",
			a: mailbox.Message{
				ID:     "message-1",
				Folder: mailbox.FolderInbox,
			},
			b: mailbox.Message{
				ID:     "message-2",
				Folder: mailbox.FolderInbox,
			},
			want: false,
		},
		{
			name: "same ID different folder",
			a: mailbox.Message{
				ID:     "message-1",
				Folder: mailbox.FolderInbox,
			},
			b: mailbox.Message{
				ID:     "message-1",
				Folder: mailbox.FolderArchive,
			},
			want: false,
		},
		{
			name: "empty identity",
			a: mailbox.Message{
				Folder: mailbox.FolderInbox,
			},
			b: mailbox.Message{
				Folder: mailbox.FolderInbox,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sameMessageIdentity(
				tt.a,
				tt.b,
			); got != tt.want {
				t.Fatalf(
					"sameMessageIdentity() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}
