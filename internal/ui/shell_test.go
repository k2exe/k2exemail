package ui

import (
	"testing"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

func TestMessageSnippet(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "single line",
			body: "Hello from K2EXEmail",
			want: "Hello from K2EXEmail",
		},
		{
			name: "first non-empty line",
			body: "\n\n  First line  \nSecond line",
			want: "First line",
		},
		{
			name: "windows line endings",
			body: "\r\nFirst line\r\nSecond line",
			want: "First line",
		},
		{
			name: "empty body",
			body: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := messageSnippet(tt.body)
			if got != tt.want {
				t.Fatalf(
					"messageSnippet(%q) = %q, want %q",
					tt.body,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestMessageListPrimary(t *testing.T) {
	outgoing := mailbox.Message{
		To: []string{"K2EXE", "KR2SSY"},
	}

	if got := messageListPrimary(
		mailbox.FolderOutbox,
		outgoing,
	); got != "To: K2EXE, KR2SSY" {
		t.Fatalf("outbox primary = %q", got)
	}

	incoming := mailbox.Message{
		From: "W2ABC",
	}

	if got := messageListPrimary(
		mailbox.FolderInbox,
		incoming,
	); got != "W2ABC" {
		t.Fatalf("inbox primary = %q", got)
	}
}

func TestMessageListPrimaryForStarredViewUsesMessageFolder(
	t *testing.T,
) {
	sent := mailbox.Message{
		Folder: mailbox.FolderSent,
		To:     []string{"K2EXE", "W2ABC"},
	}

	if got := messageListPrimaryForView(
		starredMailView(),
		sent,
	); got != "To: K2EXE, W2ABC" {
		t.Fatalf(
			"starred Sent primary = %q, want recipient",
			got,
		)
	}

	inbox := mailbox.Message{
		Folder: mailbox.FolderInbox,
		From:   "W2XYZ",
	}

	if got := messageListPrimaryForView(
		starredMailView(),
		inbox,
	); got != "W2XYZ" {
		t.Fatalf(
			"starred Inbox primary = %q, want sender",
			got,
		)
	}
}
