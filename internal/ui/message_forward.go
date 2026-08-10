package ui

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

type messageForwardStore interface {
	Save(msg mailbox.Message) error
	Delete(folder mailbox.Folder, id string) error

	OpenAttachmentReader(
		folder mailbox.Folder,
		messageID string,
		attachmentID string,
	) (io.ReadCloser, mailbox.Attachment, error)

	AddAttachmentReader(
		folder mailbox.Folder,
		messageID string,
		name string,
		source io.Reader,
	) (mailbox.Attachment, error)
}

func openForwardWindow(
	a fyne.App,
	parent fyne.Window,
	store mailboxStore,
	original mailbox.Message,
	callsign string,
	activity *mailboxActivityGate,
	onChanged func(),
) {
	if activity != nil &&
		!activity.beginMutation() {
		dialog.ShowInformation(
			"Mailbox busy",
			"A forward cannot be prepared while a CMS session or another mailbox update is active.",
			parent,
		)
		return
	}

	go func() {
		draft, err := prepareForwardMessage(
			store,
			original,
		)

		if activity != nil {
			activity.endMutation()
		}

		fyne.Do(func() {
			if err != nil {
				dialog.ShowError(err, parent)
				return
			}

			openComposeMessageWithTitle(
				a,
				parent,
				store,
				draft,
				true,
				callsign,
				onChanged,
				"Forward Message — K2EXEmail",
			)
		})
	}()
}

func prepareForwardMessage(
	store messageForwardStore,
	original mailbox.Message,
) (mailbox.Message, error) {
	if store == nil {
		return mailbox.Message{}, fmt.Errorf(
			"mailbox store is required",
		)
	}

	draft, err := mailbox.NewDraft()
	if err != nil {
		return mailbox.Message{}, fmt.Errorf(
			"create forward draft: %w",
			err,
		)
	}

	draft = prepareForwardDraft(
		draft,
		original,
	)

	if err := store.Save(draft); err != nil {
		return mailbox.Message{},
			cleanupForwardDraft(
				store,
				draft.ID,
				fmt.Errorf(
					"save forward draft: %w",
					err,
				),
			)
	}

	for _, attachment := range original.Attachments {
		reader, sourceAttachment, err :=
			store.OpenAttachmentReader(
				original.Folder,
				original.ID,
				attachment.ID,
			)
		if err != nil {
			if reader != nil {
				_ = reader.Close()
			}

			return mailbox.Message{},
				cleanupForwardDraft(
					store,
					draft.ID,
					fmt.Errorf(
						"open attachment %q for forwarding: %w",
						attachment.Name,
						err,
					),
				)
		}

		copiedAttachment, copyErr :=
			store.AddAttachmentReader(
				mailbox.FolderDrafts,
				draft.ID,
				sourceAttachment.Name,
				reader,
			)

		closeErr := reader.Close()

		var attachmentErr error

		if copyErr != nil {
			attachmentErr = fmt.Errorf(
				"copy forwarded attachment %q: %w",
				sourceAttachment.Name,
				copyErr,
			)
		}

		if closeErr != nil {
			attachmentErr = errors.Join(
				attachmentErr,
				fmt.Errorf(
					"close forwarded attachment %q: %w",
					sourceAttachment.Name,
					closeErr,
				),
			)
		}

		if attachmentErr != nil {
			return mailbox.Message{},
				cleanupForwardDraft(
					store,
					draft.ID,
					attachmentErr,
				)
		}

		if sourceAttachment.Size >= 0 &&
			copiedAttachment.Size != sourceAttachment.Size {
			return mailbox.Message{},
				cleanupForwardDraft(
					store,
					draft.ID,
					fmt.Errorf(
						"forwarded attachment %q size mismatch: copied %d bytes, expected %d",
						sourceAttachment.Name,
						copiedAttachment.Size,
						sourceAttachment.Size,
					),
				)
		}

		if sourceAttachment.SHA256 != "" &&
			!strings.EqualFold(
				copiedAttachment.SHA256,
				sourceAttachment.SHA256,
			) {
			return mailbox.Message{},
				cleanupForwardDraft(
					store,
					draft.ID,
					fmt.Errorf(
						"forwarded attachment %q failed integrity verification",
						sourceAttachment.Name,
					),
				)
		}

		draft.Attachments = append(
			draft.Attachments,
			copiedAttachment,
		)
	}

	return draft, nil
}

func cleanupForwardDraft(
	store messageForwardStore,
	draftID string,
	cause error,
) error {
	if cleanupErr := store.Delete(
		mailbox.FolderDrafts,
		draftID,
	); cleanupErr != nil {
		return errors.Join(
			cause,
			fmt.Errorf(
				"clean up incomplete forward draft %q: %w",
				draftID,
				cleanupErr,
			),
		)
	}

	return cause
}

func prepareForwardDraft(
	draft mailbox.Message,
	original mailbox.Message,
) mailbox.Message {
	draft.Folder = mailbox.FolderDrafts
	draft.From = ""
	draft.To = nil
	draft.Cc = nil
	draft.Subject = forwardSubject(original.Subject)
	draft.Body = forwardBody(original)
	draft.Attachments = nil
	draft.WinlinkMID = ""
	draft.Starred = false
	draft.Unread = false
	draft.P2POnly = false

	return draft
}

func forwardSubject(subject string) string {
	subject = strings.TrimSpace(subject)

	if len(subject) >= 4 &&
		strings.EqualFold(subject[:4], "Fwd:") {
		return subject
	}

	if subject == "" {
		return "Fwd:"
	}

	return "Fwd: " + subject
}

func forwardBody(original mailbox.Message) string {
	from := strings.TrimSpace(original.From)
	if from == "" {
		from = "(unknown sender)"
	}

	var body strings.Builder

	body.WriteString(
		"\n\n---------- Forwarded message ----------\n",
	)
	fmt.Fprintf(&body, "From: %s\n", from)

	if !original.CreatedAt.IsZero() {
		fmt.Fprintf(
			&body,
			"Date: %s\n",
			original.CreatedAt.UTC().Format(
				"Mon, 02 Jan 2006 15:04 UTC",
			),
		)
	}

	fmt.Fprintf(
		&body,
		"Subject: %s\n",
		strings.TrimSpace(original.Subject),
	)

	if len(original.To) != 0 {
		fmt.Fprintf(
			&body,
			"To: %s\n",
			strings.Join(original.To, ", "),
		)
	}

	if len(original.Cc) != 0 {
		fmt.Fprintf(
			&body,
			"Cc: %s\n",
			strings.Join(original.Cc, ", "),
		)
	}

	body.WriteByte('\n')

	originalBody := strings.ReplaceAll(
		original.Body,
		"\r\n",
		"\n",
	)
	originalBody = strings.ReplaceAll(
		originalBody,
		"\r",
		"\n",
	)

	body.WriteString(originalBody)

	if originalBody != "" &&
		!strings.HasSuffix(originalBody, "\n") {
		body.WriteByte('\n')
	}

	return body.String()
}
