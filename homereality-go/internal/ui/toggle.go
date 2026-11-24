package ui

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// Toggle is a large switch with a smooth sliding knob (Windows-panel style).
// State: 0=off, 1=on, 2=busy
type Toggle struct {
	widget.BaseWidget
	State    int
	OnTapped func()

	knobPos float32 // 0 = left (off), 1 = right (on)
	anim    *fyne.Animation
}

func NewToggle(onTap func()) *Toggle {
	t := &Toggle{OnTapped: onTap, knobPos: 0}
	t.ExtendBaseWidget(t)
	return t
}

func (t *Toggle) SetState(s int) {
	if t.State == s {
		return
	}
	t.State = s
	var target float32
	switch s {
	case 1:
		target = 1
	case 2:
		target = 0.5
	default:
		target = 0
	}
	t.animateTo(target)
	t.Refresh()
}

func (t *Toggle) animateTo(target float32) {
	if t.anim != nil {
		t.anim.Stop()
	}
	start := t.knobPos
	t.anim = fyne.NewAnimation(180*time.Millisecond, func(f float32) {
		e := 1 - (1-f)*(1-f)*(1-f)
		t.knobPos = start + (target-start)*e
		t.Refresh()
	})
	t.anim.Curve = fyne.AnimationEaseOut
	t.anim.Start()
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
	minX := float32(4)
	maxX := size.Width - ks - 4
	x := minX + (maxX-minX)*r.t.knobPos
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
	canvas.Refresh(r.t)
}

func (r *toggleRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.bg, r.knob} }
func (r *toggleRenderer) Destroy()                     {}
