package main

import (
	"fydownloader/pkg/gui"
	"fyne.io/fyne/v2/app"
)

func main() {
	a := app.New()
	gui.NewDownloader(a)
	a.Run()
}
