package ui

import (
	"fmt"
	"sync/atomic"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

const (
	mailboxActivityIdle int32 = iota
	mailboxActivityMutation
	mailboxActivityCMS
)

type mailboxActivityGate struct {
	state atomic.Int32
}

func (g *mailboxActivityGate) beginMutation() bool {
	if g == nil {
		return true
	}
	return g.state.CompareAndSwap(
		mailboxActivityIdle,
		mailboxActivityMutation,
	)
}

func (g *mailboxActivityGate) endMutation() {
	if g == nil {
		return
	}
	g.state.CompareAndSwap(
		mailboxActivityMutation,
		mailboxActivityIdle,
	)
}

func (g *mailboxActivityGate) beginCMS() bool {
	if g == nil {
		return true
	}
	return g.state.CompareAndSwap(
		mailboxActivityIdle,
		mailboxActivityCMS,
	)
}

func (g *mailboxActivityGate) endCMS() {
	if g == nil {
		return
	}
	g.state.CompareAndSwap(
		mailboxActivityCMS,
		mailboxActivityIdle,
	)
}

type messageTrashStore interface {
	Move(from, to mailbox.Folder, id string) error
	Delete(folder mailbox.Folder, id string) error
}

func trashOrDeleteMessage(
	store messageTrashStore,
	folder mailbox.Folder,
	id string,
) error {
	if folder == mailbox.FolderTrash {
		if err := store.Delete(folder, id); err != nil {
			return fmt.Errorf(
				"permanently delete message %q: %w",
				id,
				err,
			)
		}
		return nil
	}

	if err := store.Move(
		folder,
		mailbox.FolderTrash,
		id,
	); err != nil {
		return fmt.Errorf(
			"move message %q to Trash: %w",
			id,
			err,
		)
	}

	return nil
}
