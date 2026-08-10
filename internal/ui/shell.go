package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

func newMailShell(
	a fyne.App,
	parent fyne.Window,
	messages mailboxStore,
) (fyne.CanvasObject, error) {
	inbox, err := messages.List(mailbox.FolderInbox)
	if err != nil {
		return nil, err
	}

	reader, showMessage := newReaderPane()

	sidebar := newSidebar(func() {
		openComposeWindow(a, parent, messages)
	})
	messagePane := newMessagePane(inbox, showMessage)

	content := container.NewHSplit(messagePane, reader)
	content.SetOffset(0.38)

	shell := container.NewHSplit(sidebar, content)
	shell.SetOffset(0.18)

	return shell, nil
}

func newSidebar(onCompose func()) fyne.CanvasObject {
	compose := widget.NewButtonWithIcon(
		"Compose",
		theme.MailComposeIcon(),
		onCompose,
	)
	compose.Importance = widget.HighImportance
	compose.Alignment = widget.ButtonAlignLeading

	items := container.NewVBox(
		navButton("Inbox", theme.FolderOpenIcon()),
		navButton("Starred", nil),
		navButton("Drafts", theme.DocumentCreateIcon()),
		navButton("Outbox", theme.MailSendIcon()),
		navButton("Sent", theme.MailSendIcon()),
		navButton("Archive", theme.FolderIcon()),
		navButton("Spam", theme.WarningIcon()),
		navButton("Trash", theme.DeleteIcon()),
		widget.NewSeparator(),
		navButton("Connections", theme.FolderIcon()),
		navButton("Contacts", nil),
		navButton("RMS List", nil),
		navButton("Activity", theme.HistoryIcon()),
		navButton("Settings", theme.SettingsIcon()),
	)

	return container.NewBorder(
		compose,
		nil,
		nil,
		nil,
		container.NewVScroll(items),
	)
}

func navButton(label string, icon fyne.Resource) *widget.Button {
	var button *widget.Button

	if icon == nil {
		button = widget.NewButton(label, func() {})
	} else {
		button = widget.NewButtonWithIcon(label, icon, func() {})
	}

	button.Alignment = widget.ButtonAlignLeading
	button.Importance = widget.LowImportance

	return button
}

func newMessagePane(
	messages []mailbox.Message,
	showMessage func(mailbox.Message),
) fyne.CanvasObject {
	title := widget.NewLabelWithStyle(
		"Inbox",
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	search := widget.NewEntry()
	search.SetPlaceHolder("Search mail")

	if len(messages) == 0 {
		empty := widget.NewLabel("No messages in Inbox")
		empty.Alignment = fyne.TextAlignCenter

		return container.NewBorder(
			container.NewVBox(title, search),
			nil,
			nil,
			nil,
			container.NewCenter(empty),
		)
	}

	list := widget.NewList(
		func() int {
			return len(messages)
		},
		func() fyne.CanvasObject {
			sender := widget.NewLabelWithStyle(
				"Sender",
				fyne.TextAlignLeading,
				fyne.TextStyle{Bold: true},
			)

			subject := widget.NewLabel("Subject")

			snippet := widget.NewLabel("Message preview")
			snippet.Truncation = fyne.TextTruncateEllipsis

			return container.NewVBox(sender, subject, snippet)
		},
		func(id widget.ListItemID, object fyne.CanvasObject) {
			message := messages[id]
			row := object.(*fyne.Container)

			row.Objects[0].(*widget.Label).SetText(message.From)
			row.Objects[1].(*widget.Label).SetText(message.Subject)
			row.Objects[2].(*widget.Label).SetText(messageSnippet(message.Body))
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		showMessage(messages[id])
	}

	list.Select(0)

	return container.NewBorder(
		container.NewVBox(title, search),
		nil,
		nil,
		nil,
		list,
	)
}

func newReaderPane() (fyne.CanvasObject, func(mailbox.Message)) {
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.MailReplyIcon(), func() {}),
		widget.NewToolbarAction(theme.MailReplyAllIcon(), func() {}),
		widget.NewToolbarAction(theme.MailForwardIcon(), func() {}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.DeleteIcon(), func() {}),
	)

	subject := widget.NewLabelWithStyle(
		"",
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)
	subject.Wrapping = fyne.TextWrapWord

	from := widget.NewLabel("")
	to := widget.NewLabel("")

	body := widget.NewLabel("")
	body.Wrapping = fyne.TextWrapWord
	body.Selectable = true

	message := container.NewVBox(
		subject,
		from,
		to,
		widget.NewSeparator(),
		body,
	)

	showMessage := func(msg mailbox.Message) {
		subject.SetText(msg.Subject)
		from.SetText("From: " + msg.From)
		to.SetText("To: " + strings.Join(msg.To, ", "))
		body.SetText(msg.Body)
	}

	reader := container.NewBorder(
		toolbar,
		nil,
		nil,
		nil,
		container.NewVScroll(message),
	)

	return reader, showMessage
}

func messageSnippet(body string) string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	body = strings.ReplaceAll(body, "\r", "\n")

	for _, line := range strings.Split(body, "\n") {
		if text := strings.TrimSpace(line); text != "" {
			return text
		}
	}

	return ""
}
