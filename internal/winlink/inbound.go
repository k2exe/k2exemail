package winlink

import (
	"fmt"
	"strings"

	"github.com/k2exe/k2exemail/internal/mailbox"
	"github.com/la5nta/wl2k-go/fbb"
)

func FromFBB(
	wire *fbb.Message,
) (mailbox.Message, []mailbox.AttachmentContent, error) {
	if wire == nil {
		return mailbox.Message{}, nil, fmt.Errorf(
			"Winlink message is required",
		)
	}

	mid := strings.TrimSpace(wire.MID())
	if mid == "" {
		return mailbox.Message{}, nil, fmt.Errorf(
			"inbound Winlink message has no MID",
		)
	}

	msg, err := mailbox.NewMessage(mailbox.FolderInbox)
	if err != nil {
		return mailbox.Message{}, nil, err
	}

	msg.WinlinkMID = mid
	msg.From = wire.From().String()
	msg.To = addressesToStrings(wire.To())
	msg.Cc = addressesToStrings(wire.Cc())
	msg.Subject = wire.Subject()
	msg.Unread = true

	// Body() provides decoded UTF-8 where possible and fallback content
	// with an error otherwise. Preserve the content rather than losing an
	// otherwise successfully received Winlink message.
	msg.Body, _ = wire.Body()

	if date := wire.Date(); !date.IsZero() {
		msg.CreatedAt = date.UTC()
	}

	files := wire.Files()
	attachments := make(
		[]mailbox.AttachmentContent,
		0,
		len(files),
	)

	for _, file := range files {
		if file == nil {
			return mailbox.Message{}, nil, fmt.Errorf(
				"inbound Winlink message %q contains nil attachment",
				mid,
			)
		}

		if strings.TrimSpace(file.Name()) == "" {
			return mailbox.Message{}, nil, fmt.Errorf(
				"inbound Winlink message %q contains unnamed attachment",
				mid,
			)
		}

		attachments = append(
			attachments,
			mailbox.AttachmentContent{
				Name: file.Name(),
				Data: file.Data(),
			},
		)
	}

	return msg, attachments, nil
}

func addressesToStrings(
	addresses []fbb.Address,
) []string {
	if len(addresses) == 0 {
		return nil
	}

	result := make([]string, 0, len(addresses))

	for _, address := range addresses {
		result = append(result, address.String())
	}

	return result
}
