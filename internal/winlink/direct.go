package winlink

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/k2exe/k2exemail/internal/mailbox"
	apptransport "github.com/k2exe/k2exemail/internal/transport"
	"github.com/la5nta/wl2k-go/fbb"
)

type DirectOptions struct {
	Address     string
	Callsign    string
	Locator     string
	TargetCall  string
	UserAgent   fbb.UserAgent
	SecureLogin SecureLoginFunc
}

type directDialFunc func(
	ctx context.Context,
	address string,
) (net.Conn, error)

func ConnectDirect(
	ctx context.Context,
	store *mailbox.Store,
	options DirectOptions,
) (fbb.TrafficStats, error) {
	return connectDirect(
		ctx,
		store,
		options,
		apptransport.DialTCP,
	)
}

func connectDirect(
	ctx context.Context,
	store *mailbox.Store,
	options DirectOptions,
	dial directDialFunc,
) (fbb.TrafficStats, error) {
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
	if dial == nil {
		return fbb.TrafficStats{}, fmt.Errorf(
			"direct TCP dialer is required",
		)
	}

	options.Address = strings.TrimSpace(options.Address)
	options.Callsign = strings.ToUpper(
		strings.TrimSpace(options.Callsign),
	)
	options.Locator = strings.TrimSpace(options.Locator)
	options.TargetCall = strings.ToUpper(
		strings.TrimSpace(options.TargetCall),
	)

	if options.Address == "" {
		return fbb.TrafficStats{}, fmt.Errorf(
			"direct TCP address is required",
		)
	}
	if options.Callsign == "" {
		return fbb.TrafficStats{}, fmt.Errorf(
			"station callsign is required",
		)
	}
	if options.Locator == "" {
		return fbb.TrafficStats{}, fmt.Errorf(
			"station locator is required",
		)
	}
	if options.TargetCall == "" {
		return fbb.TrafficStats{}, fmt.Errorf(
			"target callsign is required",
		)
	}

	conn, err := dial(
		ctx,
		options.Address,
	)
	if err != nil {
		return fbb.TrafficStats{}, fmt.Errorf(
			"connect direct TCP: %w",
			err,
		)
	}

	return Exchange(
		ctx,
		conn,
		store,
		ExchangeOptions{
			Callsign:    options.Callsign,
			Locator:     options.Locator,
			TargetCall:  options.TargetCall,
			Master:      false,
			UserAgent:   options.UserAgent,
			SecureLogin: options.SecureLogin,
		},
	)
}
