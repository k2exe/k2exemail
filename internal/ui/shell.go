package ui

import (
	"fmt"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
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

	reader, showMessage := newReaderPane(
		mailbox.FolderInbox,
		nil,
	)

	messagePane := newMessagePane(
		mailbox.FolderInbox,
		inbox,
		showMessage,
	)

	content := container.NewHSplit(messagePane, reader)
	content.SetOffset(0.38)

	var loadGeneration atomic.Uint64
	currentFolder := mailbox.FolderInbox

	var switchFolder func(mailbox.Folder)
	switchFolder = func(folder mailbox.Folder) {
		currentFolder = folder
		request := loadGeneration.Add(1)

		content.Leading = newFolderStatusPane(
			folder,
			"Loading...",
		)
		content.Refresh()

		go func() {
			loaded, err := messages.List(folder)

			fyne.Do(func() {
				if request != loadGeneration.Load() {
					return
				}

				if err != nil {
					content.Leading = newFolderStatusPane(
						folder,
						"Unable to load messages",
					)
					content.Refresh()

					dialog.ShowError(
						fmt.Errorf(
							"load %s: %w",
							folderTitle(folder),
							err,
						),
						parent,
					)
					return
				}

				var onEdit func(mailbox.Message)
				if folder == mailbox.FolderDrafts {
					onEdit = func(msg mailbox.Message) {
						openDraftWindow(
							a,
							parent,
							messages,
							msg,
							func() {
								switchFolder(currentFolder)
							},
						)
					}
				}

				nextReader, nextShowMessage := newReaderPane(
					folder,
					onEdit,
				)

				content.Leading = newMessagePane(
					folder,
					loaded,
					nextShowMessage,
				)
				content.Trailing = nextReader
				content.Refresh()
			})
		}()
	}

	sidebar := newSidebar(
		func() {
			openComposeWindow(
				a,
				parent,
				messages,
				func() {
					switchFolder(currentFolder)
				},
			)
		},
		switchFolder,
	)

	shell := container.NewHSplit(sidebar, content)
	shell.SetOffset(0.18)

	return shell, nil
}

func newSidebar(
	onCompose func(),
	onFolder func(mailbox.Folder),
) fyne.CanvasObject {
	compose := widget.NewButtonWithIcon(
		"Compose",
		theme.MailComposeIcon(),
		onCompose,
	)
	compose.Importance = widget.HighImportance
	compose.Alignment = widget.ButtonAlignLeading

	items := container.NewVBox(
		navButton(
			"Inbox",
			theme.FolderOpenIcon(),
			func() { onFolder(mailbox.FolderInbox) },
		),
		navButton("Starred", nil, nil),
		navButton(
			"Drafts",
			theme.DocumentCreateIcon(),
			func() { onFolder(mailbox.FolderDrafts) },
		),
		navButton(
			"Outbox",
			theme.MailSendIcon(),
			func() { onFolder(mailbox.FolderOutbox) },
		),
		navButton(
			"Sent",
			theme.MailSendIcon(),
			func() { onFolder(mailbox.FolderSent) },
		),
		navButton(
			"Archive",
			theme.FolderIcon(),
			func() { onFolder(mailbox.FolderArchive) },
		),
		navButton(
			"Spam",
			theme.WarningIcon(),
			func() { onFolder(mailbox.FolderSpam) },
		),
		navButton(
			"Trash",
			theme.DeleteIcon(),
			func() { onFolder(mailbox.FolderTrash) },
		),
		widget.NewSeparator(),
		navButton("Connections", theme.FolderIcon(), nil),
		navButton("Contacts", nil, nil),
		navButton("RMS List", nil, nil),
		navButton("Activity", theme.HistoryIcon(), nil),
		navButton("Settings", theme.SettingsIcon(), nil),
	)

	return container.NewBorder(
		compose,
		nil,
		nil,
		nil,
		container.NewVScroll(items),
	)
}

func navButton(
	label string,
	icon fyne.Resource,
	onTapped func(),
) *widget.Button {
	if onTapped == nil {
		onTapped = func() {}
	}

	var button *widget.Button

	if icon == nil {
		button = widget.NewButton(label, onTapped)
	} else {
		button = widget.NewButtonWithIcon(
			label,
			icon,
			onTapped,
		)
	}

	button.Alignment = widget.ButtonAlignLeading
	button.Importance = widget.LowImportance

	return button
}

func newMessagePane(
	folder mailbox.Folder,
	messages []mailbox.Message,
	showMessage func(mailbox.Message),
) fyne.CanvasObject {
	titleText := folderTitle(folder)

	title := widget.NewLabelWithStyle(
		titleText,
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	search := widget.NewEntry()
	search.SetPlaceHolder("Search mail")

	if len(messages) == 0 {
		empty := widget.NewLabel(
			"No messages in " + titleText,
		)
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
			primary := widget.NewLabelWithStyle(
				"Sender",
				fyne.TextAlignLeading,
				fyne.TextStyle{Bold: true},
			)

			subject := widget.NewLabel("Subject")

			snippet := widget.NewLabel("Message preview")
			snippet.Truncation = fyne.TextTruncateEllipsis

			return container.NewVBox(
				primary,
				subject,
				snippet,
			)
		},
		func(id widget.ListItemID, object fyne.CanvasObject) {
			message := messages[id]
			row := object.(*fyne.Container)

			row.Objects[0].(*widget.Label).SetText(
				messageListPrimary(folder, message),
			)
			row.Objects[1].(*widget.Label).SetText(
				message.Subject,
			)
			row.Objects[2].(*widget.Label).SetText(
				messageSnippet(message.Body),
			)
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

func newFolderStatusPane(
	folder mailbox.Folder,
	status string,
) fyne.CanvasObject {
	title := widget.NewLabelWithStyle(
		folderTitle(folder),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	label := widget.NewLabel(status)
	label.Alignment = fyne.TextAlignCenter

	return container.NewBorder(
		title,
		nil,
		nil,
		nil,
		container.NewCenter(label),
	)
}

func folderTitle(folder mailbox.Folder) string {
	switch folder {
	case mailbox.FolderInbox:
		return "Inbox"
	case mailbox.FolderDrafts:
		return "Drafts"
	case mailbox.FolderOutbox:
		return "Outbox"
	case mailbox.FolderSent:
		return "Sent"
	case mailbox.FolderArchive:
		return "Archive"
	case mailbox.FolderSpam:
		return "Spam"
	case mailbox.FolderTrash:
		return "Trash"
	default:
		return string(folder)
	}
}

func messageListPrimary(
	folder mailbox.Folder,
	msg mailbox.Message,
) string {
	switch folder {
	case mailbox.FolderDrafts,
		mailbox.FolderOutbox,
		mailbox.FolderSent:
		if len(msg.To) == 0 {
			return "To:"
		}
		return "To: " + strings.Join(msg.To, ", ")

	default:
		if sender := strings.TrimSpace(msg.From); sender != "" {
			return sender
		}
		return "(unknown sender)"
	}
}

func newReaderPane(
	folder mailbox.Folder,
	onEdit func(mailbox.Message),
) (fyne.CanvasObject, func(mailbox.Message)) {
	var selected mailbox.Message
	var hasSelection bool
	var editAction *widget.ToolbarAction

	var toolbar *widget.Toolbar

	if folder == mailbox.FolderDrafts && onEdit != nil {
		editAction = widget.NewToolbarAction(
			theme.DocumentCreateIcon(),
			func() {
				if hasSelection {
					onEdit(selected)
				}
			},
		)
		editAction.Disable()

		toolbar = widget.NewToolbar(
			editAction,
			widget.NewToolbarSeparator(),
			widget.NewToolbarAction(
				theme.DeleteIcon(),
				func() {},
			),
		)
	} else {
		toolbar = widget.NewToolbar(
			widget.NewToolbarAction(
				theme.MailReplyIcon(),
				func() {},
			),
			widget.NewToolbarAction(
				theme.MailReplyAllIcon(),
				func() {},
			),
			widget.NewToolbarAction(
				theme.MailForwardIcon(),
				func() {},
			),
			widget.NewToolbarSeparator(),
			widget.NewToolbarAction(
				theme.DeleteIcon(),
				func() {},
			),
		)
	}

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
		selected = msg
		hasSelection = true

		if editAction != nil {
			editAction.Enable()
		}

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
