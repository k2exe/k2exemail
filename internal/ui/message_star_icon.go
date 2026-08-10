package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

var (
	messageStarOutlineIcon = theme.NewThemedResource(
		fyne.NewStaticResource(
			"k2exemail-star-outline.svg",
			[]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="#000000" d="M12 2.4l2.84 5.75 6.35.92-4.59 4.48 1.08 6.32L12 16.88l-5.68 2.99 1.08-6.32-4.59-4.48 6.35-.92L12 2.4zm0 3.39l-1.72 3.48-3.84.56 2.78 2.71-.66 3.83L12 14.56l3.44 1.81-.66-3.83 2.78-2.71-3.84-.56L12 5.79z"/></svg>`),
		),
	)

	messageStarFilledIcon = theme.NewPrimaryThemedResource(
		fyne.NewStaticResource(
			"k2exemail-star-filled.svg",
			[]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="#000000" d="M12 2.4l2.84 5.75 6.35.92-4.59 4.48 1.08 6.32L12 16.88l-5.68 2.99 1.08-6.32-4.59-4.48 6.35-.92L12 2.4z"/></svg>`),
		),
	)
)

func messageStarIcon(starred bool) fyne.Resource {
	if starred {
		return messageStarFilledIcon
	}

	return messageStarOutlineIcon
}
