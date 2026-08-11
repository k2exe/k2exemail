package winlink

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
)

type telnetDialFunc func(
	ctx context.Context,
	address string,
	callsign string,
	password string,
) (net.Conn, error)

type bufferedReadConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *bufferedReadConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}

// dialWinlinkTelnet establishes TCP and performs the Winlink
// callsign/password Telnet login.
//
// A successfully returned connection is ready for an FBB exchange.
// The caller owns that connection.
func dialWinlinkTelnet(
	ctx context.Context,
	address string,
	callsign string,
	password string,
) (net.Conn, error) {
	var dialer net.Dialer

	conn, err := dialer.DialContext(
		ctx,
		"tcp",
		address,
	)
	if err != nil {
		return nil, err
	}

	keep := false

	stopCancel := context.AfterFunc(ctx, func() {
		_ = conn.Close()
	})

	defer func() {
		stopCancel()

		if !keep {
			_ = conn.Close()
		}
	}()

	reader := bufio.NewReader(conn)

	for {
		line, err := reader.ReadString('\r')
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}

			return nil, fmt.Errorf(
				"Winlink telnet login: %w",
				err,
			)
		}

		line = strings.TrimSpace(
			strings.ToLower(line),
		)

		switch {
		case strings.HasPrefix(line, "callsign"):
			if _, err := fmt.Fprintf(
				conn,
				"%s\r",
				callsign,
			); err != nil {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}

				return nil, fmt.Errorf(
					"send Winlink telnet callsign: %w",
					err,
				)
			}

		case strings.HasPrefix(line, "password"):
			if _, err := fmt.Fprintf(
				conn,
				"%s\r",
				password,
			); err != nil {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}

				return nil, fmt.Errorf(
					"send Winlink telnet password: %w",
					err,
				)
			}

			if ctx.Err() != nil {
				return nil, ctx.Err()
			}

			if !stopCancel() && ctx.Err() != nil {
				return nil, ctx.Err()
			}

			keep = true
			return &bufferedReadConn{
				Conn:   conn,
				reader: reader,
			}, nil
		}
	}
}
