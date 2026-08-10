package ui

import (
	"testing"
	"time"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

func TestSplitRecipients(t *testing.T) {
	got := splitRecipients("W2ABC, W3XYZ; SMTP:test@example.com")

	want := []string{
		"W2ABC",
		"W3XYZ",
		"SMTP:test@example.com",
	}

	if len(got) != len(want) {
		t.Fatalf("recipient count = %d, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf(
				"recipient %d = %q, want %q",
				i,
				got[i],
				want[i],
			)
		}
	}
}

func TestValidateQueueMessage(t *testing.T) {
	valid := mailbox.Message{
		To:      []string{"W2ABC"},
		Subject: "Test",
		Body:    "Hello",
	}

	if err := validateQueueMessage(valid); err != nil {
		t.Fatalf("valid message rejected: %v", err)
	}

	tests := []struct {
		name string
		msg  mailbox.Message
	}{
		{
			name: "missing recipient",
			msg: mailbox.Message{
				Subject: "Test",
				Body:    "Hello",
			},
		},
		{
			name: "missing subject",
			msg: mailbox.Message{
				To:   []string{"W2ABC"},
				Body: "Hello",
			},
		},
		{
			name: "missing body",
			msg: mailbox.Message{
				To:      []string{"W2ABC"},
				Subject: "Test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateQueueMessage(tt.msg); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestComposeSnapshotPreservesDraftIdentity(t *testing.T) {
	created := time.Date(
		2026, 8, 10, 1, 0, 0, 0, time.UTC,
	)
	updated := created.Add(time.Minute)

	base := mailbox.Message{
		ID:        "draft-1",
		Folder:    mailbox.FolderDrafts,
		CreatedAt: created,
		Attachments: []mailbox.Attachment{
			{
				ID:   "attachment-1",
				Name: "field-notes.txt",
			},
		},
	}

	got := composeSnapshot(
		base,
		"k2exe",
		"W2ABC; W3XYZ",
		"KR2SSY",
		" Updated subject ",
		"Updated body",
		updated,
	)

	if got.From != "K2EXE" {
		t.Fatalf("From = %q, want K2EXE", got.From)
	}

	if got.ID != base.ID {
		t.Fatalf("ID = %q, want %q", got.ID, base.ID)
	}

	if !got.CreatedAt.Equal(created) {
		t.Fatalf("CreatedAt changed")
	}

	if !got.UpdatedAt.Equal(updated) {
		t.Fatalf("UpdatedAt = %v, want %v", got.UpdatedAt, updated)
	}

	if len(got.Attachments) != 1 ||
		got.Attachments[0].ID != "attachment-1" {
		t.Fatal("existing attachment was not preserved")
	}

	if len(got.To) != 2 ||
		got.To[0] != "W2ABC" ||
		got.To[1] != "W3XYZ" {
		t.Fatalf("To = %#v", got.To)
	}

	if got.Subject != "Updated subject" {
		t.Fatalf("Subject = %q", got.Subject)
	}

	if got.Body != "Updated body" {
		t.Fatalf("Body = %q", got.Body)
	}
}
