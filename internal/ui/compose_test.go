package ui

import (
	"testing"

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
