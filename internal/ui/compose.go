package ui

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

func openComposeWindow(
	a fyne.App,
	parent fyne.Window,
	store mailboxStore,
	callsign string,
	onChanged func(),
) {
	draft, err := mailbox.NewDraft()
	if err != nil {
		dialog.ShowError(err, parent)
		return
	}

	openComposeMessage(
		a,
		parent,
		store,
		draft,
		false,
		callsign,
		onChanged,
	)
}

func openDraftWindow(
	a fyne.App,
	parent fyne.Window,
	store mailboxStore,
	draft mailbox.Message,
	callsign string,
	onChanged func(),
) {
	if draft.Folder != mailbox.FolderDrafts {
		dialog.ShowError(
			fmt.Errorf("message %q is not a draft", draft.ID),
			parent,
		)
		return
	}

	openComposeMessage(
		a,
		parent,
		store,
		draft,
		true,
		callsign,
		onChanged,
	)
}

func openComposeMessage(
	a fyne.App,
	parent fyne.Window,
	store mailboxStore,
	draft mailbox.Message,
	persisted bool,
	callsign string,
	onChanged func(),
) {
	title := "New Message — K2EXEmail"
	if persisted {
		title = "Edit Draft — K2EXEmail"
	}

	w := a.NewWindow(title)
	w.Resize(fyne.NewSize(800, 650))

	to := widget.NewEntry()
	to.SetPlaceHolder("Callsign or email address")
	to.SetText(strings.Join(draft.To, ", "))

	cc := widget.NewEntry()
	cc.SetPlaceHolder("Optional")
	cc.SetText(strings.Join(draft.Cc, ", "))

	subject := widget.NewEntry()
	subject.SetPlaceHolder("Subject")
	subject.SetText(draft.Subject)

	body := widget.NewMultiLineEntry()
	body.SetPlaceHolder("Write your message")
	body.SetText(draft.Body)

	header := widget.NewForm(
		widget.NewFormItem("To", to),
		widget.NewFormItem("Cc", cc),
		widget.NewFormItem("Subject", subject),
	)

	status := widget.NewLabel("")

	var busy bool

	saveButton := widget.NewButtonWithIcon(
		"Save & Close",
		theme.DocumentSaveIcon(),
		nil,
	)

	queueButton := widget.NewButtonWithIcon(
		"Queue to Outbox",
		theme.MailSendIcon(),
		nil,
	)
	queueButton.Importance = widget.HighImportance

	setBusy := func(value bool) {
		busy = value

		if value {
			saveButton.Disable()
			queueButton.Disable()
		} else {
			saveButton.Enable()
			queueButton.Enable()
		}
	}

	snapshot := func() mailbox.Message {
		return composeSnapshot(
			draft,
			callsign,
			to.Text,
			cc.Text,
			subject.Text,
			body.Text,
			time.Now().UTC(),
		)
	}

	closeWindow := func() {
		w.SetCloseIntercept(nil)
		w.Close()
	}

	notifyChanged := func() {
		if onChanged != nil {
			onChanged()
		}
	}

	saveDraft := func(closeAfter bool) {
		if busy {
			return
		}

		msg := snapshot()

		// A brand-new untouched compose window should not create an
		// empty draft just because the user closed it.
		if closeAfter && !persisted && !messageHasContent(msg) {
			closeWindow()
			return
		}

		setBusy(true)
		status.SetText("Saving draft...")

		go func() {
			err := store.Save(msg)

			fyne.Do(func() {
				if err != nil {
					setBusy(false)
					status.SetText("Draft not saved")
					dialog.ShowError(
						fmt.Errorf("save draft: %w", err),
						w,
					)
					return
				}

				draft = msg
				persisted = true
				notifyChanged()

				if closeAfter {
					closeWindow()
					return
				}

				setBusy(false)
				status.SetText("Draft saved")
			})
		}()
	}

	saveButton.OnTapped = func() {
		saveDraft(true)
	}

	queueButton.OnTapped = func() {
		if busy {
			return
		}

		msg := snapshot()

		if err := validateQueueMessage(msg); err != nil {
			dialog.ShowError(err, w)
			return
		}

		setBusy(true)
		status.SetText("Queueing message...")

		go func() {
			err := store.Save(msg)
			if err == nil {
				err = store.Move(
					mailbox.FolderDrafts,
					mailbox.FolderOutbox,
					msg.ID,
				)
			}

			fyne.Do(func() {
				if err != nil {
					setBusy(false)
					status.SetText("Message not queued")
					dialog.ShowError(
						fmt.Errorf("queue message: %w", err),
						w,
					)
					return
				}

				notifyChanged()
				closeWindow()

				dialog.ShowInformation(
					"Message queued",
					"Message saved to Outbox.",
					parent,
				)
			})
		}()
	}

	w.SetCloseIntercept(func() {
		if busy {
			status.SetText("Please wait for the current operation.")
			return
		}

		saveDraft(true)
	})

	actions := container.NewHBox(
		queueButton,
		saveButton,
		status,
	)

	w.SetContent(
		container.NewBorder(
			container.NewVBox(
				header,
				widget.NewSeparator(),
			),
			actions,
			nil,
			nil,
			body,
		),
	)

	w.Show()
}

func composeSnapshot(
	base mailbox.Message,
	callsign string,
	to string,
	cc string,
	subject string,
	body string,
	updatedAt time.Time,
) mailbox.Message {
	base.Folder = mailbox.FolderDrafts
	base.From = strings.ToUpper(strings.TrimSpace(callsign))
	base.To = splitRecipients(to)
	base.Cc = splitRecipients(cc)
	base.Subject = strings.TrimSpace(subject)
	base.Body = body
	base.UpdatedAt = updatedAt

	return base
}

func splitRecipients(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';'
	})

	recipients := make([]string, 0, len(fields))

	for _, field := range fields {
		if recipient := strings.TrimSpace(field); recipient != "" {
			recipients = append(recipients, recipient)
		}
	}

	return recipients
}

func messageHasContent(msg mailbox.Message) bool {
	return len(msg.To) > 0 ||
		len(msg.Cc) > 0 ||
		strings.TrimSpace(msg.Subject) != "" ||
		strings.TrimSpace(msg.Body) != "" ||
		len(msg.Attachments) > 0
}

func validateQueueMessage(msg mailbox.Message) error {
	switch {
	case len(msg.To) == 0:
		return fmt.Errorf("at least one recipient is required")
	case strings.TrimSpace(msg.Subject) == "":
		return fmt.Errorf("subject is required")
	case strings.TrimSpace(msg.Body) == "":
		return fmt.Errorf("message body is required")
	default:
		return nil
	}
}
