package mailbox

import (
	"errors"
	"fmt"
	"os"
)

type AttachmentContent struct {
	Name string
	Data []byte
}

// ImportMessage persists a complete message and its attachments without
// exposing a partially written message in the destination folder.
//
// The complete message is built beneath the mailbox root and published
// with a single directory rename.
func (s *Store) ImportMessage(
	msg Message,
	attachments []AttachmentContent,
) (Message, error) {
	if err := validateMessageForStorage(msg); err != nil {
		return Message{}, err
	}

	if len(msg.Attachments) != 0 {
		return Message{}, fmt.Errorf(
			"imported message %q already contains attachment metadata",
			msg.ID,
		)
	}

	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return Message{}, fmt.Errorf("create mailbox root: %w", err)
	}

	if err := os.MkdirAll(s.folderPath(msg.Folder), 0o700); err != nil {
		return Message{}, fmt.Errorf(
			"create mailbox folder %q: %w",
			msg.Folder,
			err,
		)
	}

	destination := s.messagePath(msg.Folder, msg.ID)

	if _, err := os.Stat(destination); err == nil {
		return Message{}, fmt.Errorf(
			"message %q already exists in folder %q",
			msg.ID,
			msg.Folder,
		)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Message{}, fmt.Errorf(
			"check destination message %q: %w",
			msg.ID,
			err,
		)
	}

	stagingRoot, err := os.MkdirTemp(s.root, ".import-*")
	if err != nil {
		return Message{}, fmt.Errorf("create import staging directory: %w", err)
	}
	defer os.RemoveAll(stagingRoot)

	staging := &Store{root: stagingRoot}

	if err := staging.Save(msg); err != nil {
		return Message{}, fmt.Errorf("stage message %q: %w", msg.ID, err)
	}

	for _, attachment := range attachments {
		if _, err := staging.AddAttachmentData(
			msg.Folder,
			msg.ID,
			attachment.Name,
			attachment.Data,
		); err != nil {
			return Message{}, fmt.Errorf(
				"stage attachment %q for message %q: %w",
				attachment.Name,
				msg.ID,
				err,
			)
		}
	}

	staged, err := staging.Load(msg.Folder, msg.ID)
	if err != nil {
		return Message{}, fmt.Errorf(
			"verify staged message %q: %w",
			msg.ID,
			err,
		)
	}

	if err := os.Rename(
		staging.messagePath(msg.Folder, msg.ID),
		destination,
	); err != nil {
		return Message{}, fmt.Errorf(
			"publish imported message %q: %w",
			msg.ID,
			err,
		)
	}

	return staged, nil
}
