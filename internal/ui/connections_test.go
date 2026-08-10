package ui

import "testing"

func TestConnectionResultText(t *testing.T) {
	tests := []struct {
		name     string
		sent     int
		received int
		want     string
	}{
		{
			name:     "empty exchange",
			sent:     0,
			received: 0,
			want:     "Complete - 0 sent, 0 received",
		},
		{
			name:     "mail exchanged",
			sent:     2,
			received: 3,
			want:     "Complete - 2 sent, 3 received",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := connectionResultText(
				tt.sent,
				tt.received,
			)

			if got != tt.want {
				t.Fatalf(
					"connectionResultText() = %q, want %q",
					got,
					tt.want,
				)
			}
		})
	}
}
