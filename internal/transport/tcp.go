package transport

import (
	"context"
	"fmt"
	"net"
	"strings"
)

// DialTCP establishes a cancellable raw TCP connection.
//
// The caller owns a successfully returned connection.
func DialTCP(
	ctx context.Context,
	address string,
) (net.Conn, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is required")
	}

	address = strings.TrimSpace(address)
	if address == "" {
		return nil, fmt.Errorf("TCP address is required")
	}

	var dialer net.Dialer

	conn, err := dialer.DialContext(
		ctx,
		"tcp",
		address,
	)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		return nil, fmt.Errorf(
			"dial TCP %s: %w",
			address,
			err,
		)
	}

	return conn, nil
}
