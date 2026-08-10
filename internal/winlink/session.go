package winlink

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"github.com/k2exe/k2exemail/internal/mailbox"
	"github.com/la5nta/wl2k-go/fbb"
)

type SecureLoginFunc func(
	address fbb.Address,
) (string, error)

type ExchangeOptions struct {
	Callsign    string
	Locator     string
	TargetCall  string
	Master      bool
	UserAgent   fbb.UserAgent
	SecureLogin SecureLoginFunc
	Logger      *log.Logger
}

func Exchange(
	ctx context.Context,
	conn net.Conn,
	store *mailbox.Store,
	options ExchangeOptions,
) (fbb.TrafficStats, error) {
	if conn == nil {
		return fbb.TrafficStats{}, fmt.Errorf(
			"connection is required",
		)
	}

	// Exchange owns a non-nil connection from this point forward.
	// fbb.Session.Exchange also closes it; a second Close is harmless
	// and ensures early validation failures cannot leak a transport.
	defer conn.Close()

	if ctx == nil {
		return fbb.TrafficStats{}, fmt.Errorf(
			"context is required",
		)
	}
	if store == nil {
		return fbb.TrafficStats{}, fmt.Errorf(
			"mailbox store is required",
		)
	}

	options.Callsign = strings.ToUpper(
		strings.TrimSpace(options.Callsign),
	)
	options.TargetCall = strings.ToUpper(
		strings.TrimSpace(options.TargetCall),
	)
	options.Locator = strings.TrimSpace(options.Locator)

	if options.Callsign == "" {
		return fbb.TrafficStats{}, fmt.Errorf(
			"station callsign is required",
		)
	}
	if options.TargetCall == "" {
		return fbb.TrafficStats{}, fmt.Errorf(
			"target callsign is required",
		)
	}
	if options.Locator == "" {
		return fbb.TrafficStats{}, fmt.Errorf(
			"station locator is required",
		)
	}

	if (options.UserAgent.Name == "") !=
		(options.UserAgent.Version == "") {
		return fbb.TrafficStats{}, fmt.Errorf(
			"user agent name and version must be set together",
		)
	}

	handler := NewHandler(store, options.Callsign)

	session := fbb.NewSession(
		options.Callsign,
		options.TargetCall,
		options.Locator,
		handler,
	)

	session.IsMaster(options.Master)

	logger := options.Logger
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	session.SetLogger(logger)

	if options.UserAgent.Name != "" {
		session.SetUserAgent(options.UserAgent)
	}

	if options.SecureLogin != nil {
		session.SetSecureLoginHandleFunc(
			func(address fbb.Address) (string, error) {
				return options.SecureLogin(address)
			},
		)
	}

	// fbb.Session.Exchange does not take a context. Closing the
	// transport is its documented cancellation mechanism, so arrange
	// for cancellation to close the connection.
	stopCancel := context.AfterFunc(ctx, func() {
		_ = conn.Close()
	})

	stats, exchangeErr := session.Exchange(conn)
	stopCancel()

	// Session.Exchange reports a locally closed connection as
	// ErrConnLost. Preserve the caller's cancellation reason when the
	// context caused that close.
	if exchangeErr != nil && ctx.Err() != nil {
		exchangeErr = ctx.Err()
	}

	return stats, errors.Join(
		exchangeErr,
		handler.Err(),
	)
}
