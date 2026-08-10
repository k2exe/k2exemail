package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type CMSMode string

const (
	CMSModeTest       CMSMode = "Test CMS"
	CMSModeProduction CMSMode = "Production CMS"
)

type CMSConnectFunc func(
	ctx context.Context,
	mode CMSMode,
	securePassword string,
) (sent int, received int, err error)

func newConnectionsWindow(
	a fyne.App,
	callsign string,
	locator string,
	connectCMS CMSConnectFunc,
	active *atomic.Bool,
	onMailboxChanged func(),
	onClosed func(),
) fyne.Window {
	w := a.NewWindow("Connections")

	if active == nil {
		active = &atomic.Bool{}
	}

	header := widget.NewLabelWithStyle(
		"Winlink CMS",
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	description := widget.NewLabel(
		"Internet / CMS Telnet",
	)

	serverNote := widget.NewLabel(
		cmsModeDescription(CMSModeTest),
	)
	serverNote.Wrapping = fyne.TextWrapWord

	server := widget.NewSelect(
		[]string{
			string(CMSModeTest),
			string(CMSModeProduction),
		},
		func(value string) {
			serverNote.SetText(
				cmsModeDescription(CMSMode(value)),
			)
		},
	)
	server.SetSelected(string(CMSModeTest))

	callsignValue := strings.TrimSpace(callsign)
	locatorValue := strings.TrimSpace(locator)

	if callsignValue == "" {
		callsignValue = "(not configured)"
	}
	if locatorValue == "" {
		locatorValue = "(not configured)"
	}

	identity := widget.NewForm(
		widget.NewFormItem(
			"Callsign",
			widget.NewLabel(callsignValue),
		),
		widget.NewFormItem(
			"Locator",
			widget.NewLabel(locatorValue),
		),
	)

	password := widget.NewPasswordEntry()
	password.SetPlaceHolder(
		"Winlink secure login password",
	)

	status := widget.NewLabel("Ready")
	status.Wrapping = fyne.TextWrapWord

	connectButton := widget.NewButton(
		"Connect",
		nil,
	)
	connectButton.Importance = widget.HighImportance

	cancelButton := widget.NewButton(
		"Cancel",
		nil,
	)
	cancelButton.Disable()

	var currentCancel context.CancelFunc

	connectButton.OnTapped = func() {
		if connectCMS == nil {
			status.SetText(
				"CMS connection is unavailable.",
			)
			return
		}

		if strings.TrimSpace(callsign) == "" ||
			strings.TrimSpace(locator) == "" {
			status.SetText(
				"Configure a callsign and locator before connecting.",
			)
			return
		}

		mode := CMSMode(server.Selected)
		if mode != CMSModeTest &&
			mode != CMSModeProduction {
			status.SetText("Select a CMS server.")
			return
		}

		secret := password.Text
		if strings.TrimSpace(secret) == "" {
			status.SetText(
				"Enter your Winlink secure login password.",
			)
			return
		}

		if !active.CompareAndSwap(false, true) {
			status.SetText(
				"A CMS session is already running.",
			)
			return
		}

		ctx, cancel := context.WithCancel(
			context.Background(),
		)
		currentCancel = cancel

		// Do not retain the password in the visible widget.
		password.SetText("")
		password.Disable()
		server.Disable()

		connectButton.Disable()
		cancelButton.Enable()

		status.SetText(
			"Connecting to Winlink CMS and exchanging mail...",
		)

		go func(mode CMSMode, secret string) {
			sent, received, err := connectCMS(
				ctx,
				mode,
				secret,
			)

			// Drop our reference as soon as the exchange returns.
			secret = ""

			fyne.Do(func() {
				active.Store(false)
				currentCancel = nil

				cancelButton.Disable()
				password.Enable()
				server.Enable()
				connectButton.Enable()

				if onMailboxChanged != nil {
					onMailboxChanged()
				}

				switch {
				case errors.Is(err, context.Canceled):
					status.SetText("Cancelled")

				case err != nil:
					status.SetText(
						"Failed: " + err.Error(),
					)

				default:
					status.SetText(
						connectionResultText(
							sent,
							received,
						),
					)
				}
			})
		}(mode, secret)
	}

	cancelButton.OnTapped = func() {
		if currentCancel == nil {
			return
		}

		status.SetText("Cancelling...")
		cancelButton.Disable()
		currentCancel()
	}

	w.SetOnClosed(func() {
		if currentCancel != nil {
			currentCancel()
		}

		if onClosed != nil {
			onClosed()
		}
	})

	buttons := container.NewHBox(
		connectButton,
		cancelButton,
	)

	content := container.NewVBox(
		header,
		description,
		widget.NewSeparator(),
		widget.NewLabel("Server"),
		server,
		serverNote,
		widget.NewSeparator(),
		identity,
		widget.NewSeparator(),
		widget.NewLabel(
			"Secure login password",
		),
		password,
		buttons,
		widget.NewSeparator(),
		status,
	)

	w.SetContent(
		container.NewPadded(content),
	)
	w.Resize(fyne.NewSize(480, 340))

	return w
}

func cmsModeDescription(mode CMSMode) string {
	switch mode {
	case CMSModeTest:
		return "Development and interoperability testing."

	case CMSModeProduction:
		return "Production currently rejects the unregistered K2EXEmail client type."

	default:
		return ""
	}
}

func connectionResultText(
	sent int,
	received int,
) string {
	return fmt.Sprintf(
		"Complete - %d sent, %d received",
		sent,
		received,
	)
}
