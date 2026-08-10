package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"fyne.io/fyne/v2/app"
	"github.com/la5nta/wl2k-go/fbb"

	"github.com/k2exe/k2exemail/internal/appdirs"
	"github.com/k2exe/k2exemail/internal/config"
	"github.com/k2exe/k2exemail/internal/mailbox"
	"github.com/k2exe/k2exemail/internal/ui"
	"github.com/k2exe/k2exemail/internal/winlink"
)

const (
	appID      = "com.k2exe.k2exemail"
	appName    = "K2EXEmail"
	appVersion = "0.1.0"
)

func main() {
	dirs, err := appdirs.Default()
	if err != nil {
		log.Fatalf(
			"determine application directories: %v",
			err,
		)
	}

	cfg, err := config.Load(
		config.Path(dirs.Config),
	)
	if err != nil {
		log.Fatalf(
			"load configuration: %v",
			err,
		)
	}

	store := mailbox.NewStore(dirs.Data)
	if err := store.Prepare(); err != nil {
		log.Fatalf(
			"prepare mailbox: %v",
			err,
		)
	}

	a := app.NewWithID(appID)

	connectCMS := func(
		ctx context.Context,
		mode ui.CMSMode,
		password string,
	) (int, int, error) {
		address := winlink.CMSTestAddress

		switch mode {
		case ui.CMSModeTest:
		case ui.CMSModeProduction:
			address = winlink.CMSProductionAddress
		default:
			return 0, 0, fmt.Errorf(
				"unsupported CMS mode %q",
				mode,
			)
		}

		stats, err := winlink.ConnectCMS(
			ctx,
			store,
			winlink.CMSOptions{
				Address:  address,
				Callsign: cfg.Callsign,
				Locator:  cfg.Locator,
				UserAgent: fbb.UserAgent{
					Name:    appName,
					Version: appVersion,
				},
				SecureLogin: func(
					fbb.Address,
				) (string, error) {
					if password == "" {
						return "",
							errors.New(
								"Winlink secure login password is required",
							)
					}

					return password, nil
				},
			},
		)

		return len(stats.Sent),
			len(stats.Received),
			err
	}

	w, err := ui.NewMainWindow(
		a,
		appName,
		store,
		cfg.Callsign,
		cfg.Locator,
		connectCMS,
	)
	if err != nil {
		log.Fatalf(
			"create main window: %v",
			err,
		)
	}

	w.ShowAndRun()
}
