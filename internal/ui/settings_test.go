package ui

import "testing"

func TestValidateIdentitySettings(t *testing.T) {
	tests := []struct {
		name     string
		callsign string
		locator  string
		wantErr  bool
	}{
		{
			name:     "valid",
			callsign: "K2EXE",
			locator:  "FN23va",
		},
		{
			name:     "missing callsign",
			callsign: "   ",
			locator:  "FN23va",
			wantErr:  true,
		},
		{
			name:     "missing locator",
			callsign: "K2EXE",
			locator:  "   ",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIdentitySettings(
				tt.callsign,
				tt.locator,
			)

			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"validateIdentitySettings() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}
