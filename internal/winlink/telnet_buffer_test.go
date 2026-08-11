package winlink

import (
	"bufio"
	"io"
	"net"
	"testing"
)

func TestBufferedReadConnPreservesBufferedBytes(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	const wire = "Password :\rFBB"
	const want = "FBB"

	writeDone := make(chan error, 1)

	go func() {
		_, err := io.WriteString(server, wire)
		writeDone <- err
	}()

	reader := bufio.NewReader(client)

	line, err := reader.ReadString('\r')
	if err != nil {
		t.Fatalf("ReadString() error = %v", err)
	}
	if line != "Password :\r" {
		t.Fatalf(
			"login prompt = %q, want %q",
			line,
			"Password :\r",
		)
	}

	conn := &bufferedReadConn{
		Conn:   client,
		reader: reader,
	}

	got := make([]byte, len(want))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("ReadFull() error = %v", err)
	}

	if string(got) != want {
		t.Fatalf(
			"buffered bytes = %q, want %q",
			got,
			want,
		)
	}

	if err := <-writeDone; err != nil {
		t.Fatalf("server WriteString() error = %v", err)
	}
}
