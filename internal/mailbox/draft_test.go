package mailbox

import "testing"

func TestNewDraft(t *testing.T) {
	first, err := NewDraft()
	if err != nil {
		t.Fatalf("NewDraft() error = %v", err)
	}

	second, err := NewDraft()
	if err != nil {
		t.Fatalf("second NewDraft() error = %v", err)
	}

	if first.ID == "" {
		t.Fatal("draft ID is empty")
	}

	if first.ID == second.ID {
		t.Fatal("two drafts received the same ID")
	}

	if first.Folder != FolderDrafts {
		t.Fatalf("Folder = %q, want %q", first.Folder, FolderDrafts)
	}

	if first.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf(
			"SchemaVersion = %d, want %d",
			first.SchemaVersion,
			CurrentSchemaVersion,
		)
	}

	if first.CreatedAt.IsZero() || first.UpdatedAt.IsZero() {
		t.Fatal("draft timestamps are not initialized")
	}
}
