package winlink

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/k2exe/k2exemail/internal/mailbox"
	"github.com/la5nta/wl2k-go/fbb"
)

func TestHandlerPreparePersistsMID(t *testing.T) {
	store := newTestStore(t)

	msg := testOutboundMessage("message-1", "KR2SSY")
	if err := store.Save(msg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	handler := NewHandler(store, "K2EXE")

	if err := handler.Prepare(); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	outbound := handler.GetOutbound()
	if len(outbound) != 1 {
		t.Fatalf("GetOutbound() count = %d, want 1", len(outbound))
	}

	mid := outbound[0].MID()
	if mid == "" {
		t.Fatal("generated MID is empty")
	}

	stored, err := store.Load(
		mailbox.FolderOutbox,
		msg.ID,
	)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if stored.WinlinkMID != mid {
		t.Fatalf(
			"stored WinlinkMID = %q, want %q",
			stored.WinlinkMID,
			mid,
		)
	}

	// A new session must reuse the durable MID rather than create
	// another Winlink message identity.
	if err := handler.Prepare(); err != nil {
		t.Fatalf("second Prepare() error = %v", err)
	}

	outbound = handler.GetOutbound()
	if len(outbound) != 1 {
		t.Fatalf(
			"second GetOutbound() count = %d, want 1",
			len(outbound),
		)
	}

	if outbound[0].MID() != mid {
		t.Fatalf(
			"second MID = %q, want %q",
			outbound[0].MID(),
			mid,
		)
	}
}

func TestHandlerSetSentMovesMessage(t *testing.T) {
	store := newTestStore(t)

	msg := testOutboundMessage("message-1", "KR2SSY")
	if err := store.Save(msg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	handler := NewHandler(store, "K2EXE")
	if err := handler.Prepare(); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	outbound := handler.GetOutbound()
	mid := outbound[0].MID()

	handler.SetSent(mid, false)

	if err := handler.Err(); err != nil {
		t.Fatalf("Handler Err() = %v", err)
	}

	if got := handler.GetOutbound(); len(got) != 0 {
		t.Fatalf(
			"GetOutbound() after SetSent count = %d, want 0",
			len(got),
		)
	}

	if _, err := store.Load(
		mailbox.FolderOutbox,
		msg.ID,
	); err == nil {
		t.Fatal("message still exists in Outbox")
	}

	sent, err := store.Load(
		mailbox.FolderSent,
		msg.ID,
	)
	if err != nil {
		t.Fatalf("load Sent message: %v", err)
	}

	if sent.WinlinkMID != mid {
		t.Fatalf(
			"Sent WinlinkMID = %q, want %q",
			sent.WinlinkMID,
			mid,
		)
	}
}

func TestHandlerSetSentRecordsMoveFailure(t *testing.T) {
	store := newTestStore(t)

	msg := testOutboundMessage("message-1", "KR2SSY")
	if err := store.Save(msg); err != nil {
		t.Fatalf("Save Outbox message: %v", err)
	}

	collision := msg
	collision.Folder = mailbox.FolderSent

	if err := store.Save(collision); err != nil {
		t.Fatalf("Save Sent collision: %v", err)
	}

	handler := NewHandler(store, "K2EXE")
	if err := handler.Prepare(); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	mid := handler.GetOutbound()[0].MID()
	handler.SetSent(mid, false)

	if handler.Err() == nil {
		t.Fatal("Handler Err() expected move failure")
	}

	// Do not offer an already acknowledged message again during this
	// session even though local post-transfer bookkeeping failed.
	if got := handler.GetOutbound(); len(got) != 0 {
		t.Fatalf(
			"GetOutbound() after failed move count = %d, want 0",
			len(got),
		)
	}

	// Most importantly, the queued copy remains durable.
	if _, err := store.Load(
		mailbox.FolderOutbox,
		msg.ID,
	); err != nil {
		t.Fatalf("Outbox message was lost: %v", err)
	}
}

func TestHandlerDeferredMessageReturnsNextSession(t *testing.T) {
	store := newTestStore(t)

	msg := testOutboundMessage("message-1", "KR2SSY")
	if err := store.Save(msg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	handler := NewHandler(store, "K2EXE")
	if err := handler.Prepare(); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	mid := handler.GetOutbound()[0].MID()
	handler.SetDeferred(mid)

	if got := handler.GetOutbound(); len(got) != 0 {
		t.Fatalf(
			"deferred GetOutbound() count = %d, want 0",
			len(got),
		)
	}

	if err := handler.Prepare(); err != nil {
		t.Fatalf("next Prepare() error = %v", err)
	}

	if got := handler.GetOutbound(); len(got) != 1 {
		t.Fatalf(
			"next-session GetOutbound() count = %d, want 1",
			len(got),
		)
	}
}

func TestHandlerFiltersCMSAndP2P(t *testing.T) {
	store := newTestStore(t)

	p2p := testOutboundMessage("p2p", "KR2SSY")
	p2p.P2POnly = true

	regular := testOutboundMessage("regular", "KR2SSY")
	other := testOutboundMessage("other", "W2ABC")

	for _, msg := range []mailbox.Message{
		p2p,
		regular,
		other,
	} {
		if err := store.Save(msg); err != nil {
			t.Fatalf("Save(%q) error = %v", msg.ID, err)
		}
	}

	handler := NewHandler(store, "K2EXE")
	if err := handler.Prepare(); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	// CMS receives regular traffic, but never P2P-only traffic.
	cms := handler.GetOutbound()
	if len(cms) != 2 {
		t.Fatalf(
			"CMS GetOutbound() count = %d, want 2",
			len(cms),
		)
	}

	if err := handler.Prepare(); err != nil {
		t.Fatalf("second Prepare() error = %v", err)
	}

	// P2P follows wl2k-go's established behavior: only messages
	// whose sole receiver is the remote forwarder are offered.
	peer := handler.GetOutbound(
		fbb.AddressFromString("KR2SSY"),
	)

	if len(peer) != 2 {
		t.Fatalf(
			"P2P GetOutbound() count = %d, want 2",
			len(peer),
		)
	}

	for _, msg := range peer {
		if !msg.IsOnlyReceiver(
			fbb.AddressFromString("KR2SSY"),
		) {
			t.Fatalf(
				"P2P message %q has wrong receiver",
				msg.MID(),
			)
		}
	}
}

func TestHandlerLoadsStoredAttachment(t *testing.T) {
	store := newTestStore(t)

	msg := testOutboundMessage("message-1", "KR2SSY")
	if err := store.Save(msg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	source := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(
		source,
		[]byte("hello"),
		0o600,
	); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := store.AddAttachment(
		mailbox.FolderOutbox,
		msg.ID,
		source,
	); err != nil {
		t.Fatalf("AddAttachment() error = %v", err)
	}

	handler := NewHandler(store, "K2EXE")
	if err := handler.Prepare(); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	outbound := handler.GetOutbound()
	if len(outbound) != 1 {
		t.Fatalf("GetOutbound() count = %d, want 1", len(outbound))
	}

	files := outbound[0].Files()
	if len(files) != 1 {
		t.Fatalf("Files() count = %d, want 1", len(files))
	}

	if files[0].Name() != "notes.txt" {
		t.Fatalf("attachment name = %q", files[0].Name())
	}

	if string(files[0].Data()) != "hello" {
		t.Fatalf(
			"attachment data = %q",
			files[0].Data(),
		)
	}
}

func TestHandlerDefersInbound(t *testing.T) {
	store := newTestStore(t)
	handler := NewHandler(store, "K2EXE")

	if got := handler.GetInboundAnswer(
		fbb.Proposal{},
	); got != fbb.Defer {
		t.Fatalf(
			"GetInboundAnswer() = %v, want Defer",
			got,
		)
	}

	if err := handler.ProcessInbound(
		fbb.NewMessage(fbb.Private, "W2ABC"),
	); err == nil {
		t.Fatal("ProcessInbound() expected disabled error")
	}
}

func newTestStore(t *testing.T) *mailbox.Store {
	t.Helper()

	store := mailbox.NewStore(t.TempDir())

	if err := store.Prepare(); err != nil {
		t.Fatalf("Prepare store: %v", err)
	}

	return store
}

func testOutboundMessage(
	id string,
	to string,
) mailbox.Message {
	return mailbox.Message{
		ID:      id,
		Folder:  mailbox.FolderOutbox,
		From:    "K2EXE",
		To:      []string{to},
		Subject: "Test " + id,
		Body:    "Test body",
	}
}

func TestHandlerPrepareSkipsInvalidMessage(t *testing.T) {
	store := newTestStore(t)

	invalid := testOutboundMessage("invalid", "KR2SSY")
	invalid.Body = ""

	valid := testOutboundMessage("valid", "W2ABC")

	for _, msg := range []mailbox.Message{invalid, valid} {
		if err := store.Save(msg); err != nil {
			t.Fatalf("Save(%q) error = %v", msg.ID, err)
		}
	}

	handler := NewHandler(store, "K2EXE")

	if err := handler.Prepare(); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	if handler.Err() == nil {
		t.Fatal("Handler Err() expected invalid-message error")
	}

	outbound := handler.GetOutbound()
	if len(outbound) != 1 {
		t.Fatalf(
			"GetOutbound() count = %d, want 1",
			len(outbound),
		)
	}

	if outbound[0].Subject() != valid.Subject {
		t.Fatalf(
			"outbound subject = %q, want %q",
			outbound[0].Subject(),
			valid.Subject,
		)
	}

	if _, err := store.Load(
		mailbox.FolderOutbox,
		invalid.ID,
	); err != nil {
		t.Fatalf("invalid message was lost: %v", err)
	}
}
