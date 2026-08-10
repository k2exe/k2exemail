package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

func openReplyWindow(
	a fyne.App,
	parent fyne.Window,
	store mailboxStore,
	original mailbox.Message,
	callsign string,
	replyAll bool,
	onChanged func(),
) {
	draft, err := mailbox.NewDraft()
	if err != nil {
		dialog.ShowError(
			fmt.Errorf("create reply draft: %w", err),
			parent,
		)
		return
	}

	draft, err = prepareReplyDraft(
		draft,
		original,
		callsign,
		replyAll,
	)
	if err != nil {
		dialog.ShowError(err, parent)
		return
	}

	openComposeMessage(
		a,
		parent,
		store,
		draft,
		false,
		callsign,
		onChanged,
	)
}

func prepareReplyDraft(
	draft mailbox.Message,
	original mailbox.Message,
	callsign string,
	replyAll bool,
) (mailbox.Message, error) {
	to, cc := replyRecipients(
		original,
		callsign,
		replyAll,
	)

	if len(to)+len(cc) == 0 {
		return mailbox.Message{}, fmt.Errorf(
			"reply to message %q: no recipient available",
			original.ID,
		)
	}

	draft.To = to
	draft.Cc = cc
	draft.Subject = replySubject(original.Subject)
	draft.Body = replyBody(original)

	// A reply is a new message. Do not inherit protocol identity,
	// attachments, or P2P-only state from the original.
	draft.WinlinkMID = ""
	draft.Attachments = nil
	draft.P2POnly = false

	return draft, nil
}

func replyRecipients(
	original mailbox.Message,
	callsign string,
	replyAll bool,
) ([]string, []string) {
	me := strings.TrimSpace(callsign)
	from := strings.TrimSpace(original.From)

	var to []string
	var cc []string
	seen := make(map[string]struct{})

	add := func(
		target *[]string,
		address string,
	) {
		address = strings.TrimSpace(address)
		if address == "" {
			return
		}
		if me != "" &&
			strings.EqualFold(address, me) {
			return
		}

		key := strings.ToUpper(address)
		if _, exists := seen[key]; exists {
			return
		}

		seen[key] = struct{}{}
		*target = append(*target, address)
	}

	fromIsMe := from != "" &&
		me != "" &&
		strings.EqualFold(from, me)

	if !replyAll {
		if from != "" && !fromIsMe {
			add(&to, from)
			return to, nil
		}

		for _, address := range original.To {
			add(&to, address)
			if len(to) != 0 {
				return to, nil
			}
		}

		for _, address := range original.Cc {
			add(&to, address)
			if len(to) != 0 {
				return to, nil
			}
		}

		return nil, nil
	}

	if from != "" && !fromIsMe {
		add(&to, from)

		for _, address := range original.To {
			add(&cc, address)
		}
		for _, address := range original.Cc {
			add(&cc, address)
		}

		return to, cc
	}

	// For a message sent by us, Reply All should preserve the
	// original To/Cc roles rather than replying to ourselves.
	for _, address := range original.To {
		add(&to, address)
	}
	for _, address := range original.Cc {
		add(&cc, address)
	}

	return to, cc
}

func replySubject(subject string) string {
	subject = strings.TrimSpace(subject)

	if len(subject) >= 3 &&
		strings.EqualFold(subject[:3], "Re:") {
		return subject
	}

	if subject == "" {
		return "Re:"
	}

	return "Re: " + subject
}

func replyBody(original mailbox.Message) string {
	from := strings.TrimSpace(original.From)
	if from == "" {
		from = "(unknown sender)"
	}

	var body strings.Builder
	body.WriteString("\n\n")

	if original.CreatedAt.IsZero() {
		fmt.Fprintf(
			&body,
			"%s wrote:\n",
			from,
		)
	} else {
		fmt.Fprintf(
			&body,
			"On %s, %s wrote:\n",
			original.CreatedAt.UTC().Format(
				"Mon, 02 Jan 2006 15:04 UTC",
			),
			from,
		)
	}

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

	for _, line := range strings.Split(
		originalBody,
		"\n",
	) {
		if line == "" {
			body.WriteString(">\n")
			continue
		}

		body.WriteString("> ")
		body.WriteString(line)
		body.WriteByte('\n')
	}

	return body.String()
}
