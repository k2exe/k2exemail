package transport

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestDialTCPConnects(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	defer listener.Close()

	accepted := make(chan net.Conn, 1)
	acceptErr := make(chan error, 1)

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			acceptErr <- err
			return
		}

		accepted <- conn
	}()

	conn, err := DialTCP(
		context.Background(),
		listener.Addr().String(),
	)
	if err != nil {
		t.Fatalf("DialTCP() error = %v", err)
	}
	defer conn.Close()

	select {
	case peer := <-accepted:
		peer.Close()

	case err := <-acceptErr:
		t.Fatalf("Accept() error = %v", err)
	}
}

func TestDialTCPRejectsNilContext(t *testing.T) {
	conn, err := DialTCP(nil, "127.0.0.1:8772")

	if conn != nil {
		conn.Close()
		t.Fatal("DialTCP() returned connection with nil context")
	}
	if err == nil {
		t.Fatal("DialTCP() expected context error")
	}
}

func TestDialTCPRejectsEmptyAddress(t *testing.T) {
	conn, err := DialTCP(
		context.Background(),
		"   ",
	)

	if conn != nil {
		conn.Close()
		t.Fatal("DialTCP() returned connection for empty address")
	}
	if err == nil {
		t.Fatal("DialTCP() expected address error")
	}
}

func TestDialTCPCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	conn, err := DialTCP(
		ctx,
		"127.0.0.1:8772",
	)

	if conn != nil {
		conn.Close()
		t.Fatal("DialTCP() returned connection after cancellation")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"DialTCP() error = %v, want context.Canceled",
			err,
		)
	}
}
