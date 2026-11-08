package widget

import (
	"math"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/ui/theme"
)

// HSV is a color as hue (0-360), saturation and value (0-1). Pickers keep
// it so a gray does not lose its hue while the user drags.
type HSV struct{ H, S, V float64 }

// Color is the sRGB color of h.
func (h HSV) Color() ui.Color {
	c := h.V * h.S
	x := c * (1 - math.Abs(math.Mod(h.H/60, 2)-1))
	m := h.V - c
	var r, g, b float64
	switch int(h.H/60) % 6 {
	case 0:
		r, g, b = c, x, 0
	case 1:
		r, g, b = x, c, 0
	case 2:
		r, g, b = 0, c, x
	case 3:
		r, g, b = 0, x, c
	case 4:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return ui.RGB(uint8(math.Round((r+m)*255)), uint8(math.Round((g+m)*255)), uint8(math.Round((b+m)*255)))
}

// HSVOf is col as hue, saturation and value.
func HSVOf(col ui.Color) HSV {
	r, g, b := float64(col.R)/255, float64(col.G)/255, float64(col.B)/255
	mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	d := mx - mn
	h := 0.0
	switch {
	case d == 0:
	case mx == r:
		h = 60 * math.Mod((g-b)/d, 6)
	case mx == g:
		h = 60 * ((b-r)/d + 2)
	default:
		h = 60 * ((r-g)/d + 4)
	}
	if h < 0 {
		h += 360
	}
	s := 0.0
	if mx > 0 {
		s = d / mx
	}
	return HSV{h, s, mx}
}

// ColorPicker is three sliders (hue, saturation, brightness) over a
// preview of the color, so nobody has to know a hex code. It edits *v and
// reports whether it changed.
func ColorPicker(c *ui.Context, v *HSV) bool {
	changed := false
	var hue []ui.GradientStop
	for i := 0; i <= 6; i++ {
		hue = append(hue, ui.GradientStop{Color: HSV{float64(i) * 60, 1, 1}.Color(), At: float32(i) / 6})
	}
	track := func(key, label string, val *float64, lo, hi float64, g ui.LinearGradient) {
		row := c.Key(key)
		ui.Row(row).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(row, label).FontSize(12).TextColor(theme.TextSoft).Width(78)
			cur := *val
			s := ui.SliderBase(row, &cur, lo, hi).Grow(1).Height(22).PaddingX(11).Radius(11).LinearGradient(g).
				Border(1, theme.BorderStrong).Cursor(ui.CursorPointer).Label(label)
			s.DrawOver(func(p *ui.Painter, r ui.Rect) {
				x := r.X + 11 + (r.W-22)*float32((cur-lo)/(hi-lo))
				p.Fill(ui.Rect{X: x - 10, Y: r.Y + 1, W: 20, H: 20}, ui.RGB(255, 255, 255), 10)
				p.Fill(ui.Rect{X: x - 8, Y: r.Y + 3, W: 16, H: 16}, v.Color(), 8)
			})
			if s.Changed() {
				*val, changed = cur, true
			}
		})
	}
	ui.Row(c).FillWidth().Gap(14).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(64, 64).Radius(12).Background(v.Color()).Border(1, theme.BorderStrong)
		ui.Column(c).Grow(1).MinWidth(0).Gap(10).Children(func() {
			track("hue", "Color", &v.H, 0, 360, ui.LinearGradient{Angle: 90, Stops: hue})
			track("sat", "Intensity", &v.S, 0, 1, ui.LinearGradient{Angle: 90,
				From: HSV{v.H, 0, v.V}.Color(), To: HSV{v.H, 1, v.V}.Color()})
			track("val", "Brightness", &v.V, 0, 1, ui.LinearGradient{Angle: 90,
				From: ui.RGB(0, 0, 0), To: HSV{v.H, v.S, 1}.Color()})
		})
	})
	ui.Text(c, theme.Hex(v.Color())).FontSize(11).TextColor(theme.Muted).Font(theme.MonoFont)
	return changed
}
