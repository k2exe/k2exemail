package winlink

import (
	"context"
	"fmt"
	"strings"

	"github.com/k2exe/k2exemail/internal/mailbox"
	"github.com/la5nta/wl2k-go/fbb"
)

type P2PTelnetOptions struct {
	Address    string
	Callsign   string
	Locator    string
	TargetCall string

	// Password is the Telnet transport-login password.
	// It is distinct from Winlink FBB secure login.
	Password string

	UserAgent   fbb.UserAgent
	SecureLogin SecureLoginFunc
}

func ConnectP2PTelnet(
	ctx context.Context,
	store *mailbox.Store,
	options P2PTelnetOptions,
) (fbb.TrafficStats, error) {
	return connectP2PTelnet(
		ctx,
		store,
		options,
		dialWinlinkTelnet,
	)
}

func connectP2PTelnet(
	ctx context.Context,
	store *mailbox.Store,
	options P2PTelnetOptions,
	dial telnetDialFunc,
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
			"P2P telnet dialer is required",
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
			"P2P telnet address is required",
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
		options.Callsign,
		options.Password,
	)
	if err != nil {
		return fbb.TrafficStats{}, fmt.Errorf(
			"connect P2P telnet: %w",
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
