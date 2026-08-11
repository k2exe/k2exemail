package winlink

import (
	"context"
	"fmt"
	"strings"

	"github.com/k2exe/k2exemail/internal/mailbox"
	"github.com/la5nta/wl2k-go/fbb"
	"github.com/la5nta/wl2k-go/transport/telnet"
)

const (
	CMSProductionAddress = telnet.CMSAddress
	CMSTestAddress       = "cms-z.winlink.org:8772"
)

type CMSOptions struct {
	Address     string
	Callsign    string
	Locator     string
	UserAgent   fbb.UserAgent
	SecureLogin SecureLoginFunc
}

func ConnectCMS(
	ctx context.Context,
	store *mailbox.Store,
	options CMSOptions,
) (fbb.TrafficStats, error) {
	return connectCMS(
		ctx,
		store,
		options,
		dialWinlinkTelnet,
	)
}

func connectCMS(
	ctx context.Context,
	store *mailbox.Store,
	options CMSOptions,
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
			"CMS dialer is required",
		)
	}

	options.Address = strings.TrimSpace(options.Address)
	if options.Address == "" {
		return fbb.TrafficStats{}, fmt.Errorf(
			"CMS address is required",
		)
	}

	options.Callsign = strings.ToUpper(
		strings.TrimSpace(options.Callsign),
	)
	options.Locator = strings.TrimSpace(
		options.Locator,
	)

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

	conn, err := dial(
		ctx,
		options.Address,
		options.Callsign,
		telnet.CMSPassword,
	)
	if err != nil {
		return fbb.TrafficStats{}, fmt.Errorf(
			"connect CMS telnet: %w",
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
			TargetCall:  telnet.CMSTargetCall,
			UserAgent:   options.UserAgent,
			SecureLogin: options.SecureLogin,
		},
	)
}
