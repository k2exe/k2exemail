package mailbox

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreAddAttachmentCopiesAndPersistsMetadata(t *testing.T) {
	store := newTestStore(t)

	msg := testMessage("message-1", FolderDrafts)
	if err := store.Save(msg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "field-notes.txt")
	content := []byte("K2EXEmail attachment test data")

	if err := os.WriteFile(sourcePath, content, 0o600); err != nil {
		t.Fatalf("write source attachment: %v", err)
	}

	attachment, err := store.AddAttachment(
		FolderDrafts,
		msg.ID,
		sourcePath,
	)
	if err != nil {
		t.Fatalf("AddAttachment() error = %v", err)
	}

	if attachment.ID == "" {
		t.Fatal("attachment ID is empty")
	}
	if attachment.Name != "field-notes.txt" {
		t.Fatalf(
			"attachment Name = %q, want field-notes.txt",
			attachment.Name,
		)
	}
	if attachment.Size != int64(len(content)) {
		t.Fatalf(
			"attachment Size = %d, want %d",
			attachment.Size,
			len(content),
		)
	}

	wantHashBytes := sha256.Sum256(content)
	wantHash := hex.EncodeToString(wantHashBytes[:])

	if attachment.SHA256 != wantHash {
		t.Fatalf(
			"attachment SHA256 = %q, want %q",
			attachment.SHA256,
			wantHash,
		)
	}

	stored, err := store.Load(FolderDrafts, msg.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(stored.Attachments) != 1 {
		t.Fatalf(
			"stored Attachments = %d, want 1",
			len(stored.Attachments),
		)
	}

	if stored.Attachments[0].ID != attachment.ID {
		t.Fatalf(
			"stored attachment ID = %q, want %q",
			stored.Attachments[0].ID,
			attachment.ID,
		)
	}

	if err := os.Remove(sourcePath); err != nil {
		t.Fatalf("remove original source: %v", err)
	}

	file, metadata, err := store.OpenAttachment(
		FolderDrafts,
		msg.ID,
		attachment.ID,
	)
	if err != nil {
		t.Fatalf("OpenAttachment() error = %v", err)
	}
	defer file.Close()

	gotContent, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("read stored attachment: %v", err)
	}

	if string(gotContent) != string(content) {
		t.Fatalf(
			"stored attachment content = %q, want %q",
			gotContent,
			content,
		)
	}

	if metadata.Name != attachment.Name {
		t.Fatalf(
			"metadata Name = %q, want %q",
			metadata.Name,
			attachment.Name,
		)
	}
}

func TestStoreMoveCarriesAttachments(t *testing.T) {
	store := newTestStore(t)

	msg := testMessage("message-1", FolderDrafts)
	if err := store.Save(msg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	sourcePath := filepath.Join(t.TempDir(), "map.txt")
	content := []byte("portable operating map")

	if err := os.WriteFile(sourcePath, content, 0o600); err != nil {
		t.Fatalf("write source attachment: %v", err)
	}

	attachment, err := store.AddAttachment(
		FolderDrafts,
		msg.ID,
		sourcePath,
	)
	if err != nil {
		t.Fatalf("AddAttachment() error = %v", err)
	}

	if err := store.Move(
		FolderDrafts,
		FolderOutbox,
		msg.ID,
	); err != nil {
		t.Fatalf("Move() error = %v", err)
	}

	file, _, err := store.OpenAttachment(
		FolderOutbox,
		msg.ID,
		attachment.ID,
	)
	if err != nil {
		t.Fatalf("OpenAttachment() after move error = %v", err)
	}
	defer file.Close()

	gotContent, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("read moved attachment: %v", err)
	}

	if string(gotContent) != string(content) {
		t.Fatalf(
			"moved attachment content = %q, want %q",
			gotContent,
			content,
		)
	}
}

func TestStoreAddAttachmentRejectsDirectory(t *testing.T) {
	store := newTestStore(t)

	msg := testMessage("message-1", FolderDrafts)
	if err := store.Save(msg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	_, err := store.AddAttachment(
		FolderDrafts,
		msg.ID,
		t.TempDir(),
	)
	if err == nil {
		t.Fatal("AddAttachment() expected directory error")
	}
}

func TestStoreOpenAttachmentRejectsUnknownID(t *testing.T) {
	store := newTestStore(t)

	msg := testMessage("message-1", FolderDrafts)
	if err := store.Save(msg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	file, _, err := store.OpenAttachment(
		FolderDrafts,
		msg.ID,
		"not-present",
	)
	if file != nil {
		_ = file.Close()
		t.Fatal("OpenAttachment() returned file for unknown attachment")
	}
	if err == nil {
		t.Fatal("OpenAttachment() expected error")
	}
}
