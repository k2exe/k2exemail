package ui

import "testing"

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
