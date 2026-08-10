package mailbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const messageFileName = "message.json"

var folders = []Folder{
	FolderInbox,
	FolderDrafts,
	FolderOutbox,
	FolderSent,
	FolderArchive,
	FolderSpam,
	FolderTrash,
}

type Store struct {
	root string
}

func NewStore(dataDir string) *Store {
	return &Store{
		root: filepath.Join(dataDir, "mail"),
	}
}

func (s *Store) Root() string {
	return s.root
}

func (s *Store) Prepare() error {
	for _, folder := range folders {
		if err := os.MkdirAll(s.folderPath(folder), 0o700); err != nil {
			return fmt.Errorf("create mailbox folder %q: %w", folder, err)
		}
	}
	return nil
}

func (s *Store) Save(msg Message) error {
	if err := validateMessageForStorage(msg); err != nil {
		return err
	}

	dir := s.messagePath(msg.Folder, msg.ID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create message directory: %w", err)
	}

	msg.SchemaVersion = CurrentSchemaVersion

	data, err := json.MarshalIndent(msg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode message %q: %w", msg.ID, err)
	}
	data = append(data, '\n')

	if err := writeFileSafely(
		filepath.Join(dir, messageFileName),
		data,
		0o600,
	); err != nil {
		return fmt.Errorf("save message %q: %w", msg.ID, err)
	}

	return nil
}

func (s *Store) Load(folder Folder, id string) (Message, error) {
	if !folder.Valid() {
		return Message{}, fmt.Errorf("invalid folder %q", folder)
	}
	if err := validateID(id); err != nil {
		return Message{}, err
	}

	path := filepath.Join(s.messagePath(folder, id), messageFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		return Message{}, fmt.Errorf("read message %q: %w", id, err)
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return Message{}, fmt.Errorf("decode message %q: %w", id, err)
	}

	if msg.SchemaVersion != CurrentSchemaVersion {
		return Message{}, fmt.Errorf(
			"message %q uses unsupported schema version %d",
			id,
			msg.SchemaVersion,
		)
	}

	if msg.ID != id {
		return Message{}, fmt.Errorf(
			"message ID mismatch: directory %q contains %q",
			id,
			msg.ID,
		)
	}

	// The directory location is authoritative. This keeps a recoverable
	// mailbox even if a previous move was interrupted after the rename.
	msg.Folder = folder

	return msg, nil
}

func (s *Store) List(folder Folder) ([]Message, error) {
	if !folder.Valid() {
		return nil, fmt.Errorf("invalid folder %q", folder)
	}

	entries, err := os.ReadDir(s.folderPath(folder))
	if err != nil {
		return nil, fmt.Errorf("read mailbox folder %q: %w", folder, err)
	}

	messages := make([]Message, 0, len(entries))

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		msg, err := s.Load(folder, entry.Name())
		if err != nil {
			return nil, err
		}

		messages = append(messages, msg)
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].UpdatedAt.After(messages[j].UpdatedAt)
	})

	return messages, nil
}

func (s *Store) Move(from, to Folder, id string) error {
	if !from.Valid() {
		return fmt.Errorf("invalid source folder %q", from)
	}
	if !to.Valid() {
		return fmt.Errorf("invalid destination folder %q", to)
	}
	if err := validateID(id); err != nil {
		return err
	}
	if from == to {
		return nil
	}

	source := s.messagePath(from, id)
	destination := s.messagePath(to, id)

	if _, err := os.Stat(destination); err == nil {
		return fmt.Errorf(
			"message %q already exists in folder %q",
			id,
			to,
		)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("check destination message %q: %w", id, err)
	}

	if err := os.Rename(source, destination); err != nil {
		return fmt.Errorf(
			"move message %q from %q to %q: %w",
			id,
			from,
			to,
			err,
		)
	}

	// The directory location is authoritative, so a successful rename
	// completes the move. Avoid a second metadata write that could fail
	// after the message has already changed folders.
	return nil
}

func (s *Store) Delete(folder Folder, id string) error {
	if !folder.Valid() {
		return fmt.Errorf("invalid folder %q", folder)
	}
	if err := validateID(id); err != nil {
		return err
	}

	if err := os.RemoveAll(s.messagePath(folder, id)); err != nil {
		return fmt.Errorf("delete message %q: %w", id, err)
	}

	return nil
}

func (s *Store) folderPath(folder Folder) string {
	return filepath.Join(s.root, string(folder))
}

func (s *Store) messagePath(folder Folder, id string) string {
	return filepath.Join(s.folderPath(folder), id)
}

func validateMessageForStorage(msg Message) error {
	if !msg.Folder.Valid() {
		return fmt.Errorf("invalid folder %q", msg.Folder)
	}
	if err := validateID(msg.ID); err != nil {
		return err
	}
	return nil
}

func validateID(id string) error {
	switch {
	case strings.TrimSpace(id) == "":
		return errors.New("message ID is required")
	case id == "." || id == "..":
		return fmt.Errorf("invalid message ID %q", id)
	case strings.ContainsAny(id, `/\`):
		return fmt.Errorf("invalid message ID %q", id)
	default:
		return nil
	}
}

func writeFileSafely(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, ".message-*.tmp")
	if err != nil {
		return err
	}

	tmpName := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}
	defer cleanup()

	if err := tmp.Chmod(mode); err != nil {
		return err
	}

	if _, err := tmp.Write(data); err != nil {
		return err
	}

	if err := tmp.Sync(); err != nil {
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpName, path); err != nil {
		return err
	}

	return nil
}
