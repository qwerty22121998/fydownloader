package gui

import (
	"fmt"
	"fydownloader/pkg/core"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"image/color"
)

type ProgressTab struct {
	Container fyne.CanvasObject
	Progress  []*core.Downloader
}

func NewProgressTab() *ProgressTab {
	p := &ProgressTab{}

	table := widget.NewTable(func() (int, int) {
		return len(p.Progress) + 1, 7
	}, func() fyne.CanvasObject {
		c := container.NewWithoutLayout()
		r := canvas.NewRectangle(color.White)
		r.SetMinSize(fyne.NewSize(0, 25))
		r.Resize(fyne.NewSize(0, 25))
		c.Add(r)
		return container.NewHScroll(c)
	}, func(id widget.TableCellID, object fyne.CanvasObject) {
		rect := object.(*container.Scroll).Content.(*fyne.Container)
		rect.Resize(fyne.NewSize(800, 100))
		rect.RemoveAll()
		row := id.Row
		if row == 0 {
			txt := []string{"FileName", "FileSize", "URL", "TotalChunk", "", "", ""}[id.Col]
			rect.Add(widget.NewLabel(txt))
			object.Refresh()
			return
		}
		switch id.Col {
		case 0:
			rect.Add(widget.NewLabel(fmt.Sprint(p.Progress[row-1].FileInfo.FileName)))
		case 1:
			rect.Add(widget.NewLabel(fmt.Sprint(p.Progress[row-1].FileInfo.FileSize)))
		case 2:
			rect.Add(widget.NewLabel(fmt.Sprint(p.Progress[row-1].URL)))
		case 3:
			rect.Add(widget.NewLabel(fmt.Sprint(p.Progress[row-1].TotalChunk)))
		case 4:
			rect.Add(widget.NewLabel(fmt.Sprint(p.Progress[row-1].FileInfo.FileName)))
		case 5:
			rect.Add(widget.NewLabel(fmt.Sprint(p.Progress[row-1].FileInfo.FileName)))
		case 6:
			rect.Add(widget.NewLabel(fmt.Sprint(p.Progress[row-1].FileInfo.FileName)))
		}
		object.Refresh()
	})
	sizes := []float32{150, 100, 150, 100, 100, 100, 100}
	for i, size := range sizes {
		table.SetColumnWidth(i, size)
	}

	for i := 0; i < 10; i++ {
		d, _ := core.NewDownloader("https://nodejs.org/dist/v18.17.1/node-v18.17.1-x64.msi", 1<<4)
		p.Progress = append(p.Progress, d)
	}

	p.Container = container.NewMax(container.NewScroll(table))

	return p
}
