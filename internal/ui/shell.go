package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type messagePreview struct {
	sender  string
	subject string
	snippet string
}

var sampleMessages = []messagePreview{
	{
		sender:  "Winlink System",
		subject: "Welcome to K2EXEmail",
		snippet: "Placeholder message used to validate the desktop layout.",
	},
	{
		sender:  "BBRC",
		subject: "AREDN exercise traffic",
		snippet: "Sample traffic for the message list.",
	},
	{
		sender:  "K2EXE",
		subject: "Field deployment notes",
		snippet: "Draft planning notes for an offline operation.",
	},
	{
		sender:  "Weather",
		subject: "Weather bulletin",
		snippet: "Sample bulletin content for UI development.",
	},
}

func newMailShell() fyne.CanvasObject {
	reader, showMessage := newReaderPane()

	sidebar := newSidebar()
	messages := newMessagePane(showMessage)

	content := container.NewHSplit(messages, reader)
	content.SetOffset(0.38)

	shell := container.NewHSplit(sidebar, content)
	shell.SetOffset(0.18)

	return shell
}

func newSidebar() fyne.CanvasObject {
	compose := widget.NewButtonWithIcon(
		"Compose",
		theme.MailComposeIcon(),
		func() {},
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

func newMessagePane(showMessage func(messagePreview)) fyne.CanvasObject {
	title := widget.NewLabelWithStyle(
		"Inbox",
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	search := widget.NewEntry()
	search.SetPlaceHolder("Search mail")

	list := widget.NewList(
		func() int {
			return len(sampleMessages)
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
			message := sampleMessages[id]
			row := object.(*fyne.Container)

			row.Objects[0].(*widget.Label).SetText(message.sender)
			row.Objects[1].(*widget.Label).SetText(message.subject)
			row.Objects[2].(*widget.Label).SetText(message.snippet)
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		showMessage(sampleMessages[id])
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

func newReaderPane() (fyne.CanvasObject, func(messagePreview)) {
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
	to := widget.NewLabel("To: K2EXE")

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

	showMessage := func(msg messagePreview) {
		subject.SetText(msg.subject)
		from.SetText("From: " + msg.sender)
		body.SetText(msg.snippet)
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
