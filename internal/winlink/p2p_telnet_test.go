package winlink

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/k2exe/k2exemail/internal/mailbox"
	"github.com/la5nta/wl2k-go/fbb"
	wl2ktelnet "github.com/la5nta/wl2k-go/transport/telnet"
)

func TestConnectP2PTelnetExchangesOverLoggedInConnection(
	t *testing.T,
) {
	client, server := net.Pipe()
	remoteDone := startP2PTestRemote(server)

	var gotAddress string
	var gotCallsign string
	var gotPassword string

	dial := func(
		ctx context.Context,
		address string,
		callsign string,
		password string,
	) (net.Conn, error) {
		if ctx == nil {
			t.Fatal("dial context is nil")
		}

		gotAddress = address
		gotCallsign = callsign
		gotPassword = password

		return client, nil
	}

	store := mailbox.NewStore(t.TempDir())

	stats, err := connectP2PTelnet(
		context.Background(),
		store,
		P2PTelnetOptions{
			Address:    "  w2abc.local.mesh:8774  ",
			Callsign:   " k2exe ",
			Locator:    "FN13",
			TargetCall: " w2abc ",
			Password:   "session-password",
		},
		dial,
	)
	if err != nil {
		t.Fatalf(
			"connectP2PTelnet() error = %v",
			err,
		)
	}

	if gotAddress != "w2abc.local.mesh:8774" {
		t.Fatalf(
			"address = %q, want w2abc.local.mesh:8774",
			gotAddress,
		)
	}
	if gotCallsign != "K2EXE" {
		t.Fatalf(
			"callsign = %q, want K2EXE",
			gotCallsign,
		)
	}
	if gotPassword != "session-password" {
		t.Fatalf(
			"password = %q, want session password",
			gotPassword,
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

func TestConnectP2PTelnetAllowsEmptyTransportPassword(
	t *testing.T,
) {
	wantErr := errors.New("stop after dial")
	var gotPassword string

	dial := func(
		_ context.Context,
		_ string,
		_ string,
		password string,
	) (net.Conn, error) {
		gotPassword = password
		return nil, wantErr
	}

	store := mailbox.NewStore(t.TempDir())

	_, err := connectP2PTelnet(
		context.Background(),
		store,
		P2PTelnetOptions{
			Address:    "w2abc.local.mesh:8774",
			Callsign:   "K2EXE",
			Locator:    "FN13",
			TargetCall: "W2ABC",
		},
		dial,
	)

	if !errors.Is(err, wantErr) {
		t.Fatalf(
			"connectP2PTelnet() error = %v, want dial error",
			err,
		)
	}
	if gotPassword != "" {
		t.Fatalf(
			"password = %q, want empty",
			gotPassword,
		)
	}
}

func TestConnectP2PTelnetDialFailure(t *testing.T) {
	wantErr := errors.New("dial failed")

	dial := func(
		context.Context,
		string,
		string,
		string,
	) (net.Conn, error) {
		return nil, wantErr
	}

	store := mailbox.NewStore(t.TempDir())

	_, err := connectP2PTelnet(
		context.Background(),
		store,
		P2PTelnetOptions{
			Address:    "w2abc.local.mesh:8774",
			Callsign:   "K2EXE",
			Locator:    "FN13",
			TargetCall: "W2ABC",
		},
		dial,
	)

	if !errors.Is(err, wantErr) {
		t.Fatalf(
			"connectP2PTelnet() error = %v, want dial failure",
			err,
		)
	}

	if !strings.Contains(
		err.Error(),
		"connect P2P telnet",
	) {
		t.Fatalf(
			"connectP2PTelnet() error = %v, want context",
			err,
		)
	}
}

func TestConnectP2PTelnetValidatesBeforeDial(t *testing.T) {
	called := false

	dial := func(
		context.Context,
		string,
		string,
		string,
	) (net.Conn, error) {
		called = true
		return nil, errors.New("unexpected dial")
	}

	store := mailbox.NewStore(t.TempDir())

	_, err := connectP2PTelnet(
		context.Background(),
		store,
		P2PTelnetOptions{
			Address:  "w2abc.local.mesh:8774",
			Callsign: "K2EXE",
			Locator:  "FN13",
			// TargetCall intentionally missing.
		},
		dial,
	)

	if err == nil {
		t.Fatal(
			"connectP2PTelnet() expected validation error",
		)
	}
	if called {
		t.Fatal("dialer called for invalid options")
	}
}

func startP2PTestRemote(conn net.Conn) <-chan error {
	done := make(chan error, 1)

	go func() {
		session := fbb.NewSession(
			"W2ABC",
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

func TestConnectP2PTelnetRejectsNilDialer(t *testing.T) {
	store := mailbox.NewStore(t.TempDir())

	_, err := connectP2PTelnet(
		context.Background(),
		store,
		P2PTelnetOptions{
			Address:    "w2abc.local.mesh:8774",
			Callsign:   "K2EXE",
			Locator:    "FN13",
			TargetCall: "W2ABC",
		},
		nil,
	)

	if err == nil {
		t.Fatal(
			"connectP2PTelnet() expected dialer error",
		)
	}
}

func TestConnectP2PTelnetInteroperatesWithWL2KListener(
	t *testing.T,
) {
	listener, err := wl2ktelnet.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("telnet.Listen() error = %v", err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}

		remote, ok := conn.(interface {
			RemoteCall() string
		})
		if !ok {
			_ = conn.Close()
			serverDone <- errors.New(
				"accepted telnet connection has no remote callsign",
			)
			return
		}

		if remote.RemoteCall() != "K2EXE" {
			_ = conn.Close()
			serverDone <- errors.New(
				"unexpected telnet remote callsign",
			)
			return
		}

		session := fbb.NewSession(
			"W2ABC",
			remote.RemoteCall(),
			"AA00aa",
			nil,
		)
		session.IsMaster(true)

		_, err = session.Exchange(conn)
		serverDone <- err
	}()

	store := mailbox.NewStore(t.TempDir())

	stats, err := ConnectP2PTelnet(
		context.Background(),
		store,
		P2PTelnetOptions{
			Address:    listener.Addr().String(),
			Callsign:   "K2EXE",
			Locator:    "FN13",
			TargetCall: "W2ABC",
			Password:   "test-password",
		},
	)
	if err != nil {
		t.Fatalf(
			"ConnectP2PTelnet() error = %v",
			err,
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
	case err := <-serverDone:
		if err != nil {
			t.Fatalf(
				"wl2k telnet server error = %v",
				err,
			)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("wl2k telnet server did not finish")
	}
}
