package mailbox

import "time"

type Folder string

const (
	FolderInbox   Folder = "inbox"
	FolderDrafts  Folder = "drafts"
	FolderOutbox  Folder = "outbox"
	FolderSent    Folder = "sent"
	FolderArchive Folder = "archive"
	FolderSpam    Folder = "spam"
	FolderTrash   Folder = "trash"
)

func (f Folder) Valid() bool {
	switch f {
	case FolderInbox,
		FolderDrafts,
		FolderOutbox,
		FolderSent,
		FolderArchive,
		FolderSpam,
		FolderTrash:
		return true
	default:
		return false
	}
}

type Attachment struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MediaType string `json:"media_type,omitempty"`
	Size      int64  `json:"size"`
	SHA256    string `json:"sha256,omitempty"`
}

const CurrentSchemaVersion = 1

type Message struct {
	SchemaVersion int          `json:"schema_version"`
	ID            string       `json:"id"`
	WinlinkMID    string       `json:"winlink_mid,omitempty"`
	Folder        Folder       `json:"folder"`
	From          string       `json:"from,omitempty"`
	To            []string     `json:"to,omitempty"`
	Cc            []string     `json:"cc,omitempty"`
	Subject       string       `json:"subject,omitempty"`
	Body          string       `json:"body,omitempty"`
	Attachments   []Attachment `json:"attachments,omitempty"`

	Starred bool `json:"starred,omitempty"`
	Unread  bool `json:"unread,omitempty"`
	P2POnly bool `json:"p2p_only,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
