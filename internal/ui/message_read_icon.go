package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

var (
	messageMarkReadIcon = theme.NewThemedResource(
		fyne.NewStaticResource(
			"k2exemail-mark-read.svg",
			[]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="#000000" d="M12 2l9 7v11H3V9l9-7zm0 2.55L5.3 9.76 12 14.55l6.7-4.79L12 4.55zM5 11.4V18h14v-6.6l-7 5-7-5z"/></svg>`),
		),
	)

	messageMarkUnreadIcon = theme.NewThemedResource(
		fyne.NewStaticResource(
			"k2exemail-mark-unread.svg",
			[]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="#000000" d="M3 5h18v14H3V5zm2 2v.5l7 5 7-5V7H5zm14 10v-7l-7 5-7-5v7h14z"/></svg>`),
		),
	)
)

func messageReadActionIcon(unread bool) fyne.Resource {
	if unread {
		return messageMarkReadIcon
	}

	return messageMarkUnreadIcon
}
