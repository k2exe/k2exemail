package ui

import (
	"fmt"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

type messageReadStore interface {
	Save(msg mailbox.Message) error
}

func setMessageUnread(
	store messageReadStore,
	msg mailbox.Message,
	unread bool,
) (mailbox.Message, error) {
	if msg.Unread == unread {
		return msg, nil
	}

	updated := msg
	updated.Unread = unread

	if err := store.Save(updated); err != nil {
		action := "mark read"
		if unread {
			action = "mark unread"
		}

		return msg, fmt.Errorf(
			"%s message %q: %w",
			action,
			msg.ID,
			err,
		)
	}

	return updated, nil
}
