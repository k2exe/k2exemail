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

	inboxView := folderMailView(mailbox.FolderInbox)

	inbox, err := loadMailView(messages, inboxView)
	if err != nil {
		return nil, err
	}

	currentView := inboxView
	var switchView func(mailView)
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
				if switchView != nil {
					switchView(currentView)
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
				if switchView != nil {
					switchView(currentView)
				}
			},
		)
	}

	var updateMessage func(mailbox.Message)

	reader, showMessage, clearMessage := newReaderPane(
		parent,
		messages,
		inboxView,
		nil,
		openReply,
		openForward,
		activity,
		func(updated mailbox.Message) {
			if updateMessage != nil {
				updateMessage(updated)
			}
		},
		func() {
			if switchView != nil &&
				currentView == inboxView {
				switchView(inboxView)
			}
		},
	)

	messagePane, updateMessage := newMessagePane(
		inboxView,
		inbox,
		showMessage,
		clearMessage,
	)

	content := container.NewHSplit(messagePane, reader)
	content.SetOffset(0.38)

	var loadGeneration atomic.Uint64

	switchView = func(view mailView) {
		currentView = view
		request := loadGeneration.Add(1)

		content.Leading = newMailViewStatusPane(
			view,
			"Loading...",
		)
		content.Refresh()

		go func() {
			loaded, err := loadMailView(messages, view)

			fyne.Do(func() {
				if request != loadGeneration.Load() {
					return
				}

				if err != nil {
					content.Leading = newMailViewStatusPane(
						view,
						"Unable to load messages",
					)
					content.Refresh()

					dialog.ShowError(
						fmt.Errorf(
							"load %s: %w",
							view.title(),
							err,
						),
						parent,
					)
					return
				}

				var onEdit func(mailbox.Message)
				if view.isDrafts() {
					onEdit = func(msg mailbox.Message) {
						callsign, _ := getIdentity()

						openDraftWindow(
							a,
							parent,
							messages,
							msg,
							callsign,
							func() {
								switchView(currentView)
							},
						)
					}
				}

				var updateMessage func(mailbox.Message)

				nextReader, nextShowMessage, nextClearMessage :=
					newReaderPane(
						parent,
						messages,
						view,
						onEdit,
						openReply,
						openForward,
						activity,
						func(updated mailbox.Message) {
							if updateMessage != nil {
								updateMessage(updated)
							}
						},
						func() {
							if currentView == view {
								switchView(view)
							}
						},
					)

				var nextMessagePane fyne.CanvasObject
				nextMessagePane, updateMessage = newMessagePane(
					view,
					loaded,
					nextShowMessage,
					nextClearMessage,
				)

				content.Leading = nextMessagePane
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
				switchView(currentView)
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
					switchView(currentView)
				},
			)
		},
		switchView,
		openConnections,
		openSettings,
	)

	shell := container.NewHSplit(sidebar, content)
	shell.SetOffset(0.18)

	return shell, nil
}

func newSidebar(
	onCompose func(),
	onView func(mailView),
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
			func() {
				onView(folderMailView(mailbox.FolderInbox))
			},
		),
		navButton(
			"Starred",
			messageStarOutlineIcon,
			func() {
				onView(starredMailView())
			},
		),
		navButton(
			"Drafts",
			theme.DocumentCreateIcon(),
			func() {
				onView(folderMailView(mailbox.FolderDrafts))
			},
		),
		navButton(
			"Outbox",
			theme.MailSendIcon(),
			func() {
				onView(folderMailView(mailbox.FolderOutbox))
			},
		),
		navButton(
			"Sent",
			theme.MailSendIcon(),
			func() {
				onView(folderMailView(mailbox.FolderSent))
			},
		),
		navButton(
			"Archive",
			theme.FolderIcon(),
			func() {
				onView(folderMailView(mailbox.FolderArchive))
			},
		),
		navButton(
			"Spam",
			theme.WarningIcon(),
			func() {
				onView(folderMailView(mailbox.FolderSpam))
			},
		),
		navButton(
			"Trash",
			theme.DeleteIcon(),
			func() {
				onView(folderMailView(mailbox.FolderTrash))
			},
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
	view mailView,
	messages []mailbox.Message,
	showMessage func(mailbox.Message),
	clearMessage func(),
) (fyne.CanvasObject, func(mailbox.Message)) {
	titleText := view.title()

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
		), func(mailbox.Message) {}
	}

	filtered := messages

	list := widget.NewList(
		func() int {
			return len(filtered)
		},
		func() fyne.CanvasObject {
			primary := widget.NewLabel("Sender")
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
			message := filtered[id]
			row := object.(*fyne.Container)
			style := messageListTextStyle(message)

			primary := row.Objects[0].(*widget.Label)
			primary.TextStyle = style
			primary.SetText(
				messageListPrimaryForView(view, message),
			)

			subject := row.Objects[1].(*widget.Label)
			subject.TextStyle = style
			subject.SetText(message.Subject)
			row.Objects[2].(*widget.Label).SetText(
				messageSnippet(message.Body),
			)
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(filtered) {
			return
		}

		showMessage(filtered[id])
	}

	search.OnChanged = func(query string) {
		filtered = filterMessages(
			messages,
			query,
		)

		list.UnselectAll()
		list.Refresh()

		if clearMessage != nil {
			clearMessage()
		}
	}

	updateMessage := func(updated mailbox.Message) {
		replaceMessageSnapshot(messages, updated)
		replaceMessageSnapshot(filtered, updated)
		list.Refresh()
	}

	return container.NewBorder(
		container.NewVBox(title, search),
		nil,
		nil,
		nil,
		list,
	), updateMessage
}

func messageListTextStyle(
	msg mailbox.Message,
) fyne.TextStyle {
	return fyne.TextStyle{
		Bold: msg.Unread,
	}
}

func messageListPrimaryForView(
	view mailView,
	msg mailbox.Message,
) string {
	folder := view.folder
	if view.isStarred() {
		folder = msg.Folder
	}

	return messageListPrimary(folder, msg)
}

func newMailViewStatusPane(
	view mailView,
	status string,
) fyne.CanvasObject {
	title := widget.NewLabelWithStyle(
		view.title(),
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
	view mailView,
	onEdit func(mailbox.Message),
	onReply func(mailbox.Message, bool),
	onForward func(mailbox.Message),
	activity *mailboxActivityGate,
	onUpdated func(mailbox.Message),
	onRemoved func(),
) (
	fyne.CanvasObject,
	func(mailbox.Message),
	func(),
) {
	var selected mailbox.Message
	var hasSelection bool
	var mutating bool
	var editAction *widget.ToolbarAction
	var replyAction *widget.ToolbarAction
	var replyAllAction *widget.ToolbarAction
	var forwardAction *widget.ToolbarAction
	var starAction *widget.ToolbarAction
	var readAction *widget.ToolbarAction
	var archiveAction *widget.ToolbarAction
	var starSelected func()
	var toggleReadSelected func()
	var archiveSelected func()
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

	if view.isDrafts() && onEdit != nil {
		editAction = widget.NewToolbarAction(
			theme.DocumentCreateIcon(),
			func() {
				if hasSelection && !mutating {
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
					!mutating &&
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
					!mutating &&
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
					!mutating &&
					onForward != nil {
					onForward(selected)
				}
			},
		)
		forwardAction.Disable()

		starAction = widget.NewToolbarAction(
			messageStarIcon(false),
			func() {
				if starSelected != nil {
					starSelected()
				}
			},
		)
		starAction.Disable()

		readAction = widget.NewToolbarAction(
			messageReadActionIcon(false),
			func() {
				if toggleReadSelected != nil {
					toggleReadSelected()
				}
			},
		)
		readAction.Disable()

		toolbarItems := []widget.ToolbarItem{
			replyAction,
			replyAllAction,
			forwardAction,
			widget.NewToolbarSeparator(),
			starAction,
			readAction,
		}

		if !view.isStarred() &&
			(view.folder == mailbox.FolderInbox ||
				view.folder == mailbox.FolderArchive) {
			archiveIcon := theme.FolderIcon()
			if view.folder == mailbox.FolderArchive {
				archiveIcon = theme.NavigateBackIcon()
			}

			archiveAction = widget.NewToolbarAction(
				archiveIcon,
				func() {
					if archiveSelected != nil {
						archiveSelected()
					}
				},
			)
			archiveAction.Disable()
			toolbarItems = append(
				toolbarItems,
				archiveAction,
				widget.NewToolbarSeparator(),
			)
		}

		toolbarItems = append(toolbarItems, deleteAction)
		toolbar = widget.NewToolbar(toolbarItems...)
	}

	disableActions := func() {
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
		if starAction != nil {
			starAction.Disable()
		}
		if readAction != nil {
			readAction.Disable()
		}
		if archiveAction != nil {
			archiveAction.Disable()
		}
	}

	enableActions := func() {
		if !hasSelection || mutating {
			return
		}

		deleteAction.Enable()

		if editAction != nil {
			editAction.Enable()
		}
		if replyAction != nil && onReply != nil {
			replyAction.Enable()
		}
		if replyAllAction != nil && onReply != nil {
			replyAllAction.Enable()
		}
		if forwardAction != nil && onForward != nil {
			forwardAction.Enable()
		}
		if starAction != nil {
			starAction.Enable()
		}
		if readAction != nil {
			readAction.Enable()
		}
		if archiveAction != nil {
			archiveAction.Enable()
		}
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
	)

	clearSelection := func() {
		selected = mailbox.Message{}
		hasSelection = false

		disableActions()

		subject.SetText("")
		from.SetText("")
		to.SetText("")
		body.SetText("")
		showAttachments(mailbox.Message{})
	}

	starSelected = func() {
		if !hasSelection ||
			mutating ||
			starAction == nil {
			return
		}

		msg := selected

		if activity != nil &&
			!activity.beginMutation() {
			dialog.ShowInformation(
				"Mailbox busy",
				"Messages cannot be changed while a CMS session or another mailbox update is active.",
				parent,
			)
			return
		}

		mutating = true
		disableActions()

		go func() {
			updated, err := setMessageStarred(
				store,
				msg,
				!msg.Starred,
			)

			if activity != nil {
				activity.endMutation()
			}

			fyne.Do(func() {
				mutating = false

				if err != nil {
					enableActions()
					dialog.ShowError(err, parent)
					return
				}

				selected = updated

				if onUpdated != nil {
					onUpdated(updated)
				}

				if view.isStarred() && !updated.Starred {
					clearSelection()

					if onRemoved != nil {
						onRemoved()
					}
					return
				}

				starAction.SetIcon(
					messageStarIcon(updated.Starred),
				)
				enableActions()
			})
		}()
	}

	toggleReadSelected = func() {
		if !hasSelection ||
			mutating ||
			readAction == nil {
			return
		}

		msg := selected

		if activity != nil &&
			!activity.beginMutation() {
			dialog.ShowInformation(
				"Mailbox busy",
				"Messages cannot be changed while a CMS session or another mailbox update is active.",
				parent,
			)
			return
		}

		mutating = true
		disableActions()

		go func() {
			updated, err := setMessageUnread(
				store,
				msg,
				!msg.Unread,
			)

			if activity != nil {
				activity.endMutation()
			}

			fyne.Do(func() {
				mutating = false

				if err != nil {
					enableActions()
					dialog.ShowError(err, parent)
					return
				}

				selected = updated

				if onUpdated != nil {
					onUpdated(updated)
				}

				readAction.SetIcon(
					messageReadActionIcon(updated.Unread),
				)
				enableActions()
			})
		}()
	}

	archiveSelected = func() {
		if !hasSelection ||
			mutating ||
			archiveAction == nil {
			return
		}

		msg := selected

		if activity != nil &&
			!activity.beginMutation() {
			dialog.ShowInformation(
				"Mailbox busy",
				"Messages cannot be moved while a CMS session or another mailbox update is active.",
				parent,
			)
			return
		}

		mutating = true
		disableActions()

		go func() {
			err := archiveOrRestoreMessage(
				store,
				msg.Folder,
				msg.ID,
			)

			if activity != nil {
				activity.endMutation()
			}

			fyne.Do(func() {
				mutating = false

				if err != nil {
					enableActions()

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

	removeSelected = func() {
		if !hasSelection || mutating {
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

			mutating = true
			disableActions()

			go func() {
				err := trashOrDeleteMessage(
					store,
					msg.Folder,
					msg.ID,
				)

				if activity != nil {
					activity.endMutation()
				}

				fyne.Do(func() {
					mutating = false

					if err != nil {
						enableActions()

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

		if msg.Folder == mailbox.FolderTrash {
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

		if starAction != nil {
			starAction.SetIcon(
				messageStarIcon(msg.Starred),
			)
		}

		if readAction != nil {
			readAction.SetIcon(
				messageReadActionIcon(msg.Unread),
			)
		}

		enableActions()

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

	return reader, showMessage, clearSelection
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
