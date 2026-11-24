package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// Toggle is a large iOS-style switch (matches Windows ToggleSwitch size/feel).
// State: 0=off, 1=on, 2=busy
type Toggle struct {
	widget.BaseWidget
	State    int
	OnTapped func()
}

func NewToggle(onTap func()) *Toggle {
	t := &Toggle{OnTapped: onTap}
	t.ExtendBaseWidget(t)
	return t
}

func (t *Toggle) SetState(s int) {
	t.State = s
	t.Refresh()
}

func (t *Toggle) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(ColOff)
	bg.CornerRadius = 20
	knob := canvas.NewCircle(color.White)
	return &toggleRenderer{t: t, bg: bg, knob: knob}
}

func (t *Toggle) Tapped(*fyne.PointEvent) {
	if t.State == 2 {
		return
	}
	if t.OnTapped != nil {
		t.OnTapped()
	}
}

func (t *Toggle) Cursor() desktop.Cursor { return desktop.PointerCursor }

type toggleRenderer struct {
	t    *Toggle
	bg   *canvas.Rectangle
	knob *canvas.Circle
}

func (r *toggleRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.bg.Move(fyne.NewPos(0, 0))
	ks := size.Height - 8
	if ks < 12 {
		ks = size.Height * 0.7
	}
	r.knob.Resize(fyne.NewSize(ks, ks))
	var x float32
	switch r.t.State {
	case 1: // on — right
		x = size.Width - ks - 4
	case 2: // busy — middle
		x = (size.Width - ks) / 2
	default: // off — left
		x = 4
	}
	y := (size.Height - ks) / 2
	r.knob.Move(fyne.NewPos(x, y))
}

func (r *toggleRenderer) MinSize() fyne.Size { return fyne.NewSize(120, 56) }

func (r *toggleRenderer) Refresh() {
	switch r.t.State {
	case 1:
		r.bg.FillColor = ColGreen
	case 2:
		r.bg.FillColor = ColAmber
	default:
		r.bg.FillColor = ColOff
	}
	r.bg.Refresh()
	r.knob.Refresh()
	r.Layout(r.t.Size())
}

func (r *toggleRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.bg, r.knob} }
func (r *toggleRenderer) Destroy()                     {}
