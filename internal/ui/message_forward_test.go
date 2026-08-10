package ui

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

func TestForwardSubject(t *testing.T) {
	tests := []struct {
		subject string
		want    string
	}{
		{"Test message", "Fwd: Test message"},
		{"Fwd: Test message", "Fwd: Test message"},
		{"FWD: Test message", "FWD: Test message"},
		{"  Test message  ", "Fwd: Test message"},
		{"", "Fwd:"},
	}

	for _, tt := range tests {
		if got := forwardSubject(tt.subject); got != tt.want {
			t.Fatalf(
				"forwardSubject(%q) = %q, want %q",
				tt.subject,
				got,
				tt.want,
			)
		}
	}
}

func TestForwardBodyIncludesOriginalHeaders(
	t *testing.T,
) {
	original := mailbox.Message{
		From:    "W2ABC",
		To:      []string{"K2EXE"},
		Cc:      []string{"W3DEF"},
		Subject: "Test",
		Body:    "hello\r\nworld",
		CreatedAt: time.Date(
			2026,
			time.August,
			10,
			15,
			4,
			0,
			0,
			time.UTC,
		),
	}

	got := forwardBody(original)

	for _, want := range []string{
		"---------- Forwarded message ----------",
		"From: W2ABC",
		"Date: Mon, 10 Aug 2026 15:04 UTC",
		"Subject: Test",
		"To: K2EXE",
		"Cc: W3DEF",
		"hello\nworld",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf(
				"forwardBody() missing %q:\n%s",
				want,
				got,
			)
		}
	}
}

func TestPrepareForwardDraftResetsMessageState(
	t *testing.T,
) {
	draft := mailbox.Message{
		ID:         "draft-1",
		Folder:     mailbox.FolderDrafts,
		From:       "OLD",
		To:         []string{"OLD"},
		Cc:         []string{"OLD"},
		WinlinkMID: "OLDMID",
		Attachments: []mailbox.Attachment{
			{ID: "old-attachment"},
		},
		Starred: true,
		Unread:  true,
		P2POnly: true,
	}

	got := prepareForwardDraft(
		draft,
		mailbox.Message{
			From:    "W2ABC",
			Subject: "Original",
			Body:    "body",
		},
	)

	if got.ID != "draft-1" {
		t.Fatalf("draft ID changed to %q", got.ID)
	}
	if got.Folder != mailbox.FolderDrafts {
		t.Fatalf("folder = %q", got.Folder)
	}
	if got.From != "" ||
		len(got.To) != 0 ||
		len(got.Cc) != 0 {
		t.Fatalf(
			"forward inherited addressing: from=%q to=%v cc=%v",
			got.From,
			got.To,
			got.Cc,
		)
	}
	if got.WinlinkMID != "" {
		t.Fatalf(
			"forward inherited MID %q",
			got.WinlinkMID,
		)
	}
	if len(got.Attachments) != 0 {
		t.Fatalf(
			"forward inherited %d attachment records",
			len(got.Attachments),
		)
	}
	if got.Starred || got.Unread || got.P2POnly {
		t.Fatalf(
			"forward inherited state: starred=%v unread=%v p2p=%v",
			got.Starred,
			got.Unread,
			got.P2POnly,
		)
	}
}

func TestPrepareForwardMessageCopiesAttachments(
	t *testing.T,
) {
	store := mailbox.NewStore(t.TempDir())

	if err := store.Prepare(); err != nil {
		t.Fatal(err)
	}

	original, err := mailbox.NewMessage(
		mailbox.FolderInbox,
	)
	if err != nil {
		t.Fatal(err)
	}

	original.From = "W2ABC"
	original.To = []string{"K2EXE"}
	original.Subject = "Attachment forward"
	original.Body = "message body"

	if err := store.Save(original); err != nil {
		t.Fatal(err)
	}

	if _, err := store.AddAttachmentData(
		mailbox.FolderInbox,
		original.ID,
		"one.txt",
		[]byte("first attachment"),
	); err != nil {
		t.Fatal(err)
	}

	if _, err := store.AddAttachmentData(
		mailbox.FolderInbox,
		original.ID,
		"two.txt",
		[]byte("second attachment"),
	); err != nil {
		t.Fatal(err)
	}

	original, err = store.Load(
		mailbox.FolderInbox,
		original.ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	forward, err := prepareForwardMessage(
		store,
		original,
	)
	if err != nil {
		t.Fatalf(
			"prepareForwardMessage() error = %v",
			err,
		)
	}

	if len(forward.Attachments) != 2 {
		t.Fatalf(
			"forward attachments = %d, want 2",
			len(forward.Attachments),
		)
	}

	wantData := []string{
		"first attachment",
		"second attachment",
	}

	for i, attachment := range forward.Attachments {
		if attachment.ID == original.Attachments[i].ID {
			t.Fatalf(
				"forward reused source attachment ID %q",
				attachment.ID,
			)
		}

		reader, _, err := store.OpenAttachmentReader(
			mailbox.FolderDrafts,
			forward.ID,
			attachment.ID,
		)
		if err != nil {
			t.Fatal(err)
		}

		data, readErr := io.ReadAll(reader)
		closeErr := reader.Close()

		if readErr != nil {
			t.Fatal(readErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}

		if string(data) != wantData[i] {
			t.Fatalf(
				"attachment %d data = %q, want %q",
				i,
				data,
				wantData[i],
			)
		}
	}

	saved, err := store.Load(
		mailbox.FolderDrafts,
		forward.ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(saved.Attachments) != 2 {
		t.Fatalf(
			"stored forward attachments = %d, want 2",
			len(saved.Attachments),
		)
	}
}

type failingForwardStore struct {
	addErr       error
	deleteCalled bool
	deleteFolder mailbox.Folder
	deleteID     string
}

func (s *failingForwardStore) Save(
	mailbox.Message,
) error {
	return nil
}

func (s *failingForwardStore) Delete(
	folder mailbox.Folder,
	id string,
) error {
	s.deleteCalled = true
	s.deleteFolder = folder
	s.deleteID = id
	return nil
}

func (s *failingForwardStore) OpenAttachmentReader(
	mailbox.Folder,
	string,
	string,
) (io.ReadCloser, mailbox.Attachment, error) {
	return io.NopCloser(
			strings.NewReader("attachment data"),
		),
		mailbox.Attachment{
			ID:     "source-attachment",
			Name:   "test.txt",
			Size:   int64(len("attachment data")),
			SHA256: "source-hash",
		},
		nil
}

func (s *failingForwardStore) AddAttachmentReader(
	mailbox.Folder,
	string,
	string,
	io.Reader,
) (mailbox.Attachment, error) {
	return mailbox.Attachment{}, s.addErr
}

func TestPrepareForwardMessageCleansUpCopyFailure(
	t *testing.T,
) {
	wantErr := errors.New("copy failed")
	store := &failingForwardStore{
		addErr: wantErr,
	}

	original := mailbox.Message{
		ID:     "message-1",
		Folder: mailbox.FolderInbox,
		Attachments: []mailbox.Attachment{
			{
				ID:   "source-attachment",
				Name: "test.txt",
			},
		},
	}

	_, err := prepareForwardMessage(
		store,
		original,
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf(
			"prepareForwardMessage() error = %v, want wrapped %v",
			err,
			wantErr,
		)
	}

	if !store.deleteCalled {
		t.Fatal(
			"incomplete forward draft was not deleted",
		)
	}
	if store.deleteFolder != mailbox.FolderDrafts {
		t.Fatalf(
			"cleanup folder = %q",
			store.deleteFolder,
		)
	}
	if store.deleteID == "" {
		t.Fatal("cleanup draft ID is empty")
	}
}
