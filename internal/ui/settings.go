package ui

import (
	"errors"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type IdentityUpdateFunc func(
	callsign string,
	locator string,
) (
	savedCallsign string,
	savedLocator string,
	err error,
)

func newSettingsWindow(
	a fyne.App,
	callsign string,
	locator string,
	updateIdentity IdentityUpdateFunc,
	onClosed func(),
) fyne.Window {
	w := a.NewWindow("Settings")

	header := widget.NewLabelWithStyle(
		"Station Identity",
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	description := widget.NewLabel(
		"Winlink identity used for new messages and connections.",
	)
	description.Wrapping = fyne.TextWrapWord

	callsignEntry := widget.NewEntry()
	callsignEntry.SetPlaceHolder("K2EXE")
	callsignEntry.SetText(callsign)

	locatorEntry := widget.NewEntry()
	locatorEntry.SetPlaceHolder("FN23va")
	locatorEntry.SetText(locator)

	form := widget.NewForm(
		widget.NewFormItem(
			"Callsign",
			callsignEntry,
		),
		widget.NewFormItem(
			"Maidenhead locator",
			locatorEntry,
		),
	)

	status := widget.NewLabel("")
	status.Wrapping = fyne.TextWrapWord

	saveButton := widget.NewButton(
		"Save",
		nil,
	)
	saveButton.Importance = widget.HighImportance

	saveButton.OnTapped = func() {
		nextCallsign := callsignEntry.Text
		nextLocator := locatorEntry.Text

		if err := validateIdentitySettings(
			nextCallsign,
			nextLocator,
		); err != nil {
			status.SetText(err.Error())
			return
		}

		if updateIdentity == nil {
			status.SetText(
				"Configuration updates are unavailable.",
			)
			return
		}

		callsignEntry.Disable()
		locatorEntry.Disable()
		saveButton.Disable()
		status.SetText("Saving...")

		go func(
			callsign string,
			locator string,
		) {
			savedCallsign, savedLocator, err :=
				updateIdentity(
					callsign,
					locator,
				)

			fyne.Do(func() {
				callsignEntry.Enable()
				locatorEntry.Enable()
				saveButton.Enable()

				if err != nil {
					status.SetText(
						"Failed: " + err.Error(),
					)
					return
				}

				callsignEntry.SetText(savedCallsign)
				locatorEntry.SetText(savedLocator)

				status.SetText(
					"Saved. New Compose and Connections windows will use this identity.",
				)
			})
		}(nextCallsign, nextLocator)
	}

	w.SetOnClosed(func() {
		if onClosed != nil {
			onClosed()
		}
	})

	content := container.NewVBox(
		header,
		description,
		widget.NewSeparator(),
		form,
		saveButton,
		status,
	)

	w.SetContent(
		container.NewPadded(content),
	)
	w.Resize(fyne.NewSize(480, 280))

	return w
}

func validateIdentitySettings(
	callsign string,
	locator string,
) error {
	if strings.TrimSpace(callsign) == "" {
		return errors.New(
			"Callsign is required.",
		)
	}

	if strings.TrimSpace(locator) == "" {
		return errors.New(
			"Maidenhead locator is required.",
		)
	}

	return nil
}
