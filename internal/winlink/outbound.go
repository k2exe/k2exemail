package winlink

import (
	"fmt"
	"strings"

	"github.com/k2exe/k2exemail/internal/mailbox"
	"github.com/la5nta/wl2k-go/fbb"
)

const p2pOnlyHeader = "X-P2POnly"

type AttachmentLoader func(
	folder mailbox.Folder,
	messageID string,
	attachment mailbox.Attachment,
) ([]byte, error)

func ToFBB(
	mycall string,
	msg mailbox.Message,
	loadAttachment AttachmentLoader,
) (*fbb.Message, error) {
	mycall = strings.ToUpper(strings.TrimSpace(mycall))
	if mycall == "" {
		return nil, fmt.Errorf("station callsign is required")
	}

	wire := fbb.NewMessage(fbb.Private, mycall)

	if mid := strings.TrimSpace(msg.WinlinkMID); mid != "" {
		wire.Header.Set(fbb.HEADER_MID, mid)
	}

	if !msg.CreatedAt.IsZero() {
		wire.SetDate(msg.CreatedAt)
	}

	from := strings.TrimSpace(msg.From)
	if from == "" {
		from = mycall
	}
	wire.SetFrom(from)

	for _, addr := range msg.To {
		if addr = strings.TrimSpace(addr); addr != "" {
			wire.AddTo(addr)
		}
	}

	for _, addr := range msg.Cc {
		if addr = strings.TrimSpace(addr); addr != "" {
			wire.AddCc(addr)
		}
	}

	wire.SetSubject(msg.Subject)

	if err := wire.SetBody(msg.Body); err != nil {
		return nil, fmt.Errorf(
			"encode body for message %q: %w",
			msg.ID,
			err,
		)
	}

	if msg.P2POnly {
		wire.Header.Set(p2pOnlyHeader, "true")
	}

	if len(msg.Attachments) > 0 && loadAttachment == nil {
		return nil, fmt.Errorf(
			"message %q has attachments but no attachment loader",
			msg.ID,
		)
	}

	for _, attachment := range msg.Attachments {
		if strings.TrimSpace(attachment.Name) == "" {
			return nil, fmt.Errorf(
				"message %q contains attachment %q with empty name",
				msg.ID,
				attachment.ID,
			)
		}

		data, err := loadAttachment(
			msg.Folder,
			msg.ID,
			attachment,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"load attachment %q for message %q: %w",
				attachment.Name,
				msg.ID,
				err,
			)
		}

		if int64(len(data)) != attachment.Size {
			return nil, fmt.Errorf(
				"attachment %q size mismatch: loaded %d bytes, expected %d",
				attachment.Name,
				len(data),
				attachment.Size,
			)
		}

		wire.AddFile(
			fbb.NewFile(attachment.Name, data),
		)
	}

	if err := wire.Validate(); err != nil {
		return nil, fmt.Errorf(
			"validate Winlink message %q: %w",
			msg.ID,
			err,
		)
	}

	return wire, nil
}
