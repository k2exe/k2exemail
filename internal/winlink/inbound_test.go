package winlink

import (
	"testing"
	"time"

	"github.com/la5nta/wl2k-go/fbb"
)

func TestFromFBB(t *testing.T) {
	sentAt := time.Date(
		2026, 8, 9, 21, 45, 0, 0, time.UTC,
	)

	wire := fbb.NewMessage(fbb.Private, "W2ABC")
	wire.Header.Set(fbb.HEADER_MID, "ABC123456789")
	wire.SetDate(sentAt)
	wire.SetFrom("W2ABC")
	wire.AddTo("K2EXE")
	wire.AddCc("test@example.com")
	wire.SetSubject("Inbound test")

	if err := wire.SetBody("Received body"); err != nil {
		t.Fatalf("SetBody() error = %v", err)
	}

	wire.AddFile(
		fbb.NewFile(
			"notes.txt",
			[]byte("attachment"),
		),
	)

	msg, attachments, err := FromFBB(wire)
	if err != nil {
		t.Fatalf("FromFBB() error = %v", err)
	}

	if msg.WinlinkMID != "ABC123456789" {
		t.Fatalf("WinlinkMID = %q", msg.WinlinkMID)
	}

	if msg.From != "W2ABC" {
		t.Fatalf("From = %q", msg.From)
	}

	if len(msg.To) != 1 || msg.To[0] != "K2EXE" {
		t.Fatalf("To = %#v", msg.To)
	}

	if len(msg.Cc) != 1 ||
		msg.Cc[0] != "SMTP:test@example.com" {
		t.Fatalf("Cc = %#v", msg.Cc)
	}

	if msg.Subject != "Inbound test" {
		t.Fatalf("Subject = %q", msg.Subject)
	}

	if msg.Body != "Received body\r\n" {
		t.Fatalf("Body = %q", msg.Body)
	}

	if !msg.Unread {
		t.Fatal("Unread = false, want true")
	}

	if !msg.CreatedAt.Equal(sentAt) {
		t.Fatalf(
			"CreatedAt = %v, want %v",
			msg.CreatedAt,
			sentAt,
		)
	}

	if msg.UpdatedAt.IsZero() {
		t.Fatal("UpdatedAt is zero")
	}

	if len(attachments) != 1 {
		t.Fatalf(
			"attachments = %d, want 1",
			len(attachments),
		)
	}

	if attachments[0].Name != "notes.txt" {
		t.Fatalf(
			"attachment name = %q",
			attachments[0].Name,
		)
	}

	if string(attachments[0].Data) != "attachment" {
		t.Fatalf(
			"attachment data = %q",
			attachments[0].Data,
		)
	}
}

func TestFromFBBRejectsNilMessage(t *testing.T) {
	if _, _, err := FromFBB(nil); err == nil {
		t.Fatal("FromFBB() expected nil-message error")
	}
}

func TestFromFBBRejectsMissingMID(t *testing.T) {
	wire := fbb.NewMessage(fbb.Private, "W2ABC")
	wire.Header.Del(fbb.HEADER_MID)

	if _, _, err := FromFBB(wire); err == nil {
		t.Fatal("FromFBB() expected missing-MID error")
	}
}
