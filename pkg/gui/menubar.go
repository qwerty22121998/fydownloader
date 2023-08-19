package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type Menubar struct {
	window    fyne.Window
	AddURLBtn *widget.Button
	PauseBtn  *widget.Button
	ResumeBtn *widget.Button
	Container *fyne.Container
}

func NewMenuBar() *Menubar {
	m := &Menubar{}
	m.Container = container.New(layout.NewGridLayout(8))

	m.AddURLBtn = widget.NewButton("Add URL", func() {

	})

	m.PauseBtn = widget.NewButton("Pause", func() {

	})

	m.ResumeBtn = widget.NewButton("Resume", func() {

	})

	m.Container.Add(m.AddURLBtn)
	m.Container.Add(m.PauseBtn)
	m.Container.Add(m.ResumeBtn)

	return m
}
