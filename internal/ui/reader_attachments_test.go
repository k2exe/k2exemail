package ui

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

type attachmentReaderStub struct {
	reader     io.ReadCloser
	attachment mailbox.Attachment
	err        error
}

func (s *attachmentReaderStub) OpenAttachmentReader(
	mailbox.Folder,
	string,
	string,
) (io.ReadCloser, mailbox.Attachment, error) {
	return s.reader, s.attachment, s.err
}

type trackedReadCloser struct {
	*strings.Reader
	closed bool
}

func (r *trackedReadCloser) Close() error {
	r.closed = true
	return nil
}

type trackedWriteCloser struct {
	bytes.Buffer
	closed bool
}

func (w *trackedWriteCloser) Close() error {
	w.closed = true
	return nil
}

func TestSaveAttachmentToWriterCopiesAndCloses(t *testing.T) {
	source := &trackedReadCloser{
		Reader: strings.NewReader("attachment data"),
	}
	destination := &trackedWriteCloser{}

	store := &attachmentReaderStub{
		reader: source,
		attachment: mailbox.Attachment{
			ID:   "attachment-1",
			Name: "test.txt",
			Size: int64(len("attachment data")),
		},
	}

	err := saveAttachmentToWriter(
		source,
		store.attachment,
		destination,
	)
	if err != nil {
		t.Fatalf(
			"saveAttachmentToWriter() error = %v",
			err,
		)
	}

	if got := destination.String(); got != "attachment data" {
		t.Fatalf(
			"saved data = %q, want attachment data",
			got,
		)
	}

	if !source.closed {
		t.Fatal("source reader was not closed")
	}

	if !destination.closed {
		t.Fatal("destination writer was not closed")
	}
}

func TestSaveAttachmentToWriterRejectsSizeMismatch(
	t *testing.T,
) {
	source := &trackedReadCloser{
		Reader: strings.NewReader("short"),
	}
	destination := &trackedWriteCloser{}

	store := &attachmentReaderStub{
		reader: source,
		attachment: mailbox.Attachment{
			ID:   "attachment-1",
			Name: "test.txt",
			Size: 20,
		},
	}

	err := saveAttachmentToWriter(
		source,
		store.attachment,
		destination,
	)
	if err == nil {
		t.Fatal(
			"saveAttachmentToWriter() expected size mismatch",
		)
	}

	if !source.closed || !destination.closed {
		t.Fatal(
			"reader and writer must close after size mismatch",
		)
	}
}

func TestSaveAttachmentToWriterRejectsNilReader(
	t *testing.T,
) {
	destination := &trackedWriteCloser{}

	err := saveAttachmentToWriter(
		nil,
		mailbox.Attachment{
			ID:   "attachment-1",
			Name: "test.txt",
		},
		destination,
	)
	if err == nil {
		t.Fatal(
			"saveAttachmentToWriter() expected nil reader error",
		)
	}

	if !destination.closed {
		t.Fatal(
			"destination writer was not closed",
		)
	}
}

func TestSafeAttachmentFileName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"report.txt", "report.txt"},
		{"../escape.txt", "escape.txt"},
		{`C:\temp\report.txt`, "report.txt"},
		{"../../", "attachment"},
		{"bad:name?.txt", "bad_name_.txt"},
		{"evil\x00name.txt", "evil_name.txt"},
		{"CON.txt", "_CON.txt"},
		{"NUL", "_NUL"},
		{"LPT1.log", "_LPT1.log"},
	}

	for _, tt := range tests {
		if got := safeAttachmentFileName(tt.name); got != tt.want {
			t.Fatalf(
				"safeAttachmentFileName(%q) = %q, want %q",
				tt.name,
				got,
				tt.want,
			)
		}
	}
}
