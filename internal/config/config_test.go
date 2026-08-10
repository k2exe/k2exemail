package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingReturnsDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf(
			"SchemaVersion = %d, want %d",
			cfg.SchemaVersion,
			CurrentSchemaVersion,
		)
	}

	if cfg.Callsign != "" || cfg.Locator != "" {
		t.Fatalf("unexpected identity in default config: %#v", cfg)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", FileName)

	input := Config{
		Callsign: "  k2exe  ",
		Locator:  " FN13 ",
	}

	if err := Save(path, input); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.Callsign != "K2EXE" {
		t.Fatalf("Callsign = %q, want K2EXE", got.Callsign)
	}

	if got.Locator != "FN13" {
		t.Fatalf("Locator = %q, want FN13", got.Locator)
	}

	if got.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf(
			"SchemaVersion = %d, want %d",
			got.SchemaVersion,
			CurrentSchemaVersion,
		)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	text := string(data)

	if strings.Contains(strings.ToLower(text), "password") {
		t.Fatalf("configuration unexpectedly contains password field: %s", text)
	}
}

func TestLoadEarlyUnversionedConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)

	data := []byte(`{
  "callsign": "k2exe",
  "locator": "FN13"
}
`)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf(
			"SchemaVersion = %d, want %d",
			cfg.SchemaVersion,
			CurrentSchemaVersion,
		)
	}

	if cfg.Callsign != "K2EXE" {
		t.Fatalf("Callsign = %q, want K2EXE", cfg.Callsign)
	}
}

func TestLoadRejectsFutureSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)

	data := []byte(`{
  "schema_version": 99,
  "callsign": "K2EXE"
}
`)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("Load() expected unsupported schema error")
	}
}

func TestPath(t *testing.T) {
	got := Path(filepath.Join("some", "config", "dir"))
	want := filepath.Join("some", "config", "dir", FileName)

	if got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
}

func TestIdentityReady(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want bool
	}{
		{
			name: "complete",
			cfg: Config{
				Callsign: "K2EXE",
				Locator:  "FN13",
			},
			want: true,
		},
		{
			name: "missing callsign",
			cfg: Config{
				Locator: "FN13",
			},
			want: false,
		},
		{
			name: "missing locator",
			cfg: Config{
				Callsign: "K2EXE",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.IdentityReady(); got != tt.want {
				t.Fatalf("IdentityReady() = %v, want %v", got, tt.want)
			}
		})
	}
}
