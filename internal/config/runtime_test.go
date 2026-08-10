package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestRuntimeUpdateIdentityPersistsAndNormalizes(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)

	runtime := NewRuntime(
		path,
		Default(),
	)

	updated, err := runtime.UpdateIdentity(
		" k2exe ",
		" FN23va ",
	)
	if err != nil {
		t.Fatalf("UpdateIdentity() error = %v", err)
	}

	if updated.Callsign != "K2EXE" {
		t.Fatalf(
			"Callsign = %q, want K2EXE",
			updated.Callsign,
		)
	}
	if updated.Locator != "FN23va" {
		t.Fatalf(
			"Locator = %q, want FN23va",
			updated.Locator,
		)
	}

	current := runtime.Current()
	if current != updated {
		t.Fatalf(
			"Current() = %#v, want %#v",
			current,
			updated,
		)
	}

	persisted, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if persisted != updated {
		t.Fatalf(
			"persisted config = %#v, want %#v",
			persisted,
			updated,
		)
	}
}

func TestRuntimeUpdateIdentityFailureKeepsCurrentConfig(t *testing.T) {
	root := t.TempDir()

	blockedParent := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(
		blockedParent,
		[]byte("file"),
		0o600,
	); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	initial := Config{
		SchemaVersion: CurrentSchemaVersion,
		Callsign:      "K2EXE",
		Locator:       "FN23va",
	}

	runtime := NewRuntime(
		filepath.Join(blockedParent, FileName),
		initial,
	)

	_, err := runtime.UpdateIdentity(
		"W2ABC",
		"FN20",
	)
	if err == nil {
		t.Fatal("UpdateIdentity() expected save error")
	}

	current := runtime.Current()
	if current != initial {
		t.Fatalf(
			"Current() = %#v after failed save, want %#v",
			current,
			initial,
		)
	}
}

func TestRuntimeConcurrentReads(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)

	runtime := NewRuntime(
		path,
		Config{
			SchemaVersion: CurrentSchemaVersion,
			Callsign:      "K2EXE",
			Locator:       "FN23va",
		},
	)

	var readers sync.WaitGroup

	for range 8 {
		readers.Add(1)

		go func() {
			defer readers.Done()

			for range 100 {
				cfg := runtime.Current()
				if cfg.Callsign == "" {
					t.Error("Current() returned empty callsign")
					return
				}
			}
		}()
	}

	if _, err := runtime.UpdateIdentity(
		"K2EXE",
		"FN23va",
	); err != nil {
		t.Fatalf("UpdateIdentity() error = %v", err)
	}

	readers.Wait()
}
