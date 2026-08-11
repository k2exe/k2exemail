package winlink

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

func TestConnectDirectExchangesOverDialedConnection(t *testing.T) {
	client, server := net.Pipe()
	remoteDone := startTestRemote(server)

	var gotAddress string

	dial := func(
		ctx context.Context,
		address string,
	) (net.Conn, error) {
		if ctx == nil {
			t.Fatal("dial context is nil")
		}

		gotAddress = address
		return client, nil
	}

	store := mailbox.NewStore(t.TempDir())

	stats, err := connectDirect(
		context.Background(),
		store,
		DirectOptions{
			Address:    "  peer.local.mesh:8772  ",
			Callsign:   " k2exe ",
			Locator:    "FN13",
			TargetCall: " wl2k ",
		},
		dial,
	)
	if err != nil {
		t.Fatalf("connectDirect() error = %v", err)
	}

	if gotAddress != "peer.local.mesh:8772" {
		t.Fatalf(
			"address = %q, want peer.local.mesh:8772",
			gotAddress,
		)
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
			t.Fatalf(
				"remote Exchange() error = %v",
				err,
			)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("remote session did not finish")
	}
}

func TestConnectDirectDialFailure(t *testing.T) {
	wantErr := errors.New("dial failed")

	dial := func(
		context.Context,
		string,
	) (net.Conn, error) {
		return nil, wantErr
	}

	store := mailbox.NewStore(t.TempDir())

	_, err := connectDirect(
		context.Background(),
		store,
		DirectOptions{
			Address:    "peer.local.mesh:8772",
			Callsign:   "K2EXE",
			Locator:    "FN13",
			TargetCall: "W2ABC",
		},
		dial,
	)

	if !errors.Is(err, wantErr) {
		t.Fatalf(
			"connectDirect() error = %v, want dial failure",
			err,
		)
	}

	if !strings.Contains(
		err.Error(),
		"connect direct TCP",
	) {
		t.Fatalf(
			"connectDirect() error = %v, want context",
			err,
		)
	}
}

func TestConnectDirectValidatesBeforeDial(t *testing.T) {
	called := false

	dial := func(
		context.Context,
		string,
	) (net.Conn, error) {
		called = true
		return nil, errors.New("unexpected dial")
	}

	store := mailbox.NewStore(t.TempDir())

	_, err := connectDirect(
		context.Background(),
		store,
		DirectOptions{
			Address:  "peer.local.mesh:8772",
			Locator:  "FN13",
			Callsign: "K2EXE",
			// TargetCall intentionally missing.
		},
		dial,
	)

	if err == nil {
		t.Fatal(
			"connectDirect() expected validation error",
		)
	}

	if called {
		t.Fatal("dialer called for invalid options")
	}
}

func TestConnectDirectRejectsNilDialer(t *testing.T) {
	store := mailbox.NewStore(t.TempDir())

	_, err := connectDirect(
		context.Background(),
		store,
		DirectOptions{
			Address:    "peer.local.mesh:8772",
			Callsign:   "K2EXE",
			Locator:    "FN13",
			TargetCall: "W2ABC",
		},
		nil,
	)

	if err == nil {
		t.Fatal(
			"connectDirect() expected dialer error",
		)
	}
}
