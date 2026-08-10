package mailbox

import (
	"io"
	"testing"
)

func TestStoreImportMessageWithAttachments(t *testing.T) {
	store := newTestStore(t)

	msg, err := NewMessage(FolderInbox)
	if err != nil {
		t.Fatalf("NewMessage() error = %v", err)
	}

	msg.WinlinkMID = "ABC123456789"
	msg.From = "W2ABC"
	msg.To = []string{"K2EXE"}
	msg.Subject = "Inbound test"
	msg.Body = "Received body"
	msg.Unread = true

	imported, err := store.ImportMessage(
		msg,
		[]AttachmentContent{
			{
				Name: "notes.txt",
				Data: []byte("hello"),
			},
		},
	)
	if err != nil {
		t.Fatalf("ImportMessage() error = %v", err)
	}

	if len(imported.Attachments) != 1 {
		t.Fatalf(
			"Attachments = %d, want 1",
			len(imported.Attachments),
		)
	}

	stored, err := store.Load(FolderInbox, msg.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if stored.WinlinkMID != msg.WinlinkMID {
		t.Fatalf(
			"WinlinkMID = %q, want %q",
			stored.WinlinkMID,
			msg.WinlinkMID,
		)
	}

	file, _, err := store.OpenAttachment(
		FolderInbox,
		msg.ID,
		stored.Attachments[0].ID,
	)
	if err != nil {
		t.Fatalf("OpenAttachment() error = %v", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(data) != "hello" {
		t.Fatalf("attachment data = %q, want hello", data)
	}
}

func TestStoreImportFailureDoesNotPublishPartialMessage(t *testing.T) {
	store := newTestStore(t)

	msg, err := NewMessage(FolderInbox)
	if err != nil {
		t.Fatalf("NewMessage() error = %v", err)
	}

	msg.From = "W2ABC"
	msg.To = []string{"K2EXE"}
	msg.Subject = "Should not publish"
	msg.Body = "Body"

	_, err = store.ImportMessage(
		msg,
		[]AttachmentContent{
			{
				Name: "good.txt",
				Data: []byte("good"),
			},
			{
				Name: "",
				Data: []byte("bad"),
			},
		},
	)
	if err == nil {
		t.Fatal("ImportMessage() expected attachment error")
	}

	messages, err := store.List(FolderInbox)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(messages) != 0 {
		t.Fatalf(
			"Inbox contains %d partial messages, want 0",
			len(messages),
		)
	}
}

func TestStoreImportRejectsExistingMessage(t *testing.T) {
	store := newTestStore(t)

	msg, err := NewMessage(FolderInbox)
	if err != nil {
		t.Fatalf("NewMessage() error = %v", err)
	}

	if err := store.Save(msg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if _, err := store.ImportMessage(msg, nil); err == nil {
		t.Fatal("ImportMessage() expected existing-message error")
	}
}

func TestNewMessageUsesRequestedFolder(t *testing.T) {
	msg, err := NewMessage(FolderInbox)
	if err != nil {
		t.Fatalf("NewMessage() error = %v", err)
	}

	if msg.Folder != FolderInbox {
		t.Fatalf("Folder = %q, want %q", msg.Folder, FolderInbox)
	}

	if msg.ID == "" {
		t.Fatal("message ID is empty")
	}
}

func TestNewMessageRejectsInvalidFolder(t *testing.T) {
	if _, err := NewMessage(Folder("bogus")); err == nil {
		t.Fatal("NewMessage() expected invalid-folder error")
	}
}
