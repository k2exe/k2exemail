package winlink

import (
	"bufio"
	"context"
	"fmt"
	"net"
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

type cmsDialFunc func(
	ctx context.Context,
	address string,
	callsign string,
	password string,
) (net.Conn, error)

func dialCMSTelnet(
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
				"CMS telnet login: %w",
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
					"send CMS callsign: %w",
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
					"send CMS telnet password: %w",
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
			return conn, nil
		}
	}
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
		dialCMSTelnet,
	)
}

func connectCMS(
	ctx context.Context,
	store *mailbox.Store,
	options CMSOptions,
	dial cmsDialFunc,
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
		options.Address = CMSProductionAddress
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
