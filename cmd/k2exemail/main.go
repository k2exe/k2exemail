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

	configPath := config.Path(dirs.Config)

	cfg, err := config.Load(
		configPath,
	)
	if err != nil {
		log.Fatalf(
			"load configuration: %v",
			err,
		)
	}

	runtimeConfig := config.NewRuntime(
		configPath,
		cfg,
	)

	store := mailbox.NewStore(dirs.Data)
	if err := store.Prepare(); err != nil {
		log.Fatalf(
			"prepare mailbox: %v",
			err,
		)
	}

	a := app.NewWithID(appID)

	updateIdentity := func(
		callsign string,
		locator string,
	) (string, string, error) {
		updated, err := runtimeConfig.UpdateIdentity(
			callsign,
			locator,
		)
		if err != nil {
			return "", "", err
		}

		return updated.Callsign,
			updated.Locator,
			nil
	}

	connectCMS := func(
		ctx context.Context,
		mode ui.CMSMode,
		password string,
	) (int, int, error) {
		current := runtimeConfig.Current()

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
				Callsign: current.Callsign,
				Locator:  current.Locator,
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
		func() (string, string) {
			current := runtimeConfig.Current()
			return current.Callsign, current.Locator
		},
		updateIdentity,
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
