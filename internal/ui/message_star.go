package ui

import (
	"fmt"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

type messageStarStore interface {
	Save(msg mailbox.Message) error
}

func setMessageStarred(
	store messageStarStore,
	msg mailbox.Message,
	starred bool,
) (mailbox.Message, error) {
	if msg.Starred == starred {
		return msg, nil
	}

	updated := msg
	updated.Starred = starred

	if err := store.Save(updated); err != nil {
		action := "star"
		if !starred {
			action = "unstar"
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
