package winlink

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/k2exe/k2exemail/internal/mailbox"
	"github.com/la5nta/wl2k-go/fbb"
)

func TestExchangeEmptySession(t *testing.T) {
	client, server := net.Pipe()

	remoteDone := startTestRemote(server)

	store := mailbox.NewStore(t.TempDir())

	stats, err := Exchange(
		context.Background(),
		client,
		store,
		ExchangeOptions{
			Callsign:   "k2exe",
			Locator:    "FN13",
			TargetCall: "wl2k",
		},
	)
	if err != nil {
		t.Fatalf("Exchange() error = %v", err)
	}

	if len(stats.Sent) != 0 {
		t.Fatalf("Sent = %#v, want empty", stats.Sent)
	}
	if len(stats.Received) != 0 {
		t.Fatalf(
			"Received = %#v, want empty",
			stats.Received,
		)
	}

	select {
	case err := <-remoteDone:
		if err != nil {
			t.Fatalf("remote Exchange() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("remote session did not finish")
	}
}

func TestExchangeCancellation(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	cancel()

	store := mailbox.NewStore(t.TempDir())

	_, err := Exchange(
		ctx,
		client,
		store,
		ExchangeOptions{
			Callsign:   "K2EXE",
			Locator:    "FN13",
			TargetCall: "WL2K",
		},
	)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"Exchange() error = %v, want context.Canceled",
			err,
		)
	}
}

func TestExchangeSurfacesMailboxError(t *testing.T) {
	client, server := net.Pipe()

	remoteDone := startTestRemote(server)

	store := mailbox.NewStore(t.TempDir())
	if err := store.Prepare(); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	msg, err := mailbox.NewMessage(
		mailbox.FolderOutbox,
	)
	if err != nil {
		t.Fatalf("NewMessage() error = %v", err)
	}

	msg.From = "K2EXE"
	msg.To = []string{"W2ABC"}
	msg.Subject = "Invalid queued message"
	// Body intentionally empty.

	if err := store.Save(msg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	_, err = Exchange(
		context.Background(),
		client,
		store,
		ExchangeOptions{
			Callsign:   "K2EXE",
			Locator:    "FN13",
			TargetCall: "WL2K",
		},
	)
	if err == nil {
		t.Fatal("Exchange() expected mailbox error")
	}

	if !strings.Contains(err.Error(), "Empty body") {
		t.Fatalf(
			"Exchange() error = %v, want Empty body",
			err,
		)
	}

	select {
	case remoteErr := <-remoteDone:
		if remoteErr != nil {
			t.Fatalf(
				"remote Exchange() error = %v",
				remoteErr,
			)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("remote session did not finish")
	}
}

func startTestRemote(
	conn net.Conn,
) <-chan error {
	done := make(chan error, 1)

	go func() {
		session := fbb.NewSession(
			"WL2K",
			"K2EXE",
			"AA00aa",
			nil,
		)
		session.IsMaster(true)

		_, err := session.Exchange(conn)
		done <- err
	}()

	return done
}

func TestExchangeClosesConnectionOnValidationError(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()

	store := mailbox.NewStore(t.TempDir())

	_, err := Exchange(
		context.Background(),
		client,
		store,
		ExchangeOptions{
			// Callsign intentionally missing.
			Locator:    "FN13",
			TargetCall: "WL2K",
		},
	)
	if err == nil {
		t.Fatal("Exchange() expected validation error")
	}

	_ = server.SetReadDeadline(
		time.Now().Add(time.Second),
	)

	var buf [1]byte
	_, readErr := server.Read(buf[:])

	if !errors.Is(readErr, net.ErrClosed) &&
		!errors.Is(readErr, io.EOF) {
		t.Fatalf(
			"remote read error = %v, want closed connection",
			readErr,
		)
	}
}
