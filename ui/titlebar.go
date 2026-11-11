package ui

import (
	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/ui/theme"
	"github.com/hustlex/hustlex/ui/widget"
)

// titleBarHeight is the height of the bar the window controls sit in.
const titleBarHeight = 40

// titleBar draws the window's title bar: the credits and the memory on the
// right. The host isn't there: the status bar says it, and the nav bar's
// + switches and manages hosts. It drags the window, keeping clear of the
// window controls.
func (a *App) titleBar(c *ui.Context) {
	a.titleBarWith(c, func() {
		a.updateChip(c)
		if !a.compact() {
			a.memoryChip(c)
		}
		if ag, _ := a.welcomeAgent(); !a.welcomeShown() || a.signedIn(ag.ID) {
			a.creditChip(c) // the welcome's steps say what to install and sign in to
		}
	})
}

// titleBarWith is the title bar with chips on its right.
func (a *App) titleBarWith(c *ui.Context, chips func()) {
	bar := c.TitleBar()
	height := max(bar.Height, titleBarHeight)
	right := bar.Right + 10
	if !macWindowButtons {
		right = bar.Right // the buttons sit flush with the corner
	}
	ui.Row(c).Height(height).Shrink(0).FillWidth().
		Padding(0, right, 0, bar.Left+14).Gap(8).DragWindow().Children(func() {
		// The sides keep what they hold, as wide as they were last
		// frame, the search between them giving way: centered while there
		// is room.
		ui.Row(c).Grow(1).Basis(0).MinWidth(a.titleLeftW).Children(func() {
			inner := ui.Row(c).Shrink(0).Children(func() {
				if macWindowButtons {
					a.windowButtons(c)
				}
			})
			a.titleLeftW = inner.Bounds().W
		})
		ui.Row(c).Grow(1).Basis(0).MinWidth(a.titleRightW).Justify(ui.End).Children(func() {
			inner := ui.Row(c).Shrink(0).Gap(6).Children(func() {
				chips()
				if !macWindowButtons {
					ui.Box(c).Width(4).Shrink(0)
					a.systemButtons(c, height)
				}
			})
			if w := inner.Bounds().W; w != a.titleRightW {
				a.titleRightW = w
				c.After(0) // lay out again with it
			}
		})
	})
}

// titleChip is a small button of the title bar.
func titleChip(c *ui.Context, key, label string) ui.Element {
	b := ui.ButtonBase(c.Key(key)).Height(26).PaddingX(9).Gap(6).Radius(theme.RadiusControl).Shrink(0).
		Label(label).Tooltip(label).Cursor(ui.CursorPointer).TextColor(theme.TextSoft)
	widget.Glass(b, theme.Sunken)
	border := theme.BorderStrong
	if b.Hovered() {
		border = theme.AccentLine
	}
	return widget.Pressable(b.Border(1, widget.Tint(b, "border", border)))
}
