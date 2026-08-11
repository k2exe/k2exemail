package winlink

import (
	"bufio"
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/k2exe/k2exemail/internal/mailbox"
	"github.com/la5nta/wl2k-go/transport/telnet"
)

func TestConnectCMSUsesWinlinkTelnetSettings(t *testing.T) {
	client, server := net.Pipe()
	remoteDone := startTestRemote(server)

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

	stats, err := connectCMS(
		context.Background(),
		store,
		CMSOptions{
			Address:  CMSProductionAddress,
			Callsign: " k2exe ",
			Locator:  "FN13",
		},
		dial,
	)
	if err != nil {
		t.Fatalf("connectCMS() error = %v", err)
	}

	if gotAddress != telnet.CMSAddress {
		t.Fatalf(
			"address = %q, want %q",
			gotAddress,
			telnet.CMSAddress,
		)
	}
	if gotCallsign != "K2EXE" {
		t.Fatalf(
			"callsign = %q, want K2EXE",
			gotCallsign,
		)
	}
	if gotPassword != telnet.CMSPassword {
		t.Fatalf(
			"password = %q, want CMS password",
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

func TestConnectCMSDialFailure(t *testing.T) {
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

	_, err := connectCMS(
		context.Background(),
		store,
		CMSOptions{
			Address:  CMSProductionAddress,
			Callsign: "K2EXE",
			Locator:  "FN13",
		},
		dial,
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf(
			"connectCMS() error = %v, want dial failure",
			err,
		)
	}

	if !strings.Contains(
		err.Error(),
		"connect CMS telnet",
	) {
		t.Fatalf(
			"connectCMS() error = %v, want context",
			err,
		)
	}
}

func TestConnectCMSValidatesBeforeDial(t *testing.T) {
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

	_, err := connectCMS(
		context.Background(),
		store,
		CMSOptions{
			Address: CMSProductionAddress,
			Locator: "FN13",
		},
		dial,
	)
	if err == nil {
		t.Fatal("connectCMS() expected validation error")
	}

	if called {
		t.Fatal("dialer called for invalid options")
	}
}

func TestDialWinlinkTelnetCancellationDuringLogin(t *testing.T) {
	listener, err := net.Listen(
		"tcp",
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	defer listener.Close()

	accepted := make(chan net.Conn, 1)

	go func() {
		conn, err := listener.Accept()
		if err == nil {
			accepted <- conn
		}
	}()

	ctx, cancel := context.WithCancel(
		context.Background(),
	)

	done := make(chan error, 1)

	go func() {
		conn, err := dialWinlinkTelnet(
			ctx,
			listener.Addr().String(),
			"K2EXE",
			telnet.CMSPassword,
		)
		if conn != nil {
			_ = conn.Close()
		}

		done <- err
	}()

	var server net.Conn

	select {
	case server = <-accepted:
		defer server.Close()
	case <-time.After(2 * time.Second):
		t.Fatal("client did not connect")
	}

	// Deliberately send no login prompt.
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf(
				"dialWinlinkTelnet() error = %v, want context.Canceled",
				err,
			)
		}
	case <-time.After(2 * time.Second):
		t.Fatal(
			"telnet login did not stop after cancellation",
		)
	}
}

func TestDialWinlinkTelnetLogin(t *testing.T) {
	listener, err := net.Listen(
		"tcp",
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)

		if _, err := conn.Write([]byte("Callsign :\r")); err != nil {
			serverDone <- err
			return
		}

		callsign, err := reader.ReadString('\r')
		if err != nil {
			serverDone <- err
			return
		}

		if strings.TrimSpace(callsign) != "K2EXE" {
			serverDone <- errors.New("unexpected callsign")
			return
		}

		if _, err := conn.Write([]byte("Password :\r")); err != nil {
			serverDone <- err
			return
		}

		password, err := reader.ReadString('\r')
		if err != nil {
			serverDone <- err
			return
		}

		if strings.TrimSpace(password) != telnet.CMSPassword {
			serverDone <- errors.New("unexpected password")
			return
		}

		serverDone <- nil
	}()

	conn, err := dialWinlinkTelnet(
		context.Background(),
		listener.Addr().String(),
		"K2EXE",
		telnet.CMSPassword,
	)
	if err != nil {
		t.Fatalf("dialWinlinkTelnet() error = %v", err)
	}
	defer conn.Close()

	select {
	case err := <-serverDone:
		if err != nil {
			t.Fatalf("server login error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("CMS telnet login did not finish")
	}
}

func TestConnectCMSUsesConfiguredAddress(t *testing.T) {
	wantErr := errors.New("stop after dial")
	var gotAddress string

	dial := func(
		ctx context.Context,
		address string,
		callsign string,
		password string,
	) (net.Conn, error) {
		gotAddress = address
		return nil, wantErr
	}

	store := mailbox.NewStore(t.TempDir())

	_, err := connectCMS(
		context.Background(),
		store,
		CMSOptions{
			Address:  CMSTestAddress,
			Callsign: "K2EXE",
			Locator:  "FN23va",
		},
		dial,
	)

	if !errors.Is(err, wantErr) {
		t.Fatalf(
			"connectCMS() error = %v, want %v",
			err,
			wantErr,
		)
	}

	if gotAddress != CMSTestAddress {
		t.Fatalf(
			"address = %q, want %q",
			gotAddress,
			CMSTestAddress,
		)
	}
}

func TestConnectCMSRequiresExplicitAddress(t *testing.T) {
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

	_, err := connectCMS(
		context.Background(),
		store,
		CMSOptions{
			Callsign: "K2EXE",
			Locator:  "FN23va",
		},
		dial,
	)
	if err == nil {
		t.Fatal("connectCMS() expected missing-address error")
	}

	if !strings.Contains(err.Error(), "CMS address is required") {
		t.Fatalf("connectCMS() error = %v", err)
	}

	if called {
		t.Fatal("dialer called without explicit CMS address")
	}
}
