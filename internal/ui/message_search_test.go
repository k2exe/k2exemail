package ui

import (
	"reflect"
	"testing"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

func TestFilterMessages(t *testing.T) {
	messages := []mailbox.Message{
		{
			ID:      "one",
			From:    "W2ABC",
			To:      []string{"K2EXE"},
			Cc:      []string{"W3DEF"},
			Subject: "Weather report",
			Body:    "Sunny and warm today.",
		},
		{
			ID:      "two",
			From:    "K4XYZ",
			To:      []string{"N2TEST"},
			Subject: "AREDN meeting",
			Body:    "Mesh network planning notes.",
		},
		{
			ID:      "three",
			From:    "N3AAA",
			To:      []string{"K2EXE"},
			Subject: "Winlink test",
			Body:    "Attachment forwarding check.",
		},
	}

	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{
			name:  "empty returns all",
			query: "",
			want:  []string{"one", "two", "three"},
		},
		{
			name:  "whitespace returns all",
			query: "   ",
			want:  []string{"one", "two", "three"},
		},
		{
			name:  "matches sender case insensitively",
			query: "w2abc",
			want:  []string{"one"},
		},
		{
			name:  "matches to",
			query: "n2test",
			want:  []string{"two"},
		},
		{
			name:  "matches cc",
			query: "w3def",
			want:  []string{"one"},
		},
		{
			name:  "matches subject",
			query: "aredn",
			want:  []string{"two"},
		},
		{
			name:  "matches body",
			query: "forwarding",
			want:  []string{"three"},
		},
		{
			name:  "matches multiple messages",
			query: "k2exe",
			want:  []string{"one", "three"},
		},
		{
			name:  "no matches",
			query: "does-not-exist",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterMessages(
				messages,
				tt.query,
			)

			gotIDs := make(
				[]string,
				0,
				len(got),
			)

			for _, msg := range got {
				gotIDs = append(
					gotIDs,
					msg.ID,
				)
			}

			if len(gotIDs) == 0 {
				gotIDs = nil
			}

			if !reflect.DeepEqual(
				gotIDs,
				tt.want,
			) {
				t.Fatalf(
					"IDs = %#v, want %#v",
					gotIDs,
					tt.want,
				)
			}
		})
	}
}
