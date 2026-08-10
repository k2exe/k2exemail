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
	identity IdentityFunc,
	updateIdentity IdentityUpdateFunc,
	connectCMS CMSConnectFunc,
) (fyne.CanvasObject, error) {
	getIdentity := func() (string, string) {
		if identity == nil {
			return "", ""
		}

		return identity()
	}

	inbox, err := messages.List(mailbox.FolderInbox)
	if err != nil {
		return nil, err
	}

	currentFolder := mailbox.FolderInbox
	var switchFolder func(mailbox.Folder)
	activity := &mailboxActivityGate{}

	openReply := func(
		msg mailbox.Message,
		replyAll bool,
	) {
		callsign, _ := getIdentity()

		openReplyWindow(
			a,
			parent,
			messages,
			msg,
			callsign,
			replyAll,
			func() {
				if switchFolder != nil {
					switchFolder(currentFolder)
				}
			},
		)
	}

	openForward := func(msg mailbox.Message) {
		callsign, _ := getIdentity()

		openForwardWindow(
			a,
			parent,
			messages,
			msg,
			callsign,
			activity,
			func() {
				if switchFolder != nil {
					switchFolder(currentFolder)
				}
			},
		)
	}

	reader, showMessage := newReaderPane(
		parent,
		messages,
		mailbox.FolderInbox,
		nil,
		openReply,
		openForward,
		activity,
		func() {
			if switchFolder != nil &&
				currentFolder == mailbox.FolderInbox {
				switchFolder(mailbox.FolderInbox)
			}
		},
	)

	messagePane := newMessagePane(
		mailbox.FolderInbox,
		inbox,
		showMessage,
	)

	content := container.NewHSplit(messagePane, reader)
	content.SetOffset(0.38)

	var loadGeneration atomic.Uint64

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
						callsign, _ := getIdentity()

						openDraftWindow(
							a,
							parent,
							messages,
							msg,
							callsign,
							func() {
								switchFolder(currentFolder)
							},
						)
					}
				}

				nextReader, nextShowMessage := newReaderPane(
					parent,
					messages,
					folder,
					onEdit,
					openReply,
					openForward,
					activity,
					func() {
						if currentFolder == folder {
							switchFolder(folder)
						}
					},
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

	var connectionsWindow fyne.Window
	var settingsWindow fyne.Window

	openConnections := func() {
		if connectionsWindow != nil {
			connectionsWindow.RequestFocus()
			return
		}

		callsign, locator := getIdentity()

		connectionsWindow = newConnectionsWindow(
			a,
			callsign,
			locator,
			connectCMS,
			activity,
			func() {
				switchFolder(currentFolder)
			},
			func() {
				connectionsWindow = nil
			},
		)

		connectionsWindow.Show()
	}

	openSettings := func() {
		if settingsWindow != nil {
			settingsWindow.RequestFocus()
			return
		}

		callsign, locator := getIdentity()

		settingsWindow = newSettingsWindow(
			a,
			callsign,
			locator,
			updateIdentity,
			func() {
				settingsWindow = nil
			},
		)

		settingsWindow.Show()
	}

	sidebar := newSidebar(
		func() {
			callsign, _ := getIdentity()

			openComposeWindow(
				a,
				parent,
				messages,
				callsign,
				func() {
					switchFolder(currentFolder)
				},
			)
		},
		switchFolder,
		openConnections,
		openSettings,
	)

	shell := container.NewHSplit(sidebar, content)
	shell.SetOffset(0.18)

	return shell, nil
}

func newSidebar(
	onCompose func(),
	onFolder func(mailbox.Folder),
	onConnections func(),
	onSettings func(),
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
		navButton(
			"Connections",
			theme.FolderIcon(),
			onConnections,
		),
		navButton("Contacts", nil, nil),
		navButton("RMS List", nil, nil),
		navButton("Activity", theme.HistoryIcon(), nil),
		navButton(
			"Settings",
			theme.SettingsIcon(),
			onSettings,
		),
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
	parent fyne.Window,
	store mailboxStore,
	folder mailbox.Folder,
	onEdit func(mailbox.Message),
	onReply func(mailbox.Message, bool),
	onForward func(mailbox.Message),
	activity *mailboxActivityGate,
	onRemoved func(),
) (fyne.CanvasObject, func(mailbox.Message)) {
	var selected mailbox.Message
	var hasSelection bool
	var removing bool
	var editAction *widget.ToolbarAction
	var replyAction *widget.ToolbarAction
	var replyAllAction *widget.ToolbarAction
	var forwardAction *widget.ToolbarAction
	var removeSelected func()

	deleteAction := widget.NewToolbarAction(
		theme.DeleteIcon(),
		func() {
			if removeSelected != nil {
				removeSelected()
			}
		},
	)
	deleteAction.Disable()

	var toolbar *widget.Toolbar

	if folder == mailbox.FolderDrafts && onEdit != nil {
		editAction = widget.NewToolbarAction(
			theme.DocumentCreateIcon(),
			func() {
				if hasSelection && !removing {
					onEdit(selected)
				}
			},
		)
		editAction.Disable()

		toolbar = widget.NewToolbar(
			editAction,
			widget.NewToolbarSeparator(),
			deleteAction,
		)
	} else {
		replyAction = widget.NewToolbarAction(
			theme.MailReplyIcon(),
			func() {
				if hasSelection &&
					!removing &&
					onReply != nil {
					onReply(selected, false)
				}
			},
		)
		replyAction.Disable()

		replyAllAction = widget.NewToolbarAction(
			theme.MailReplyAllIcon(),
			func() {
				if hasSelection &&
					!removing &&
					onReply != nil {
					onReply(selected, true)
				}
			},
		)
		replyAllAction.Disable()

		forwardAction = widget.NewToolbarAction(
			theme.MailForwardIcon(),
			func() {
				if hasSelection &&
					!removing &&
					onForward != nil {
					onForward(selected)
				}
			},
		)
		forwardAction.Disable()

		toolbar = widget.NewToolbar(
			replyAction,
			replyAllAction,
			forwardAction,
			widget.NewToolbarSeparator(),
			deleteAction,
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

	attachments, showAttachments := newReaderAttachments(
		parent,
		store,
		folder,
	)

	clearSelection := func() {
		selected = mailbox.Message{}
		hasSelection = false

		deleteAction.Disable()
		if editAction != nil {
			editAction.Disable()
		}
		if replyAction != nil {
			replyAction.Disable()
		}
		if replyAllAction != nil {
			replyAllAction.Disable()
		}
		if forwardAction != nil {
			forwardAction.Disable()
		}

		subject.SetText("")
		from.SetText("")
		to.SetText("")
		body.SetText("")
		showAttachments(mailbox.Message{})
	}

	removeSelected = func() {
		if !hasSelection || removing {
			return
		}

		msg := selected

		remove := func() {
			if activity != nil &&
				!activity.beginMutation() {
				dialog.ShowInformation(
					"Mailbox busy",
					"Messages cannot be moved or deleted while a CMS session or another mailbox update is active.",
					parent,
				)
				return
			}

			removing = true
			deleteAction.Disable()
			if editAction != nil {
				editAction.Disable()
			}
			if replyAction != nil {
				replyAction.Disable()
			}
			if replyAllAction != nil {
				replyAllAction.Disable()
			}
			if forwardAction != nil {
				forwardAction.Disable()
			}

			go func() {
				err := trashOrDeleteMessage(
					store,
					folder,
					msg.ID,
				)

				if activity != nil {
					activity.endMutation()
				}

				fyne.Do(func() {
					removing = false

					if err != nil {
						if hasSelection {
							deleteAction.Enable()
							if editAction != nil {
								editAction.Enable()
							}
							if replyAction != nil &&
								onReply != nil {
								replyAction.Enable()
							}
							if replyAllAction != nil &&
								onReply != nil {
								replyAllAction.Enable()
							}
							if forwardAction != nil &&
								onForward != nil {
								forwardAction.Enable()
							}
						}

						dialog.ShowError(err, parent)
						return
					}

					clearSelection()

					if onRemoved != nil {
						onRemoved()
					}
				})
			}()
		}

		if folder == mailbox.FolderTrash {
			dialog.ShowConfirm(
				"Delete permanently?",
				"This message and its attachments will be permanently deleted. This cannot be undone.",
				func(ok bool) {
					if ok {
						remove()
					}
				},
				parent,
			)
			return
		}

		remove()
	}

	message := container.NewVBox(
		subject,
		from,
		to,
		attachments,
		widget.NewSeparator(),
		body,
	)

	showMessage := func(msg mailbox.Message) {
		selected = msg
		hasSelection = true

		if !removing {
			deleteAction.Enable()
			if editAction != nil {
				editAction.Enable()
			}
			if replyAction != nil &&
				onReply != nil {
				replyAction.Enable()
			}
			if replyAllAction != nil &&
				onReply != nil {
				replyAllAction.Enable()
			}
			if forwardAction != nil &&
				onForward != nil {
				forwardAction.Enable()
			}
		}

		subject.SetText(msg.Subject)
		from.SetText("From: " + msg.From)
		to.SetText("To: " + strings.Join(msg.To, ", "))
		body.SetText(msg.Body)
		showAttachments(msg)
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
