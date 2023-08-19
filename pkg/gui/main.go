package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

var WindowSize = fyne.NewSize(800, 600)

type Downloader struct {
	app    fyne.App
	window fyne.Window
}

func NewDownloader(app fyne.App) *Downloader {
	d := &Downloader{
		app: app,
	}
	d.window = app.NewWindow("Downloader")
	d.window.Resize(WindowSize)

	menuBar := NewMenuBar()
	progressTab := NewProgressTab()

	c := container.NewBorder(menuBar.Container, nil, nil, nil, progressTab.Container)

	d.window.SetContent(c)

	d.window.Show()

	return d
}
