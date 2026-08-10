package ui

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

func TestReplyRecipients(t *testing.T) {
	tests := []struct {
		name     string
		original mailbox.Message
		callsign string
		replyAll bool
		wantTo   []string
		wantCc   []string
	}{
		{
			name: "reply to inbound sender",
			original: mailbox.Message{
				From: "W2ABC",
				To:   []string{"K2EXE"},
				Cc:   []string{"K3XYZ"},
			},
			callsign: "K2EXE",
			wantTo:   []string{"W2ABC"},
		},
		{
			name: "reply all inbound",
			original: mailbox.Message{
				From: "W2ABC",
				To: []string{
					"K2EXE",
					"W3DEF",
				},
				Cc: []string{"K3XYZ"},
			},
			callsign: "K2EXE",
			replyAll: true,
			wantTo:   []string{"W2ABC"},
			wantCc: []string{
				"W3DEF",
				"K3XYZ",
			},
		},
		{
			name: "reply from sent targets recipient",
			original: mailbox.Message{
				From: "K2EXE",
				To: []string{
					"W2ABC",
					"W3DEF",
				},
			},
			callsign: "k2exe",
			wantTo:   []string{"W2ABC"},
		},
		{
			name: "reply all from sent preserves to and cc",
			original: mailbox.Message{
				From: "K2EXE",
				To: []string{
					"W2ABC",
					"W3DEF",
				},
				Cc: []string{"K4GHI"},
			},
			callsign: "K2EXE",
			replyAll: true,
			wantTo: []string{
				"W2ABC",
				"W3DEF",
			},
			wantCc: []string{"K4GHI"},
		},
		{
			name: "reply all deduplicates case insensitively",
			original: mailbox.Message{
				From: "W2ABC",
				To: []string{
					"K2EXE",
					"W3DEF",
					"w3def",
				},
				Cc: []string{
					"W3DEF",
					"K4GHI",
				},
			},
			callsign: "K2EXE",
			replyAll: true,
			wantTo:   []string{"W2ABC"},
			wantCc: []string{
				"W3DEF",
				"K4GHI",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTo, gotCc := replyRecipients(
				tt.original,
				tt.callsign,
				tt.replyAll,
			)

			if !reflect.DeepEqual(
				gotTo,
				tt.wantTo,
			) {
				t.Fatalf(
					"To = %#v, want %#v",
					gotTo,
					tt.wantTo,
				)
			}

			if !reflect.DeepEqual(
				gotCc,
				tt.wantCc,
			) {
				t.Fatalf(
					"Cc = %#v, want %#v",
					gotCc,
					tt.wantCc,
				)
			}
		})
	}
}

func TestReplySubject(t *testing.T) {
	tests := []struct {
		subject string
		want    string
	}{
		{"Test message", "Re: Test message"},
		{"Re: Test message", "Re: Test message"},
		{"RE: Test message", "RE: Test message"},
		{"  test  ", "Re: test"},
		{"", "Re:"},
	}

	for _, tt := range tests {
		if got := replySubject(tt.subject); got != tt.want {
			t.Fatalf(
				"replySubject(%q) = %q, want %q",
				tt.subject,
				got,
				tt.want,
			)
		}
	}
}

func TestReplyBodyQuotesOriginal(t *testing.T) {
	original := mailbox.Message{
		From: "W2ABC",
		Body: "first\r\n\r\nsecond",
		CreatedAt: time.Date(
			2026,
			time.August,
			10,
			15,
			4,
			0,
			0,
			time.UTC,
		),
	}

	got := replyBody(original)

	want := "\n\n" +
		"On Mon, 10 Aug 2026 15:04 UTC, W2ABC wrote:\n" +
		"> first\n" +
		">\n" +
		"> second\n"

	if got != want {
		t.Fatalf(
			"replyBody() = %q, want %q",
			got,
			want,
		)
	}
}

func TestPrepareReplyDraftDoesNotCopyAttachments(
	t *testing.T,
) {
	draft := mailbox.Message{
		ID:     "draft-1",
		Folder: mailbox.FolderDrafts,
	}

	original := mailbox.Message{
		ID:      "message-1",
		From:    "W2ABC",
		To:      []string{"K2EXE"},
		Subject: "Attachment test",
		Body:    "hello",
		Attachments: []mailbox.Attachment{
			{
				ID:   "attachment-1",
				Name: "test.txt",
				Size: 10,
			},
		},
		P2POnly: true,
	}

	got, err := prepareReplyDraft(
		draft,
		original,
		"K2EXE",
		false,
	)
	if err != nil {
		t.Fatalf(
			"prepareReplyDraft() error = %v",
			err,
		)
	}

	if len(got.Attachments) != 0 {
		t.Fatalf(
			"reply copied %d attachments",
			len(got.Attachments),
		)
	}
	if got.P2POnly {
		t.Fatal("reply inherited P2POnly")
	}
	if got.WinlinkMID != "" {
		t.Fatalf(
			"reply inherited MID %q",
			got.WinlinkMID,
		)
	}
	if !strings.HasPrefix(got.Body, "\n\n") {
		t.Fatalf(
			"reply body does not leave compose space: %q",
			got.Body,
		)
	}
}

func TestPrepareReplyDraftRejectsMissingRecipient(
	t *testing.T,
) {
	_, err := prepareReplyDraft(
		mailbox.Message{
			ID:     "draft-1",
			Folder: mailbox.FolderDrafts,
		},
		mailbox.Message{
			ID:   "message-1",
			From: "K2EXE",
			To:   []string{"K2EXE"},
		},
		"K2EXE",
		false,
	)
	if err == nil {
		t.Fatal(
			"prepareReplyDraft() expected recipient error",
		)
	}
}
