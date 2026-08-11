package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	fyneTest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

func TestMessageSnippet(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "single line",
			body: "Hello from K2EXEmail",
			want: "Hello from K2EXEmail",
		},
		{
			name: "first non-empty line",
			body: "\n\n  First line  \nSecond line",
			want: "First line",
		},
		{
			name: "windows line endings",
			body: "\r\nFirst line\r\nSecond line",
			want: "First line",
		},
		{
			name: "empty body",
			body: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := messageSnippet(tt.body)
			if got != tt.want {
				t.Fatalf(
					"messageSnippet(%q) = %q, want %q",
					tt.body,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestMessageListPrimary(t *testing.T) {
	outgoing := mailbox.Message{
		To: []string{"K2EXE", "KR2SSY"},
	}

	if got := messageListPrimary(
		mailbox.FolderOutbox,
		outgoing,
	); got != "To: K2EXE, KR2SSY" {
		t.Fatalf("outbox primary = %q", got)
	}

	incoming := mailbox.Message{
		From: "W2ABC",
	}

	if got := messageListPrimary(
		mailbox.FolderInbox,
		incoming,
	); got != "W2ABC" {
		t.Fatalf("inbox primary = %q", got)
	}
}

func TestMessageListPrimaryForStarredViewUsesMessageFolder(
	t *testing.T,
) {
	sent := mailbox.Message{
		Folder: mailbox.FolderSent,
		To:     []string{"K2EXE", "W2ABC"},
	}

	if got := messageListPrimaryForView(
		starredMailView(),
		sent,
	); got != "To: K2EXE, W2ABC" {
		t.Fatalf(
			"starred Sent primary = %q, want recipient",
			got,
		)
	}

	inbox := mailbox.Message{
		Folder: mailbox.FolderInbox,
		From:   "W2XYZ",
	}

	if got := messageListPrimaryForView(
		starredMailView(),
		inbox,
	); got != "W2XYZ" {
		t.Fatalf(
			"starred Inbox primary = %q, want sender",
			got,
		)
	}
}

func TestMessageListTextStyleUsesUnreadState(
	t *testing.T,
) {
	unread := messageListTextStyle(
		mailbox.Message{Unread: true},
	)
	if !unread.Bold {
		t.Fatal("unread message style Bold = false")
	}

	read := messageListTextStyle(
		mailbox.Message{Unread: false},
	)
	if read.Bold {
		t.Fatal("read message style Bold = true")
	}
}

func TestMessagePaneDoesNotOpenMessageOnCreation(
	t *testing.T,
) {
	fyneTest.NewApp()
	messages := []mailbox.Message{
		{
			ID:      "message-1",
			Folder:  mailbox.FolderInbox,
			From:    "W2ABC",
			Subject: "First",
		},
	}

	opened := 0

	newMessagePane(
		folderMailView(mailbox.FolderInbox),
		messages,
		func(mailbox.Message) {
			opened++
		},
		func() {},
	)

	if opened != 0 {
		t.Fatalf(
			"showMessage calls = %d, want 0",
			opened,
		)
	}
}

func TestMessagePaneSearchClearsWithoutOpeningFirstResult(
	t *testing.T,
) {
	fyneTest.NewApp()
	messages := []mailbox.Message{
		{
			ID:      "message-1",
			Folder:  mailbox.FolderInbox,
			From:    "W2ABC",
			Subject: "Alpha",
		},
		{
			ID:      "message-2",
			Folder:  mailbox.FolderInbox,
			From:    "W2XYZ",
			Subject: "Bravo",
		},
	}

	opened := 0
	cleared := 0

	pane, _, _ := newMessagePane(
		folderMailView(mailbox.FolderInbox),
		messages,
		func(mailbox.Message) {
			opened++
		},
		func() {
			cleared++
		},
	)

	search := findEntryWithPlaceholder(
		pane,
		"Search mail",
	)
	if search == nil {
		t.Fatal("Search mail entry not found")
	}

	search.OnChanged("Bravo")

	if opened != 0 {
		t.Fatalf(
			"showMessage calls = %d, want 0",
			opened,
		)
	}

	if cleared != 1 {
		t.Fatalf(
			"clearMessage calls = %d, want 1",
			cleared,
		)
	}
}

func findEntryWithPlaceholder(
	object fyne.CanvasObject,
	placeholder string,
) *widget.Entry {
	if entry, ok := object.(*widget.Entry); ok &&
		entry.PlaceHolder == placeholder {
		return entry
	}

	container, ok := object.(*fyne.Container)
	if !ok {
		return nil
	}

	for _, child := range container.Objects {
		if entry := findEntryWithPlaceholder(
			child,
			placeholder,
		); entry != nil {
			return entry
		}
	}

	return nil
}

func TestMessagePaneOpensMessageWhenSelected(
	t *testing.T,
) {
	fyneTest.NewApp()

	messages := []mailbox.Message{
		{
			ID:      "message-1",
			Folder:  mailbox.FolderInbox,
			From:    "W2ABC",
			Subject: "First",
		},
	}

	var opened mailbox.Message
	openCount := 0

	pane, _, _ := newMessagePane(
		folderMailView(mailbox.FolderInbox),
		messages,
		func(msg mailbox.Message) {
			opened = msg
			openCount++
		},
		func() {},
	)

	list := findMessageList(pane)
	if list == nil {
		t.Fatal("message list not found")
	}

	list.Select(0)

	if openCount != 1 {
		t.Fatalf(
			"showMessage calls = %d, want 1",
			openCount,
		)
	}

	if opened.ID != "message-1" {
		t.Fatalf(
			"opened message ID = %q, want %q",
			opened.ID,
			"message-1",
		)
	}
}

func findMessageList(
	object fyne.CanvasObject,
) *widget.List {
	if list, ok := object.(*widget.List); ok {
		return list
	}

	container, ok := object.(*fyne.Container)
	if !ok {
		return nil
	}

	for _, child := range container.Objects {
		if list := findMessageList(child); list != nil {
			return list
		}
	}

	return nil
}

func TestMessagePaneRemovalPreservesNewerSelection(
	t *testing.T,
) {
	fyneTest.NewApp()

	messages := []mailbox.Message{
		{
			ID:      "message-a",
			Folder:  mailbox.FolderInbox,
			Subject: "Message A",
		},
		{
			ID:      "message-b",
			Folder:  mailbox.FolderInbox,
			Subject: "Message B",
		},
		{
			ID:      "message-c",
			Folder:  mailbox.FolderInbox,
			Subject: "Message C",
		},
	}

	var opened mailbox.Message
	cleared := 0

	pane, _, removeMessage := newMessagePane(
		folderMailView(mailbox.FolderInbox),
		messages,
		func(msg mailbox.Message) {
			opened = msg
		},
		func() {
			cleared++
		},
	)

	list := findMessageList(pane)
	if list == nil {
		t.Fatal("message list not found")
	}

	list.Select(1)
	if opened.ID != "message-b" {
		t.Fatalf(
			"opened message = %q, want message-b",
			opened.ID,
		)
	}

	// A disappears after B became the active selection.
	// B moves from row 1 to row 0 and must remain active.
	removeMessage(messages[0])

	if got := list.Length(); got != 2 {
		t.Fatalf("list length = %d, want 2", got)
	}
	if opened.ID != "message-b" {
		t.Fatalf(
			"opened message after removal = %q, want message-b",
			opened.ID,
		)
	}
	if cleared != 0 {
		t.Fatalf(
			"clearMessage calls = %d, want 0",
			cleared,
		)
	}
}

func TestMessagePaneRemovalClearsRemovedSelection(
	t *testing.T,
) {
	fyneTest.NewApp()

	messages := []mailbox.Message{
		{
			ID:     "message-a",
			Folder: mailbox.FolderInbox,
		},
		{
			ID:     "message-b",
			Folder: mailbox.FolderInbox,
		},
	}

	cleared := 0

	pane, _, removeMessage := newMessagePane(
		folderMailView(mailbox.FolderInbox),
		messages,
		func(mailbox.Message) {},
		func() {
			cleared++
		},
	)

	list := findMessageList(pane)
	if list == nil {
		t.Fatal("message list not found")
	}

	list.Select(0)
	removeMessage(messages[0])

	if got := list.Length(); got != 1 {
		t.Fatalf("list length = %d, want 1", got)
	}
	if cleared != 1 {
		t.Fatalf(
			"clearMessage calls = %d, want 1",
			cleared,
		)
	}
}
