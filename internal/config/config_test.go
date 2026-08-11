package config

import (
	"os"
	"path/filepath"
	"reflect"
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

func TestLoadSchemaV1AddsDefaultConnectionProfiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)

	data := []byte(`{
  "schema_version": 1,
  "callsign": "k2exe",
  "locator": "FN23va"
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

	want := DefaultConnectionProfiles()
	if !reflect.DeepEqual(cfg.ConnectionProfiles, want) {
		t.Fatalf(
			"ConnectionProfiles = %#v, want %#v",
			cfg.ConnectionProfiles,
			want,
		)
	}

	if cfg.Callsign != "K2EXE" {
		t.Fatalf("Callsign = %q, want K2EXE", cfg.Callsign)
	}
	if cfg.Locator != "FN23va" {
		t.Fatalf("Locator = %q, want FN23va", cfg.Locator)
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

func TestLoadRejectsDuplicateConnectionProfileIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)

	data := []byte(`{
  "schema_version": 2,
  "callsign": "K2EXE",
  "locator": "FN23va",
  "connection_profiles": [
    {
      "id": "duplicate",
      "name": "First",
      "network": "internet",
      "transport": "cms_telnet",
      "cms": {
        "mode": "test"
      }
    },
    {
      "id": "duplicate",
      "name": "Second",
      "network": "internet",
      "transport": "cms_telnet",
      "cms": {
        "mode": "production"
      }
    }
  ]
}
`)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("Load() expected duplicate profile ID error")
	}
}

func TestLoadRejectsInvalidDirectTCPProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)

	data := []byte(`{
  "schema_version": 2,
  "callsign": "K2EXE",
  "locator": "FN23va",
  "connection_profiles": [
    {
      "id": "mesh-peer",
      "name": "Mesh Peer",
      "network": "aredn",
      "transport": "direct_tcp",
      "tcp": {
        "address": "node.local.mesh",
        "target_call": "W2ABC"
      }
    }
  ]
}
`)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("Load() expected invalid TCP address error")
	}
}

func TestSaveRejectsInvalidProfileWithoutReplacingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)

	valid := Config{
		Callsign: "K2EXE",
		Locator:  "FN23va",
	}

	if err := Save(path, valid); err != nil {
		t.Fatalf("initial Save() error = %v", err)
	}

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() before invalid save error = %v", err)
	}

	invalid := valid.Normalized()
	invalid.ConnectionProfiles = []ConnectionProfile{
		{
			ID:        "bad",
			Name:      "Bad Direct TCP",
			Network:   NetworkAREDN,
			Transport: TransportDirectTCP,
			TCP: &TCPProfile{
				Address:    "missing-port.local.mesh",
				TargetCall: "W2ABC",
			},
		},
	}

	if err := Save(path, invalid); err == nil {
		t.Fatal("Save() expected invalid profile error")
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() after invalid save error = %v", err)
	}

	if string(after) != string(before) {
		t.Fatal("invalid Save() replaced the existing configuration")
	}
}

func TestConfigNormalizedNormalizesConnectionProfiles(t *testing.T) {
	cfg := Config{
		ConnectionProfiles: []ConnectionProfile{
			{
				ID:        " peer ",
				Name:      " Mesh Peer ",
				Network:   NetworkType(" AREDN "),
				Transport: TransportType(" DIRECT_TCP "),
				TCP: &TCPProfile{
					Address:    " node.local.mesh:8772 ",
					TargetCall: " w2abc ",
				},
			},
		},
	}

	got := cfg.Normalized()
	profile := got.ConnectionProfiles[0]

	if profile.Network != NetworkAREDN {
		t.Fatalf(
			"Network = %q, want %q",
			profile.Network,
			NetworkAREDN,
		)
	}
	if profile.Transport != TransportDirectTCP {
		t.Fatalf(
			"Transport = %q, want %q",
			profile.Transport,
			TransportDirectTCP,
		)
	}
	if profile.ID != "peer" {
		t.Fatalf("ID = %q, want peer", profile.ID)
	}
	if profile.TCP == nil ||
		profile.TCP.TargetCall != "W2ABC" {
		t.Fatalf("TCP = %#v, want normalized target W2ABC", profile.TCP)
	}
}
