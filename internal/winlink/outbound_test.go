package winlink

import (
	"errors"
	"testing"
	"time"

	"github.com/k2exe/k2exemail/internal/mailbox"
	"github.com/la5nta/wl2k-go/fbb"
)

func TestToFBB(t *testing.T) {
	created := time.Date(
		2026, 8, 9, 21, 30, 0, 0, time.UTC,
	)

	msg := mailbox.Message{
		ID:        "local-1",
		Folder:    mailbox.FolderOutbox,
		From:      "k2exe",
		To:        []string{"kr2ssy"},
		Cc:        []string{"test@example.com"},
		Subject:   "Adapter test",
		Body:      "Hello from K2EXEmail",
		P2POnly:   true,
		CreatedAt: created,
	}

	got, err := ToFBB("K2EXE", msg, nil)
	if err != nil {
		t.Fatalf("ToFBB() error = %v", err)
	}

	if got.From().String() != "K2EXE" {
		t.Fatalf("From = %q, want K2EXE", got.From().String())
	}

	if len(got.To()) != 1 || got.To()[0].String() != "KR2SSY" {
		t.Fatalf("To = %#v", got.To())
	}

	if len(got.Cc()) != 1 ||
		got.Cc()[0].String() != "SMTP:test@example.com" {
		t.Fatalf("Cc = %#v", got.Cc())
	}

	if got.Subject() != "Adapter test" {
		t.Fatalf("Subject = %q", got.Subject())
	}

	body, err := got.Body()
	if err != nil {
		t.Fatalf("Body() error = %v", err)
	}

	if body != "Hello from K2EXEmail\r\n" {
		t.Fatalf("Body = %q", body)
	}

	if got.Header.Get(p2pOnlyHeader) != "true" {
		t.Fatal("P2P-only header not set")
	}

	if !got.Date().Equal(created) {
		t.Fatalf(
			"Date = %v, want %v",
			got.Date(),
			created,
		)
	}

	if len(got.MID()) != fbb.MaxMIDLength {
		t.Fatalf(
			"MID length = %d, want %d",
			len(got.MID()),
			fbb.MaxMIDLength,
		)
	}
}

func TestToFBBUsesStationCallsignWhenFromEmpty(t *testing.T) {
	msg := mailbox.Message{
		ID:      "legacy-outbox",
		Folder:  mailbox.FolderOutbox,
		To:      []string{"KR2SSY"},
		Subject: "Legacy message",
		Body:    "Body",
	}

	got, err := ToFBB("k2exe", msg, nil)
	if err != nil {
		t.Fatalf("ToFBB() error = %v", err)
	}

	if got.From().String() != "K2EXE" {
		t.Fatalf("From = %q, want K2EXE", got.From().String())
	}
}

func TestToFBBPreservesExistingMID(t *testing.T) {
	msg := mailbox.Message{
		ID:         "local-2",
		WinlinkMID: "ABC123456789",
		Folder:     mailbox.FolderOutbox,
		From:       "K2EXE",
		To:         []string{"KR2SSY"},
		Subject:    "Existing MID",
		Body:       "Body",
	}

	got, err := ToFBB("K2EXE", msg, nil)
	if err != nil {
		t.Fatalf("ToFBB() error = %v", err)
	}

	if got.MID() != msg.WinlinkMID {
		t.Fatalf(
			"MID = %q, want %q",
			got.MID(),
			msg.WinlinkMID,
		)
	}
}

func TestToFBBLoadsAttachments(t *testing.T) {
	attachment := mailbox.Attachment{
		ID:   "attachment-1",
		Name: "notes.txt",
		Size: 5,
	}

	msg := mailbox.Message{
		ID:          "local-3",
		Folder:      mailbox.FolderOutbox,
		From:        "K2EXE",
		To:          []string{"KR2SSY"},
		Subject:     "Attachment",
		Body:        "See attached.",
		Attachments: []mailbox.Attachment{attachment},
	}

	loader := func(
		folder mailbox.Folder,
		messageID string,
		got mailbox.Attachment,
	) ([]byte, error) {
		if folder != mailbox.FolderOutbox {
			t.Fatalf("folder = %q", folder)
		}
		if messageID != msg.ID {
			t.Fatalf("messageID = %q", messageID)
		}
		if got.ID != attachment.ID {
			t.Fatalf("attachment ID = %q", got.ID)
		}

		return []byte("hello"), nil
	}

	wire, err := ToFBB("K2EXE", msg, loader)
	if err != nil {
		t.Fatalf("ToFBB() error = %v", err)
	}

	files := wire.Files()
	if len(files) != 1 {
		t.Fatalf("Files() count = %d, want 1", len(files))
	}

	if files[0].Name() != "notes.txt" {
		t.Fatalf("filename = %q", files[0].Name())
	}

	if string(files[0].Data()) != "hello" {
		t.Fatalf("attachment data = %q", files[0].Data())
	}
}

func TestToFBBRequiresAttachmentLoader(t *testing.T) {
	msg := mailbox.Message{
		ID:      "local-4",
		Folder:  mailbox.FolderOutbox,
		From:    "K2EXE",
		To:      []string{"KR2SSY"},
		Subject: "Attachment",
		Body:    "Body",
		Attachments: []mailbox.Attachment{
			{
				ID:   "attachment-1",
				Name: "notes.txt",
				Size: 5,
			},
		},
	}

	if _, err := ToFBB("K2EXE", msg, nil); err == nil {
		t.Fatal("ToFBB() expected attachment loader error")
	}
}

func TestToFBBPropagatesAttachmentError(t *testing.T) {
	msg := mailbox.Message{
		ID:      "local-5",
		Folder:  mailbox.FolderOutbox,
		From:    "K2EXE",
		To:      []string{"KR2SSY"},
		Subject: "Attachment",
		Body:    "Body",
		Attachments: []mailbox.Attachment{
			{
				ID:   "attachment-1",
				Name: "notes.txt",
				Size: 5,
			},
		},
	}

	want := errors.New("read failed")

	_, err := ToFBB(
		"K2EXE",
		msg,
		func(
			mailbox.Folder,
			string,
			mailbox.Attachment,
		) ([]byte, error) {
			return nil, want
		},
	)

	if !errors.Is(err, want) {
		t.Fatalf("ToFBB() error = %v, want %v", err, want)
	}
}

func TestToFBBRejectsMissingStationCallsign(t *testing.T) {
	msg := mailbox.Message{
		To:      []string{"KR2SSY"},
		Subject: "Test",
		Body:    "Body",
	}

	if _, err := ToFBB("", msg, nil); err == nil {
		t.Fatal("ToFBB() expected callsign error")
	}
}

func TestToFBBRejectsInvalidMessage(t *testing.T) {
	msg := mailbox.Message{
		ID:      "local-6",
		Folder:  mailbox.FolderOutbox,
		From:    "K2EXE",
		To:      []string{"KR2SSY"},
		Subject: "Empty body",
	}

	if _, err := ToFBB("K2EXE", msg, nil); err == nil {
		t.Fatal("ToFBB() expected validation error")
	}
}
