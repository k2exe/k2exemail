package mailbox

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStorePrepareCreatesFolders(t *testing.T) {
	store := NewStore(t.TempDir())

	if err := store.Prepare(); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	for _, folder := range folders {
		info, err := os.Stat(filepath.Join(store.Root(), string(folder)))
		if err != nil {
			t.Fatalf("folder %q: %v", folder, err)
		}
		if !info.IsDir() {
			t.Fatalf("%q is not a directory", folder)
		}
	}
}

func TestStoreSaveLoad(t *testing.T) {
	store := newTestStore(t)

	now := time.Date(2026, 8, 9, 20, 30, 0, 0, time.UTC)

	msg := Message{
		ID:        "message-1",
		Folder:    FolderDrafts,
		From:      "K2EXE",
		To:        []string{"W2ABC"},
		Subject:   "Portable operation",
		Body:      "Testing offline mail.",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := store.Save(msg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Load(FolderDrafts, msg.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf(
			"SchemaVersion = %d, want %d",
			got.SchemaVersion,
			CurrentSchemaVersion,
		)
	}
	if got.Subject != msg.Subject {
		t.Fatalf("Subject = %q, want %q", got.Subject, msg.Subject)
	}
	if got.Folder != FolderDrafts {
		t.Fatalf("Folder = %q, want %q", got.Folder, FolderDrafts)
	}
}

func TestStoreSaveReplacesExistingMessageSafely(t *testing.T) {
	store := newTestStore(t)

	msg := testMessage("message-1", FolderDrafts)
	msg.Subject = "Before"

	if err := store.Save(msg); err != nil {
		t.Fatalf("first Save() error = %v", err)
	}

	msg.Subject = "After"
	msg.UpdatedAt = msg.UpdatedAt.Add(time.Minute)

	if err := store.Save(msg); err != nil {
		t.Fatalf("second Save() error = %v", err)
	}

	got, err := store.Load(FolderDrafts, msg.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.Subject != "After" {
		t.Fatalf("Subject = %q, want After", got.Subject)
	}

	entries, err := os.ReadDir(
		filepath.Join(store.Root(), string(FolderDrafts), msg.ID),
	)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	if len(entries) != 1 || entries[0].Name() != messageFileName {
		t.Fatalf("message directory contains unexpected files: %v", entries)
	}
}

func TestStoreListNewestFirst(t *testing.T) {
	store := newTestStore(t)

	older := testMessage("older", FolderInbox)
	newer := testMessage("newer", FolderInbox)
	newer.UpdatedAt = older.UpdatedAt.Add(time.Hour)

	if err := store.Save(older); err != nil {
		t.Fatalf("save older: %v", err)
	}
	if err := store.Save(newer); err != nil {
		t.Fatalf("save newer: %v", err)
	}

	messages, err := store.List(FolderInbox)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("List() length = %d, want 2", len(messages))
	}
	if messages[0].ID != "newer" {
		t.Fatalf("first message = %q, want newer", messages[0].ID)
	}
}

func TestStoreMove(t *testing.T) {
	store := newTestStore(t)

	msg := testMessage("message-1", FolderDrafts)

	if err := store.Save(msg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := store.Move(FolderDrafts, FolderOutbox, msg.ID); err != nil {
		t.Fatalf("Move() error = %v", err)
	}

	if _, err := store.Load(FolderDrafts, msg.ID); err == nil {
		t.Fatal("message still exists in drafts")
	}

	got, err := store.Load(FolderOutbox, msg.ID)
	if err != nil {
		t.Fatalf("load outbox message: %v", err)
	}

	if got.Folder != FolderOutbox {
		t.Fatalf("Folder = %q, want %q", got.Folder, FolderOutbox)
	}
}

func TestStoreRejectsUnsafeMessageID(t *testing.T) {
	store := newTestStore(t)

	msg := testMessage("../escape", FolderDrafts)

	if err := store.Save(msg); err == nil {
		t.Fatal("Save() expected unsafe ID error")
	}
}

func TestStoreDelete(t *testing.T) {
	store := newTestStore(t)

	msg := testMessage("message-1", FolderTrash)

	if err := store.Save(msg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := store.Delete(FolderTrash, msg.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := store.Load(FolderTrash, msg.ID); err == nil {
		t.Fatal("deleted message still loads")
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()

	store := NewStore(t.TempDir())

	if err := store.Prepare(); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	return store
}

func testMessage(id string, folder Folder) Message {
	now := time.Date(2026, 8, 9, 20, 30, 0, 0, time.UTC)

	return Message{
		ID:        id,
		Folder:    folder,
		From:      "K2EXE",
		To:        []string{"W2ABC"},
		Subject:   "Test",
		Body:      "Test body",
		CreatedAt: now,
		UpdatedAt: now,
	}
}
