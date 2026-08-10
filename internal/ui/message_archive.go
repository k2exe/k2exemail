package ui

import (
	"fmt"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

type messageArchiveStore interface {
	Move(from, to mailbox.Folder, id string) error
}

func archiveOrRestoreMessage(
	store messageArchiveStore,
	folder mailbox.Folder,
	id string,
) error {
	var destination mailbox.Folder

	switch folder {
	case mailbox.FolderInbox:
		destination = mailbox.FolderArchive
	case mailbox.FolderArchive:
		destination = mailbox.FolderInbox
	default:
		return fmt.Errorf(
			"folder %q does not support archive action",
			folder,
		)
	}

	if err := store.Move(folder, destination, id); err != nil {
		return fmt.Errorf(
			"move message %q from %q to %q: %w",
			id,
			folder,
			destination,
			err,
		)
	}

	return nil
}
