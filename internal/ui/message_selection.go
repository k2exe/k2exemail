package ui

import "github.com/k2exe/k2exemail/internal/mailbox"

func sameMessageIdentity(
	a mailbox.Message,
	b mailbox.Message,
) bool {
	if a.ID == "" || b.ID == "" {
		return false
	}

	return a.Folder == b.Folder &&
		a.ID == b.ID
}
