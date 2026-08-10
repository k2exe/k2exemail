package ui

import (
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/k2exe/k2exemail/internal/mailbox"
)

type attachmentReaderStore interface {
	OpenAttachmentReader(
		folder mailbox.Folder,
		messageID string,
		attachmentID string,
	) (io.ReadCloser, mailbox.Attachment, error)
}

func newReaderAttachments(
	parent fyne.Window,
	store attachmentReaderStore,
	folder mailbox.Folder,
) (fyne.CanvasObject, func(mailbox.Message)) {
	summary := widget.NewLabel("")
	rows := container.NewVBox()

	section := container.NewVBox(
		summary,
		rows,
	)
	section.Hide()

	showMessage := func(msg mailbox.Message) {
		rows.RemoveAll()

		if len(msg.Attachments) == 0 {
			section.Hide()
			return
		}

		summary.SetText(
			attachmentSummaryText(msg.Attachments),
		)

		for _, attachment := range msg.Attachments {
			attachment := attachment

			label := widget.NewLabel(
				fmt.Sprintf(
					"%s  (%s)",
					attachment.Name,
					formatAttachmentSize(attachment.Size),
				),
			)
			label.Truncation = fyne.TextTruncateEllipsis

			saveButton := widget.NewButtonWithIcon(
				"Save As...",
				theme.DownloadIcon(),
				func() {
					showAttachmentSaveDialog(
						parent,
						store,
						folder,
						msg.ID,
						attachment,
					)
				},
			)

			row := container.NewBorder(
				nil,
				nil,
				widget.NewIcon(theme.MailAttachmentIcon()),
				saveButton,
				label,
			)

			rows.Add(row)
		}

		rows.Refresh()
		section.Show()
	}

	return section, showMessage
}

func showAttachmentSaveDialog(
	parent fyne.Window,
	store attachmentReaderStore,
	folder mailbox.Folder,
	messageID string,
	attachment mailbox.Attachment,
) {
	go func() {
		reader, storedAttachment, err :=
			store.OpenAttachmentReader(
				folder,
				messageID,
				attachment.ID,
			)

		fyne.Do(func() {
			if err != nil {
				dialog.ShowError(
					fmt.Errorf(
						"open attachment %q: %w",
						attachment.Name,
						err,
					),
					parent,
				)
				return
			}

			if reader == nil {
				dialog.ShowError(
					fmt.Errorf(
						"open attachment %q: store returned a nil reader",
						attachment.Name,
					),
					parent,
				)
				return
			}

			picker := dialog.NewFileSave(
				func(
					writer fyne.URIWriteCloser,
					saveErr error,
				) {
					if saveErr != nil {
						_ = reader.Close()

						if writer != nil {
							_ = writer.Close()
						}

						dialog.ShowError(
							fmt.Errorf(
								"choose save location for %q: %w",
								storedAttachment.Name,
								saveErr,
							),
							parent,
						)
						return
					}

					if writer == nil {
						_ = reader.Close()
						return
					}

					destinationName :=
						safeAttachmentFileName(
							storedAttachment.Name,
						)

					if uri := writer.URI(); uri != nil {
						if name := strings.TrimSpace(
							uri.Name(),
						); name != "" {
							destinationName = name
						}
					}

					go func() {
						err := saveAttachmentToWriter(
							reader,
							storedAttachment,
							writer,
						)

						fyne.Do(func() {
							if err != nil {
								dialog.ShowError(
									err,
									parent,
								)
								return
							}

							dialog.ShowInformation(
								"Attachment saved",
								fmt.Sprintf(
									"Saved as %s.",
									destinationName,
								),
								parent,
							)
						})
					}()
				},
				parent,
			)

			picker.SetTitleText("Save Attachment")
			picker.SetConfirmText("Save")
			picker.SetFileName(
				safeAttachmentFileName(
					storedAttachment.Name,
				),
			)
			picker.Show()
		})
	}()
}

func saveAttachmentToWriter(
	reader io.ReadCloser,
	attachment mailbox.Attachment,
	writer io.WriteCloser,
) (err error) {
	if writer == nil {
		return fmt.Errorf(
			"attachment destination writer is required",
		)
	}

	defer func() {
		if closeErr := writer.Close(); closeErr != nil {
			err = errors.Join(
				err,
				fmt.Errorf(
					"close attachment destination: %w",
					closeErr,
				),
			)
		}
	}()

	if reader == nil {
		return fmt.Errorf(
			"attachment source reader is required",
		)
	}

	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			err = errors.Join(
				err,
				fmt.Errorf(
					"close attachment source: %w",
					closeErr,
				),
			)
		}
	}()

	copied, err := io.Copy(writer, reader)
	if err != nil {
		return fmt.Errorf(
			"save attachment %q: %w",
			attachment.Name,
			err,
		)
	}

	if attachment.Size >= 0 &&
		copied != attachment.Size {
		return fmt.Errorf(
			"save attachment %q: copied %d bytes, expected %d",
			attachment.Name,
			copied,
			attachment.Size,
		)
	}

	return nil
}

func safeAttachmentFileName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, `\`, "/")
	name = path.Base(name)

	if name == "" ||
		name == "." ||
		name == ".." ||
		name == "/" {
		return "attachment"
	}

	name = strings.Map(
		func(r rune) rune {
			switch {
			case r < 0x20:
				return '_'
			case r == 0x7f:
				return '_'
			case strings.ContainsRune(
				`<>:"|?*`,
				r,
			):
				return '_'
			default:
				return r
			}
		},
		name,
	)

	name = strings.TrimSpace(name)
	name = strings.TrimRight(name, ". ")

	if name == "" {
		return "attachment"
	}

	if windowsReservedFileName(name) {
		return "_" + name
	}

	return name
}

func windowsReservedFileName(name string) bool {
	stem := name
	if dot := strings.IndexByte(stem, '.'); dot >= 0 {
		stem = stem[:dot]
	}

	stem = strings.ToUpper(stem)

	switch stem {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}

	if len(stem) == 4 {
		prefix := stem[:3]
		digit := stem[3]

		if (prefix == "COM" || prefix == "LPT") &&
			digit >= '1' &&
			digit <= '9' {
			return true
		}
	}

	return false
}
