package config

import (
	"os"
	"path/filepath"
	"reflect"
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
	if !reflect.DeepEqual(current, updated) {
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

	if !reflect.DeepEqual(persisted, updated) {
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
	}.Normalized()

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
	if !reflect.DeepEqual(current, initial) {
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

func TestRuntimeCurrentReturnsIndependentSnapshot(t *testing.T) {
	runtime := NewRuntime(
		filepath.Join(t.TempDir(), FileName),
		Config{
			Callsign: "K2EXE",
			Locator:  "FN23va",
		},
	)

	snapshot := runtime.Current()
	if len(snapshot.ConnectionProfiles) == 0 {
		t.Fatal("Current() returned no default connection profiles")
	}
	if snapshot.ConnectionProfiles[0].CMS == nil {
		t.Fatal("first default connection profile has no CMS settings")
	}

	snapshot.ConnectionProfiles[0].Name = "mutated"
	snapshot.ConnectionProfiles[0].CMS.Mode = CMSModeProduction

	current := runtime.Current()

	if current.ConnectionProfiles[0].Name == "mutated" {
		t.Fatal("Current() snapshot mutated runtime profile name")
	}
	if current.ConnectionProfiles[0].CMS.Mode != CMSModeTest {
		t.Fatalf(
			"runtime CMS mode = %q after snapshot mutation, want %q",
			current.ConnectionProfiles[0].CMS.Mode,
			CMSModeTest,
		)
	}
}

func TestRuntimeUpdateIdentityReturnsIndependentSnapshot(t *testing.T) {
	runtime := NewRuntime(
		filepath.Join(t.TempDir(), FileName),
		Config{
			Callsign: "K2EXE",
			Locator:  "FN23va",
		},
	)

	updated, err := runtime.UpdateIdentity(
		"K2EXE",
		"FN23va",
	)
	if err != nil {
		t.Fatalf("UpdateIdentity() error = %v", err)
	}

	if len(updated.ConnectionProfiles) == 0 ||
		updated.ConnectionProfiles[0].CMS == nil {
		t.Fatal("UpdateIdentity() returned incomplete default profiles")
	}

	updated.ConnectionProfiles[0].Name = "mutated"
	updated.ConnectionProfiles[0].CMS.Mode = CMSModeProduction

	current := runtime.Current()

	if current.ConnectionProfiles[0].Name == "mutated" {
		t.Fatal("UpdateIdentity() result mutated runtime profile name")
	}
	if current.ConnectionProfiles[0].CMS.Mode != CMSModeTest {
		t.Fatalf(
			"runtime CMS mode = %q after result mutation, want %q",
			current.ConnectionProfiles[0].CMS.Mode,
			CMSModeTest,
		)
	}
}

func TestRuntimeCreateConnectionProfilePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)

	runtime := NewRuntime(
		path,
		Config{
			Callsign: "K2EXE",
			Locator:  "FN23va",
		},
	)

	created, err := runtime.CreateConnectionProfile(
		ConnectionProfile{
			Name:      "  Mesh Peer  ",
			Network:   NetworkAREDN,
			Transport: TransportDirectTCP,
			TCP: &TCPProfile{
				Address:    "  node.local.mesh:8772  ",
				TargetCall: " w2abc ",
			},
		},
	)
	if err != nil {
		t.Fatalf("CreateConnectionProfile() error = %v", err)
	}

	if created.ID == "" {
		t.Fatal("created profile has empty ID")
	}
	if created.Name != "Mesh Peer" {
		t.Fatalf("Name = %q, want Mesh Peer", created.Name)
	}
	if created.TCP == nil {
		t.Fatal("created profile has nil TCP settings")
	}
	if created.TCP.TargetCall != "W2ABC" {
		t.Fatalf(
			"TargetCall = %q, want W2ABC",
			created.TCP.TargetCall,
		)
	}

	persisted, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	var found bool
	for _, profile := range persisted.ConnectionProfiles {
		if profile.ID == created.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf(
			"created profile %q was not persisted",
			created.ID,
		)
	}

	created.Name = "mutated"
	created.TCP.TargetCall = "N0BAD"

	current := runtime.Current()
	for _, profile := range current.ConnectionProfiles {
		if profile.ID != created.ID {
			continue
		}
		if profile.Name == "mutated" ||
			profile.TCP.TargetCall == "N0BAD" {
			t.Fatal(
				"created result mutated runtime configuration",
			)
		}
	}
}

func TestRuntimeCreateConnectionProfileRejectsCallerID(t *testing.T) {
	runtime := NewRuntime(
		filepath.Join(t.TempDir(), FileName),
		Default(),
	)

	_, err := runtime.CreateConnectionProfile(
		ConnectionProfile{
			ID:        "caller-id",
			Name:      "Peer",
			Network:   NetworkAREDN,
			Transport: TransportDirectTCP,
			TCP: &TCPProfile{
				Address:    "node.local.mesh:8772",
				TargetCall: "W2ABC",
			},
		},
	)
	if err == nil {
		t.Fatal(
			"CreateConnectionProfile() expected caller ID error",
		)
	}
}

func TestRuntimeUpdateConnectionProfilePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	runtime := NewRuntime(path, Default())

	created, err := runtime.CreateConnectionProfile(
		ConnectionProfile{
			Name:      "Mesh Peer",
			Network:   NetworkAREDN,
			Transport: TransportDirectTCP,
			TCP: &TCPProfile{
				Address:    "node.local.mesh:8772",
				TargetCall: "W2ABC",
			},
		},
	)
	if err != nil {
		t.Fatalf("CreateConnectionProfile() error = %v", err)
	}

	created.Name = " Updated Peer "
	created.TCP.Address = " 10.42.1.15:8772 "
	created.TCP.TargetCall = " w3xyz "

	updated, err := runtime.UpdateConnectionProfile(created)
	if err != nil {
		t.Fatalf("UpdateConnectionProfile() error = %v", err)
	}

	if updated.ID != created.ID {
		t.Fatalf(
			"ID = %q, want %q",
			updated.ID,
			created.ID,
		)
	}
	if updated.Name != "Updated Peer" {
		t.Fatalf(
			"Name = %q, want Updated Peer",
			updated.Name,
		)
	}
	if updated.TCP.TargetCall != "W3XYZ" {
		t.Fatalf(
			"TargetCall = %q, want W3XYZ",
			updated.TCP.TargetCall,
		)
	}

	persisted, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	var got *ConnectionProfile
	for i := range persisted.ConnectionProfiles {
		if persisted.ConnectionProfiles[i].ID == updated.ID {
			got = &persisted.ConnectionProfiles[i]
			break
		}
	}

	if got == nil {
		t.Fatal("updated profile not found after reload")
	}
	if got.Name != "Updated Peer" {
		t.Fatalf(
			"persisted Name = %q, want Updated Peer",
			got.Name,
		)
	}
}

func TestRuntimeUpdateConnectionProfileRejectsUnknownID(t *testing.T) {
	runtime := NewRuntime(
		filepath.Join(t.TempDir(), FileName),
		Default(),
	)

	_, err := runtime.UpdateConnectionProfile(
		ConnectionProfile{
			ID:        "missing",
			Name:      "Missing",
			Network:   NetworkAREDN,
			Transport: TransportDirectTCP,
			TCP: &TCPProfile{
				Address:    "node.local.mesh:8772",
				TargetCall: "W2ABC",
			},
		},
	)
	if err == nil {
		t.Fatal(
			"UpdateConnectionProfile() expected not-found error",
		)
	}
}

func TestRuntimeDeleteConnectionProfilePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	runtime := NewRuntime(path, Default())

	created, err := runtime.CreateConnectionProfile(
		ConnectionProfile{
			Name:      "Mesh Peer",
			Network:   NetworkAREDN,
			Transport: TransportDirectTCP,
			TCP: &TCPProfile{
				Address:    "node.local.mesh:8772",
				TargetCall: "W2ABC",
			},
		},
	)
	if err != nil {
		t.Fatalf("CreateConnectionProfile() error = %v", err)
	}

	if err := runtime.DeleteConnectionProfile(created.ID); err != nil {
		t.Fatalf("DeleteConnectionProfile() error = %v", err)
	}

	current := runtime.Current()
	for _, profile := range current.ConnectionProfiles {
		if profile.ID == created.ID {
			t.Fatal("deleted profile remains in runtime")
		}
	}

	persisted, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	for _, profile := range persisted.ConnectionProfiles {
		if profile.ID == created.ID {
			t.Fatal("deleted profile remains after reload")
		}
	}
}

func TestRuntimeDeleteConnectionProfileRejectsUnknownID(t *testing.T) {
	runtime := NewRuntime(
		filepath.Join(t.TempDir(), FileName),
		Default(),
	)

	if err := runtime.DeleteConnectionProfile("missing"); err == nil {
		t.Fatal(
			"DeleteConnectionProfile() expected not-found error",
		)
	}
}

func TestRuntimeUpdateConnectionProfileSaveFailureKeepsCurrentConfig(t *testing.T) {
	root := t.TempDir()

	blockedParent := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(
		blockedParent,
		[]byte("file"),
		0o600,
	); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	initial := Default()
	initial.ConnectionProfiles = append(
		initial.ConnectionProfiles,
		ConnectionProfile{
			ID:        "mesh-peer",
			Name:      "Mesh Peer",
			Network:   NetworkAREDN,
			Transport: TransportDirectTCP,
			TCP: &TCPProfile{
				Address:    "node.local.mesh:8772",
				TargetCall: "W2ABC",
			},
		},
	)
	initial = initial.Normalized()

	runtime := NewRuntime(
		filepath.Join(blockedParent, FileName),
		initial,
	)

	before := runtime.Current()

	updated := before.ConnectionProfiles[len(before.ConnectionProfiles)-1]
	updated.Name = "Updated Peer"

	if _, err := runtime.UpdateConnectionProfile(updated); err == nil {
		t.Fatal(
			"UpdateConnectionProfile() expected save error",
		)
	}

	after := runtime.Current()

	if !reflect.DeepEqual(after, before) {
		t.Fatalf(
			"Current() changed after failed profile save:\n got %#v\nwant %#v",
			after,
			before,
		)
	}
}
