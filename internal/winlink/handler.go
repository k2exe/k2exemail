package winlink

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/k2exe/k2exemail/internal/mailbox"
	"github.com/la5nta/wl2k-go/fbb"
)

type Handler struct {
	store  *mailbox.Store
	mycall string

	mu         sync.Mutex
	outbound   []*fbb.Message
	messageID  map[string]string
	deferred   map[string]bool
	completed  map[string]bool
	inboundMID map[string]bool
	sessionErr error
}

var _ fbb.MBoxHandler = (*Handler)(nil)

func NewHandler(store *mailbox.Store, mycall string) *Handler {
	return &Handler{
		store:      store,
		mycall:     mycall,
		inboundMID: make(map[string]bool),
	}
}

func (h *Handler) Prepare() error {
	if h.store == nil {
		return fmt.Errorf("mailbox store is required")
	}

	h.mu.Lock()
	h.outbound = nil
	h.messageID = make(map[string]string)
	h.deferred = make(map[string]bool)
	h.completed = make(map[string]bool)
	h.inboundMID = make(map[string]bool)
	h.sessionErr = nil
	h.mu.Unlock()

	if err := h.store.Prepare(); err != nil {
		return fmt.Errorf("prepare mailbox store: %w", err)
	}

	inboundMID, err := h.loadInboundMIDs()
	if err != nil {
		return err
	}

	h.mu.Lock()
	h.inboundMID = inboundMID
	h.mu.Unlock()

	messages, err := h.store.List(mailbox.FolderOutbox)
	if err != nil {
		return fmt.Errorf("load Outbox: %w", err)
	}

	prepared := make([]*fbb.Message, 0, len(messages))
	messageID := make(map[string]string, len(messages))

	for _, msg := range messages {
		wire, err := ToFBB(
			h.mycall,
			msg,
			h.loadAttachment,
		)
		if err != nil {
			h.recordError(
				fmt.Errorf(
					"prepare Outbox message %q: %w",
					msg.ID,
					err,
				),
			)
			continue
		}

		mid := wire.MID()
		if existingID, exists := messageID[mid]; exists {
			return fmt.Errorf(
				"duplicate Winlink MID %q for messages %q and %q",
				mid,
				existingID,
				msg.ID,
			)
		}

		// Persist a newly generated protocol MID before it can be
		// offered to the remote station. This ensures retries use
		// the same Winlink identity.
		if msg.WinlinkMID == "" {
			msg.WinlinkMID = mid

			if err := h.store.Save(msg); err != nil {
				return fmt.Errorf(
					"persist Winlink MID %q for message %q: %w",
					mid,
					msg.ID,
					err,
				)
			}
		}

		prepared = append(prepared, wire)
		messageID[mid] = msg.ID
	}

	h.mu.Lock()
	h.outbound = prepared
	h.messageID = messageID
	h.mu.Unlock()

	return nil
}

func (h *Handler) GetOutbound(
	forwarders ...fbb.Address,
) []*fbb.Message {
	h.mu.Lock()
	defer h.mu.Unlock()

	deliver := make([]*fbb.Message, 0, len(h.outbound))

	for _, msg := range h.outbound {
		mid := msg.MID()

		if h.deferred[mid] || h.completed[mid] {
			continue
		}

		if len(forwarders) > 0 {
			matched := false

			for _, forwarder := range forwarders {
				if msg.IsOnlyReceiver(forwarder) {
					matched = true
					break
				}
			}

			if !matched {
				continue
			}
		} else if msg.Header.Get(p2pOnlyHeader) == "true" {
			// No forwarding addresses means the remote is a CMS.
			continue
		}

		// X-P2POnly is private K2EXEmail/Pat-style mailbox metadata
		// and must not be transmitted as part of the message.
		msg.Header.Del(p2pOnlyHeader)

		deliver = append(deliver, msg)
	}

	return deliver
}

func (h *Handler) SetSent(mid string, _ bool) {
	h.mu.Lock()
	messageID, exists := h.messageID[mid]

	// Whether the remote accepted the body or rejected it because it
	// already has this MID, do not offer it again during this session.
	h.completed[mid] = true
	h.mu.Unlock()

	if !exists {
		h.recordError(
			fmt.Errorf(
				"mark Winlink MID %q sent: local message not found",
				mid,
			),
		)
		return
	}

	if err := h.store.Move(
		mailbox.FolderOutbox,
		mailbox.FolderSent,
		messageID,
	); err != nil {
		h.recordError(
			fmt.Errorf(
				"move sent message %q to Sent: %w",
				messageID,
				err,
			),
		)
	}
}

func (h *Handler) SetDeferred(mid string) {
	h.mu.Lock()
	h.deferred[mid] = true
	h.mu.Unlock()
}

func (h *Handler) Err() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.sessionErr
}

func (h *Handler) GetInboundAnswer(
	proposal fbb.Proposal,
) fbb.ProposalAnswer {
	mid := strings.TrimSpace(proposal.MID())
	if mid == "" {
		return fbb.Defer
	}

	h.mu.Lock()
	seen := h.inboundMID[mid]
	h.mu.Unlock()

	if seen {
		return fbb.Reject
	}

	return fbb.Accept
}

func (h *Handler) ProcessInbound(
	messages ...*fbb.Message,
) error {
	if h.store == nil {
		return fmt.Errorf("mailbox store is required")
	}

	for _, wire := range messages {
		msg, attachments, err := FromFBB(wire)
		if err != nil {
			return fmt.Errorf(
				"convert inbound Winlink message: %w",
				err,
			)
		}

		h.mu.Lock()
		seen := h.inboundMID[msg.WinlinkMID]
		h.mu.Unlock()

		if seen {
			continue
		}

		if _, err := h.store.ImportMessage(
			msg,
			attachments,
		); err != nil {
			return fmt.Errorf(
				"persist inbound Winlink message %q: %w",
				msg.WinlinkMID,
				err,
			)
		}

		h.mu.Lock()
		if h.inboundMID == nil {
			h.inboundMID = make(map[string]bool)
		}
		h.inboundMID[msg.WinlinkMID] = true
		h.mu.Unlock()
	}

	return nil
}

func (h *Handler) loadInboundMIDs() (map[string]bool, error) {
	seen := make(map[string]bool)

	folders := []mailbox.Folder{
		mailbox.FolderInbox,
		mailbox.FolderArchive,
		mailbox.FolderSpam,
		mailbox.FolderTrash,
	}

	for _, folder := range folders {
		messages, err := h.store.List(folder)
		if err != nil {
			return nil, fmt.Errorf(
				"load %s for Winlink duplicate detection: %w",
				folder,
				err,
			)
		}

		for _, msg := range messages {
			if mid := strings.TrimSpace(msg.WinlinkMID); mid != "" {
				seen[mid] = true
			}
		}
	}

	return seen, nil
}

func (h *Handler) loadAttachment(
	folder mailbox.Folder,
	messageID string,
	attachment mailbox.Attachment,
) ([]byte, error) {
	file, _, err := h.store.OpenAttachment(
		folder,
		messageID,
		attachment.ID,
	)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (h *Handler) recordError(err error) {
	if err == nil {
		return
	}

	h.mu.Lock()
	h.sessionErr = errors.Join(h.sessionErr, err)
	h.mu.Unlock()
}
