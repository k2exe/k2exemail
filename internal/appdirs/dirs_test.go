package appdirs

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestResolveDataDir(t *testing.T) {
	tests := []struct {
		name string
		goos string
		env  map[string]string
		home string
		want string
	}{
		{
			name: "windows local app data",
			goos: "windows",
			env: map[string]string{
				"LOCALAPPDATA": "local-data",
			},
			want: filepath.Join("local-data", "K2EXEmail"),
		},
		{
			name: "mac application support",
			goos: "darwin",
			home: "home",
			want: filepath.Join(
				"home",
				"Library",
				"Application Support",
				"K2EXEmail",
			),
		},
		{
			name: "linux default",
			goos: "linux",
			home: "home",
			want: filepath.Join(
				"home",
				".local",
				"share",
				"k2exemail",
			),
		},
		{
			name: "linux xdg data",
			goos: "linux",
			env: map[string]string{
				"XDG_DATA_HOME": "/data",
			},
			home: "home",
			want: filepath.Join("/data", "k2exemail"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(key string) string {
				return tt.env[key]
			}

			homeDir := func() (string, error) {
				if tt.home == "" {
					return "", errors.New("home unavailable")
				}
				return tt.home, nil
			}

			got, err := resolveDataDir(tt.goos, getenv, homeDir)
			if err != nil {
				t.Fatalf("resolveDataDir() error = %v", err)
			}

			if got != tt.want {
				t.Fatalf("resolveDataDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDefaultDataOverrideMustBeAbsolute(t *testing.T) {
	t.Setenv(dataOverride, "relative/path")

	_, err := Default()
	if err == nil {
		t.Fatal("Default() expected error for relative data override")
	}
}
