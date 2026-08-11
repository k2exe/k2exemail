package ui

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

type mailViewStoreStub struct {
	messages map[mailbox.Folder][]mailbox.Message
	errs     map[mailbox.Folder]error
	calls    []mailbox.Folder
}

func (s *mailViewStoreStub) List(
	folder mailbox.Folder,
) ([]mailbox.Message, error) {
	s.calls = append(s.calls, folder)

	if err := s.errs[folder]; err != nil {
		return nil, err
	}

	return s.messages[folder], nil
}

func TestLoadMailViewFolder(t *testing.T) {
	want := []mailbox.Message{
		{
			ID:     "inbox-1",
			Folder: mailbox.FolderInbox,
		},
	}

	store := &mailViewStoreStub{
		messages: map[mailbox.Folder][]mailbox.Message{
			mailbox.FolderInbox: want,
		},
	}

	got, err := loadMailView(
		store,
		folderMailView(mailbox.FolderInbox),
	)
	if err != nil {
		t.Fatalf("loadMailView() error = %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("loadMailView() = %#v, want %#v", got, want)
	}

	wantCalls := []mailbox.Folder{
		mailbox.FolderInbox,
	}

	if !reflect.DeepEqual(store.calls, wantCalls) {
		t.Fatalf(
			"List() calls = %#v, want %#v",
			store.calls,
			wantCalls,
		)
	}
}

func TestLoadMailViewStarredAggregatesAndSorts(t *testing.T) {
	base := time.Date(
		2026, 8, 10, 12, 0, 0, 0, time.UTC,
	)

	store := &mailViewStoreStub{
		messages: map[mailbox.Folder][]mailbox.Message{
			mailbox.FolderInbox: {
				{
					ID:        "inbox-star",
					Folder:    mailbox.FolderInbox,
					Starred:   true,
					UpdatedAt: base.Add(time.Minute),
				},
				{
					ID:        "inbox-plain",
					Folder:    mailbox.FolderInbox,
					Starred:   false,
					UpdatedAt: base.Add(10 * time.Minute),
				},
			},
			mailbox.FolderSent: {
				{
					ID:        "sent-star",
					Folder:    mailbox.FolderSent,
					Starred:   true,
					UpdatedAt: base.Add(3 * time.Minute),
				},
			},
			mailbox.FolderArchive: {
				{
					ID:        "archive-star",
					Folder:    mailbox.FolderArchive,
					Starred:   true,
					UpdatedAt: base.Add(2 * time.Minute),
				},
			},
			mailbox.FolderSpam: {
				{
					ID:        "spam-star",
					Folder:    mailbox.FolderSpam,
					Starred:   true,
					UpdatedAt: base.Add(30 * time.Minute),
				},
			},
			mailbox.FolderTrash: {
				{
					ID:        "trash-star",
					Folder:    mailbox.FolderTrash,
					Starred:   true,
					UpdatedAt: base.Add(40 * time.Minute),
				},
			},
			mailbox.FolderDrafts: {
				{
					ID:        "draft-star",
					Folder:    mailbox.FolderDrafts,
					Starred:   true,
					UpdatedAt: base.Add(20 * time.Minute),
				},
			},
		},
	}

	got, err := loadMailView(
		store,
		starredMailView(),
	)
	if err != nil {
		t.Fatalf("loadMailView() error = %v", err)
	}

	gotIDs := make([]string, len(got))
	for i := range got {
		gotIDs[i] = got[i].ID
	}

	wantIDs := []string{
		"sent-star",
		"archive-star",
		"inbox-star",
	}

	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf(
			"starred message IDs = %#v, want %#v",
			gotIDs,
			wantIDs,
		)
	}

	if !reflect.DeepEqual(
		store.calls,
		starredViewFolders,
	) {
		t.Fatalf(
			"List() calls = %#v, want %#v",
			store.calls,
			starredViewFolders,
		)
	}
}

func TestLoadMailViewStarredWrapsFolderError(t *testing.T) {
	loadErr := errors.New("read failed")

	store := &mailViewStoreStub{
		errs: map[mailbox.Folder]error{
			mailbox.FolderSent: loadErr,
		},
	}

	_, err := loadMailView(
		store,
		starredMailView(),
	)
	if err == nil {
		t.Fatal("loadMailView() expected error")
	}

	if !errors.Is(err, loadErr) {
		t.Fatalf(
			"loadMailView() error = %v, want wrapped %v",
			err,
			loadErr,
		)
	}

	wantCalls := []mailbox.Folder{
		mailbox.FolderInbox,
		mailbox.FolderOutbox,
		mailbox.FolderSent,
	}

	if !reflect.DeepEqual(store.calls, wantCalls) {
		t.Fatalf(
			"List() calls = %#v, want %#v",
			store.calls,
			wantCalls,
		)
	}
}

func TestMailViewProperties(t *testing.T) {
	inbox := folderMailView(mailbox.FolderInbox)

	if inbox.title() != "Inbox" {
		t.Fatalf(
			"inbox title = %q, want Inbox",
			inbox.title(),
		)
	}

	if inbox.isStarred() {
		t.Fatal("Inbox unexpectedly reports Starred view")
	}

	if inbox.isDrafts() {
		t.Fatal("Inbox unexpectedly reports Drafts view")
	}

	drafts := folderMailView(mailbox.FolderDrafts)
	if !drafts.isDrafts() {
		t.Fatal("Drafts view did not report Drafts")
	}

	starred := starredMailView()

	if starred.title() != "Starred" {
		t.Fatalf(
			"starred title = %q, want Starred",
			starred.title(),
		)
	}

	if !starred.isStarred() {
		t.Fatal("Starred view did not report Starred")
	}

	if starred.isDrafts() {
		t.Fatal("Starred unexpectedly reports Drafts view")
	}
}

func TestReplaceMessageSnapshot(t *testing.T) {
	messages := []mailbox.Message{
		{
			ID:      "message-1",
			Folder:  mailbox.FolderInbox,
			Subject: "Original",
			Starred: false,
		},
		{
			ID:      "message-2",
			Folder:  mailbox.FolderSent,
			Subject: "Other",
			Starred: false,
		},
	}

	updated := messages[0]
	updated.Starred = true

	if !replaceMessageSnapshot(messages, updated) {
		t.Fatal("replaceMessageSnapshot() did not find message")
	}

	if !messages[0].Starred {
		t.Fatal("updated message Starred = false, want true")
	}

	if messages[1].Starred {
		t.Fatal("unrelated message was changed")
	}

	missing := mailbox.Message{
		ID:     "missing",
		Folder: mailbox.FolderInbox,
	}

	if replaceMessageSnapshot(messages, missing) {
		t.Fatal("replaceMessageSnapshot() found missing message")
	}
}

func TestRemoveMessageSnapshot(t *testing.T) {
	messages := []mailbox.Message{
		{
			ID:      "message-1",
			Folder:  mailbox.FolderInbox,
			Subject: "Inbox one",
		},
		{
			ID:      "message-1",
			Folder:  mailbox.FolderArchive,
			Subject: "Archive one",
		},
		{
			ID:      "message-2",
			Folder:  mailbox.FolderInbox,
			Subject: "Inbox two",
		},
	}

	got, removed := removeMessageSnapshot(
		messages,
		mailbox.Message{
			ID:     "message-1",
			Folder: mailbox.FolderInbox,
		},
	)
	if !removed {
		t.Fatal("removeMessageSnapshot() removed = false, want true")
	}

	want := []mailbox.Message{
		{
			ID:      "message-1",
			Folder:  mailbox.FolderArchive,
			Subject: "Archive one",
		},
		{
			ID:      "message-2",
			Folder:  mailbox.FolderInbox,
			Subject: "Inbox two",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"removeMessageSnapshot() = %#v, want %#v",
			got,
			want,
		)
	}

	if len(messages) != 3 {
		t.Fatalf(
			"original slice length = %d, want 3",
			len(messages),
		)
	}

	notFound, removed := removeMessageSnapshot(
		messages,
		mailbox.Message{
			ID:     "missing",
			Folder: mailbox.FolderInbox,
		},
	)
	if removed {
		t.Fatal("missing message removed = true, want false")
	}
	if !reflect.DeepEqual(notFound, messages) {
		t.Fatal("missing-message result changed original messages")
	}
}
