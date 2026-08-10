package mailbox

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestFolderValid(t *testing.T) {
	valid := []Folder{
		FolderInbox,
		FolderDrafts,
		FolderOutbox,
		FolderSent,
		FolderArchive,
		FolderSpam,
		FolderTrash,
	}

	for _, folder := range valid {
		if !folder.Valid() {
			t.Fatalf("expected folder %q to be valid", folder)
		}
	}

	if Folder("unknown").Valid() {
		t.Fatal("unexpected valid unknown folder")
	}
}

func TestMessageJSONRoundTrip(t *testing.T) {
	now := time.Date(2026, 8, 9, 20, 0, 0, 0, time.UTC)

	original := Message{
		SchemaVersion: CurrentSchemaVersion,
		ID:            "local-message-1",
		Folder:        FolderDrafts,
		From:          "K2EXE",
		To:            []string{"W2ABC"},
		Subject:       "Test message",
		Body:          "Hello from K2EXEmail.",
		Starred:       true,
		Attachments: []Attachment{
			{
				ID:        "attachment-1",
				Name:      "map.pdf",
				MediaType: "application/pdf",
				Size:      1234,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal message: %v", err)
	}

	if decoded.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("SchemaVersion = %d, want %d", decoded.SchemaVersion, CurrentSchemaVersion)
	}

	if decoded.ID != original.ID {
		t.Fatalf("ID = %q, want %q", decoded.ID, original.ID)
	}

	if strings.Contains(string(data), `"folder"`) {
		t.Fatal("serialized message unexpectedly contains folder")
	}

	if decoded.Subject != original.Subject {
		t.Fatalf("Subject = %q, want %q", decoded.Subject, original.Subject)
	}

	if len(decoded.Attachments) != 1 {
		t.Fatalf("Attachments = %d, want 1", len(decoded.Attachments))
	}
}
