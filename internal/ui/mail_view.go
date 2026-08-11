package ui

import (
	"fmt"
	"sort"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

type mailViewKind uint8

const (
	mailViewFolder mailViewKind = iota
	mailViewStarred
)

type mailView struct {
	kind   mailViewKind
	folder mailbox.Folder
}

func folderMailView(folder mailbox.Folder) mailView {
	return mailView{
		kind:   mailViewFolder,
		folder: folder,
	}
}

func starredMailView() mailView {
	return mailView{
		kind: mailViewStarred,
	}
}

func (v mailView) title() string {
	if v.kind == mailViewStarred {
		return "Starred"
	}

	return folderTitle(v.folder)
}

func (v mailView) isStarred() bool {
	return v.kind == mailViewStarred
}

func (v mailView) isDrafts() bool {
	return v.kind == mailViewFolder &&
		v.folder == mailbox.FolderDrafts
}

type mailViewStore interface {
	List(folder mailbox.Folder) ([]mailbox.Message, error)
}

var starredViewFolders = []mailbox.Folder{
	mailbox.FolderInbox,
	mailbox.FolderOutbox,
	mailbox.FolderSent,
	mailbox.FolderArchive,
}

func loadMailView(
	store mailViewStore,
	view mailView,
) ([]mailbox.Message, error) {
	if !view.isStarred() {
		return store.List(view.folder)
	}

	var messages []mailbox.Message

	for _, folder := range starredViewFolders {
		loaded, err := store.List(folder)
		if err != nil {
			return nil, fmt.Errorf(
				"load starred messages from %q: %w",
				folder,
				err,
			)
		}

		for _, msg := range loaded {
			if msg.Starred {
				messages = append(messages, msg)
			}
		}
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].UpdatedAt.After(
			messages[j].UpdatedAt,
		)
	})

	return messages, nil
}

func replaceMessageSnapshot(
	messages []mailbox.Message,
	updated mailbox.Message,
) bool {
	for i := range messages {
		if messages[i].ID == updated.ID &&
			messages[i].Folder == updated.Folder {
			messages[i] = updated
			return true
		}
	}

	return false
}

func removeMessageSnapshot(
	messages []mailbox.Message,
	removed mailbox.Message,
) ([]mailbox.Message, bool) {
	for i := range messages {
		if !sameMessageIdentity(messages[i], removed) {
			continue
		}

		remaining := make(
			[]mailbox.Message,
			0,
			len(messages)-1,
		)
		remaining = append(remaining, messages[:i]...)
		remaining = append(remaining, messages[i+1:]...)

		return remaining, true
	}

	return messages, false
}
