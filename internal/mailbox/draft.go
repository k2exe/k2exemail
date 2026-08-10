package mailbox

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

func NewMessage(folder Folder) (Message, error) {
	if !folder.Valid() {
		return Message{}, fmt.Errorf("invalid folder %q", folder)
	}

	id, err := newLocalMessageID()
	if err != nil {
		return Message{}, fmt.Errorf("generate local message ID: %w", err)
	}

	now := time.Now().UTC()

	return Message{
		SchemaVersion: CurrentSchemaVersion,
		ID:            id,
		Folder:        folder,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func NewDraft() (Message, error) {
	return NewMessage(FolderDrafts)
}

func newLocalMessageID() (string, error) {
	var value [16]byte

	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}

	return hex.EncodeToString(value[:]), nil
}
