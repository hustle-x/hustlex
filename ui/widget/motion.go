package widget

import (
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/ui/theme"
)

// The motion of the app. Colors ease out fast and land softly; things that
// move or resize ride a spring with a trace of bounce, so a change that
// interrupts another carries on from where it was instead of restarting.
// MyGo honors the desktop's Reduce Motion: springs and moves jump, colors
// still ease.
var (
	// Out is a quint ease-out: quick to start, long to settle.
	Out = ui.CubicBezier(0.22, 1, 0.36, 1)
	// Spring moves and resizes panels, pills and pop-ups.
	Spring = ui.Animation{Spring: ui.SpringOptions{Duration: 460 * time.Millisecond, Bounce: 0.12}}
	// Press is the quick spring of a pressed control.
	Press = ui.Animation{Spring: ui.SpringOptions{Duration: 260 * time.Millisecond, Bounce: 0.2}}
)

// ease is how every control's colors change.
var ease = ui.Animation{Duration: 220 * time.Millisecond, Ease: Out}

// Pressable shrinks e a little while pressed and springs it back.
func Pressable(e ui.Element) ui.Element {
	scale := float32(1)
	if e.Pressed() && !e.IsDisabled() {
		scale = 0.965
	}
	// Fade 0 is opaque: a Pose fades from 0 to 1, gone.
	return e.AnimatePose(ui.Pose{Scale: scale}, Press)
}

// pill tells apart the selection pills of different lists.
type pill struct{ group string }

// SelectionPill marks the selected row. The pills of one group are the
// same element to MyGo: selecting another row slides the pill there. An
// absolute element draws above its siblings, so the pill is a see-through
// tint the row's text shows through, and lets the pointer pass.
func SelectionPill(c *ui.Context, group string, radius float32) {
	selectionCard(c, group, radius, 0, false)
}

// SelectionUnder is SelectionPill with a line of the accent under the
// item, which slides with the card to the next one picked.
func SelectionUnder(c *ui.Context, group string, radius float32) {
	selectionCard(c, group, radius, 0, true)
}

// SelectionEdge is SelectionPill with a bar of the accent at the list's
// edge, inset DIPs to the left of the row, sliding with it.
func SelectionEdge(c *ui.Context, group string, radius, inset float32) {
	selectionCard(c, group, radius, inset, false)
}

// selectionCard is the selected item's card: raised a little from the
// panel, the accent's light coming in from its left side. It is drawn
// over the item's content, so every layer of it is see-through.
func selectionCard(c *ui.Context, group string, radius, edge float32, under bool) {
	top, bottom := theme.Fade(ui.Hex("#FFFFFF"), 0.075), theme.Fade(ui.Hex("#FFFFFF"), 0.035)
	shadow := ui.RGBA(0, 0, 0, 0.28)
	if theme.Current().Light {
		top, bottom, shadow = theme.Fade(ui.Hex("#000000"), 0.015), theme.Fade(ui.Hex("#000000"), 0.035), ui.RGBA(0, 0, 0, 0.07)
	}
	ui.Box(c.Key("pill")).Absolute().Left(0).Top(0).Right(0).Bottom(0).Radius(radius).PassThrough().
		Gradient(top, bottom, 180).Border(1, theme.BorderStrong).
		InsetShadow(0, 1, 0, 0, theme.Highlight).InsetShadow(1, 0, 0, 0, theme.Fade(theme.Accent, 0.45)).
		Shadow(0, 4, 14, 0, shadow).
		Transition(ui.ElementTransition{Animation: &Spring, Match: pill{group}}).Children(func() {
		ui.Box(c.Key("glow")).Absolute().Left(0).Top(0).Bottom(0).Width(90).Radius(radius).PassThrough().
			LinearGradient(ui.LinearGradient{Angle: 90, From: theme.Fade(theme.Accent, 0.16), To: theme.Fade(theme.Accent, 0)})
		if under {
			ui.Box(c.Key("under")).Absolute().Left(12).Right(12).Bottom(0).Height(2).Radius(2, 2, 0, 0).PassThrough().
				Background(theme.Accent).Shadow(0, 0, 10, 0, theme.Fade(theme.Accent, 0.55))
		}
		if edge > 0 {
			ui.Box(c.Key("edge")).Absolute().Left(-edge).TopPercent(20).BottomPercent(20).Width(4).Radius(0, 3, 3, 0).PassThrough().
				Background(theme.Accent).Shadow(0, 0, 10, 0, theme.Fade(theme.Accent, 0.6))
		}
	})
}

// PanelMotion is a side panel's: it slides in from its edge and folds
// away, while what is beside it moves with LayoutMotion.
func PanelMotion(fromX float32) ui.ElementTransition {
	m := &ui.Motion{X: fromX, Opacity: 0, Collapse: true}
	return ui.ElementTransition{Animation: &Spring, Enter: m, Exit: m}
}

// QuickMotion moves and resizes a small control at once, near enough:
// for what the user clicked, which should answer without a glide.
var QuickMotion = ui.ElementTransition{Animation: &ui.Animation{Duration: 140 * time.Millisecond, Ease: Out}, Position: true, Size: true}

// quick eases a small control's colors as fast.
var quick = ui.Animation{Duration: 120 * time.Millisecond, Ease: Out}

// QuickTint eases e's color named key toward target at QuickMotion's pace.
func QuickTint(e ui.Element, key string, target ui.Color) ui.Color {
	return e.AnimateColor(key, target, quick)
}

// LayoutMotion moves and resizes an element in step with its neighbours.
var LayoutMotion = ui.ElementTransition{Animation: &Spring, Position: true, Size: true}

// PageMotion is a view's as it replaces another: it rises a few DIPs as
// it fades in.
var PageMotion = ui.ElementTransition{Duration: 340 * time.Millisecond, Ease: Out, Enter: &ui.Motion{Y: 10, Opacity: 0}}

// OverlayMotion is a pop-up's or dialog's: it grows from a little smaller
// as it fades in, and fades out a touch smaller.
var OverlayMotion = ui.ElementTransition{
	Animation: &Spring,
	Enter:     &ui.Motion{Scale: 0.95, Y: -4, Opacity: 0},
	Exit:      &ui.Motion{Scale: 0.97, Opacity: 0},
}

// RowMotion is a list row's as it appears: it slides in a little later
// than the one above it.
func RowMotion(i int) ui.ElementTransition {
	delay := time.Duration(min(i, 12)) * 18 * time.Millisecond
	return ui.ElementTransition{
		Animation: &ui.Animation{Duration: 320 * time.Millisecond, Ease: Out, Delay: delay},
		Enter:     &ui.Motion{Y: 6, Opacity: 0},
		Position:  true,
	}
}
