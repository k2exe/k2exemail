package mailbox

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

const (
	attachmentsDirName        = "attachments"
	attachmentContentFileName = "content"
)

func (s *Store) AddAttachment(
	folder Folder,
	messageID string,
	sourcePath string,
) (Attachment, error) {
	msg, err := s.Load(folder, messageID)
	if err != nil {
		return Attachment{}, fmt.Errorf("load message for attachment: %w", err)
	}

	sourcePath = strings.TrimSpace(sourcePath)
	if sourcePath == "" {
		return Attachment{}, fmt.Errorf("attachment source path is required")
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return Attachment{}, fmt.Errorf("open attachment source: %w", err)
	}
	defer source.Close()

	info, err := source.Stat()
	if err != nil {
		return Attachment{}, fmt.Errorf("stat attachment source: %w", err)
	}
	if !info.Mode().IsRegular() {
		return Attachment{}, fmt.Errorf(
			"attachment source %q is not a regular file",
			sourcePath,
		)
	}

	attachmentID, err := newAttachmentID()
	if err != nil {
		return Attachment{}, fmt.Errorf("generate attachment ID: %w", err)
	}

	attachmentDir := s.attachmentDirPath(
		folder,
		messageID,
		attachmentID,
	)

	if err := os.MkdirAll(attachmentDir, 0o700); err != nil {
		return Attachment{}, fmt.Errorf("create attachment directory: %w", err)
	}

	keepFiles := false
	defer func() {
		if !keepFiles {
			_ = os.RemoveAll(attachmentDir)
		}
	}()

	contentPath := filepath.Join(
		attachmentDir,
		attachmentContentFileName,
	)

	sourceHash, copied, err := copyAttachment(source, contentPath)
	if err != nil {
		return Attachment{}, fmt.Errorf("copy attachment: %w", err)
	}

	if copied != info.Size() {
		return Attachment{}, fmt.Errorf(
			"attachment size changed while copying: copied %d bytes, expected %d",
			copied,
			info.Size(),
		)
	}

	storedHash, err := hashFile(contentPath)
	if err != nil {
		return Attachment{}, fmt.Errorf("verify stored attachment: %w", err)
	}

	if sourceHash != storedHash {
		return Attachment{}, fmt.Errorf(
			"stored attachment verification failed for %q",
			info.Name(),
		)
	}

	attachment := Attachment{
		ID:     attachmentID,
		Name:   info.Name(),
		Size:   copied,
		SHA256: sourceHash,
	}

	if mediaType := mime.TypeByExtension(
		filepath.Ext(info.Name()),
	); mediaType != "" {
		attachment.MediaType = mediaType
	}

	msg.Attachments = append(msg.Attachments, attachment)

	if err := s.Save(msg); err != nil {
		return Attachment{}, fmt.Errorf(
			"save attachment metadata for message %q: %w",
			messageID,
			err,
		)
	}

	keepFiles = true
	return attachment, nil
}

func (s *Store) RemoveAttachment(
	folder Folder,
	messageID string,
	attachmentID string,
) error {
	if err := validateAttachmentID(attachmentID); err != nil {
		return err
	}

	msg, err := s.Load(folder, messageID)
	if err != nil {
		return fmt.Errorf("load message for attachment removal: %w", err)
	}

	index := -1
	var attachment Attachment

	for i, candidate := range msg.Attachments {
		if candidate.ID == attachmentID {
			index = i
			attachment = candidate
			break
		}
	}

	if index == -1 {
		return fmt.Errorf(
			"attachment %q not found in message %q",
			attachmentID,
			messageID,
		)
	}

	updated := make([]Attachment, 0, len(msg.Attachments)-1)
	updated = append(updated, msg.Attachments[:index]...)
	updated = append(updated, msg.Attachments[index+1:]...)
	msg.Attachments = updated

	// Persist the metadata change before deleting the stored content.
	// If cleanup is interrupted, this can leave an orphaned directory,
	// but it cannot leave the message referencing a deleted attachment.
	if err := s.Save(msg); err != nil {
		return fmt.Errorf(
			"save attachment removal for message %q: %w",
			messageID,
			err,
		)
	}

	if err := os.RemoveAll(
		s.attachmentDirPath(folder, messageID, attachmentID),
	); err != nil {
		return fmt.Errorf(
			"attachment %q removed from message %q but stored data cleanup failed: %w",
			attachment.Name,
			messageID,
			err,
		)
	}

	return nil
}

func (s *Store) OpenAttachment(
	folder Folder,
	messageID string,
	attachmentID string,
) (*os.File, Attachment, error) {
	if err := validateAttachmentID(attachmentID); err != nil {
		return nil, Attachment{}, err
	}

	msg, err := s.Load(folder, messageID)
	if err != nil {
		return nil, Attachment{}, err
	}

	var attachment Attachment
	found := false

	for _, candidate := range msg.Attachments {
		if candidate.ID == attachmentID {
			attachment = candidate
			found = true
			break
		}
	}

	if !found {
		return nil, Attachment{}, fmt.Errorf(
			"attachment %q not found in message %q",
			attachmentID,
			messageID,
		)
	}

	file, err := os.Open(
		s.attachmentContentPath(
			folder,
			messageID,
			attachmentID,
		),
	)
	if err != nil {
		return nil, Attachment{}, fmt.Errorf(
			"open stored attachment %q: %w",
			attachment.Name,
			err,
		)
	}

	return file, attachment, nil
}

func (s *Store) attachmentDirPath(
	folder Folder,
	messageID string,
	attachmentID string,
) string {
	return filepath.Join(
		s.messagePath(folder, messageID),
		attachmentsDirName,
		attachmentID,
	)
}

func (s *Store) attachmentContentPath(
	folder Folder,
	messageID string,
	attachmentID string,
) string {
	return filepath.Join(
		s.attachmentDirPath(folder, messageID, attachmentID),
		attachmentContentFileName,
	)
}

func newAttachmentID() (string, error) {
	var value [16]byte

	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}

	return hex.EncodeToString(value[:]), nil
}

func validateAttachmentID(id string) error {
	switch {
	case strings.TrimSpace(id) == "":
		return fmt.Errorf("attachment ID is required")
	case id == "." || id == "..":
		return fmt.Errorf("invalid attachment ID %q", id)
	case strings.ContainsAny(id, `/\`):
		return fmt.Errorf("invalid attachment ID %q", id)
	default:
		return nil
	}
}

func copyAttachment(
	source io.Reader,
	destinationPath string,
) (string, int64, error) {
	destination, err := os.OpenFile(
		destinationPath,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0o600,
	)
	if err != nil {
		return "", 0, err
	}

	closed := false
	defer func() {
		if !closed {
			_ = destination.Close()
		}
	}()

	hasher := sha256.New()

	copied, err := io.Copy(
		io.MultiWriter(destination, hasher),
		source,
	)
	if err != nil {
		return "", copied, err
	}

	if err := destination.Sync(); err != nil {
		return "", copied, err
	}

	if err := destination.Close(); err != nil {
		return "", copied, err
	}
	closed = true

	return hex.EncodeToString(hasher.Sum(nil)), copied, nil
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()

	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
