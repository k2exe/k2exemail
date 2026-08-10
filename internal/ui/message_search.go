package ui

import (
	"strings"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

func filterMessages(
	messages []mailbox.Message,
	query string,
) []mailbox.Message {
	query = strings.ToLower(strings.TrimSpace(query))

	if query == "" {
		return messages
	}

	filtered := make(
		[]mailbox.Message,
		0,
		len(messages),
	)

	for _, msg := range messages {
		if messageMatchesQuery(msg, query) {
			filtered = append(filtered, msg)
		}
	}

	return filtered
}

func messageMatchesQuery(
	msg mailbox.Message,
	normalizedQuery string,
) bool {
	if normalizedQuery == "" {
		return true
	}

	fields := []string{
		msg.From,
		strings.Join(msg.To, " "),
		strings.Join(msg.Cc, " "),
		msg.Subject,
		msg.Body,
	}

	for _, field := range fields {
		if strings.Contains(
			strings.ToLower(field),
			normalizedQuery,
		) {
			return true
		}
	}

	return false
}
