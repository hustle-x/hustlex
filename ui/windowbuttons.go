package ui

import (
	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/ui/icon"
	"github.com/hustlex/hustlex/ui/theme"
)

// The window's close, minimize and maximize buttons, drawn by the app: on
// macOS three dots on the left, their signs showing while the pointer is
// over them, grey while the window isn't the focused one; elsewhere −, □
// and × on the right (systemButtons).

// windowDots are the buttons' colors, as people know them.
var windowDots = [3]ui.Color{ui.Hex("#FF5F57"), ui.Hex("#FEBC2E"), ui.Hex("#28C840")}

// windowControl is what the buttons do to the window: a *mygo.Window,
// or a test's.
type windowControl interface {
	Close()
	Minimize()
	ToggleMaximize()
}

// windowCtl is the window the buttons act on, nil when there is none.
func (a *App) windowCtl() windowControl {
	if a.winCtl != nil {
		return a.winCtl
	}
	if a.win != nil {
		return a.win
	}
	return nil
}

// windowButtons draws the macOS dots in the title bar.
func (a *App) windowButtons(c *ui.Context) {
	win := a.windowCtl()
	if win == nil {
		return
	}
	group := ui.Row(c.Key("window-buttons")).Gap(8).Shrink(0)
	over := group.Hovered()
	group.Children(func() {
		for i, label := range []string{"Close", "Minimize", "Zoom"} {
			col := windowDots[i]
			if a.blurred && !over {
				col = theme.Idle
			}
			b := ui.ButtonBase(c.Key(label)).Size(12, 12).Radius(6).Shrink(0).Background(col).Label(label).Center()
			b.Children(func() {
				if !over {
					return
				}
				sign := ui.RGBA(0, 0, 0, 0.55)
				switch i {
				case 0:
					ui.Icon(c, icon.Close).FontSize(8).TextColor(sign)
				case 1:
					ui.Box(c).Size(6, 1.2).Background(sign)
				default:
					ui.Icon(c, icon.Plus).FontSize(8).TextColor(sign)
				}
			})
			if b.Clicked() {
				switch i {
				case 0:
					win.Close()
				case 1:
					win.Minimize()
				default:
					win.ToggleMaximize()
				}
			}
		}
	})
}

// systemButtons draws −, □ and × as Windows does: as tall as the title bar,
// the × turning red under the pointer.
func (a *App) systemButtons(c *ui.Context, height float32) {
	win := a.windowCtl()
	if win == nil {
		return
	}
	ui.Row(c.Key("window-buttons")).Shrink(0).Children(func() {
		for i, label := range []string{"Minimize", "Zoom", "Close"} {
			b := ui.ButtonBase(c.Key(label)).Size(46, height).Shrink(0).Label(label).Center()
			fg := theme.TextSoft
			if a.blurred {
				fg = theme.Muted
			}
			if b.Hovered() {
				b.Background(theme.Hover)
				fg = theme.Text
				if i == 2 {
					b.Background(ui.Hex("#C42B1C"))
					fg = ui.Hex("#FFFFFF")
				}
			}
			b.Children(func() {
				switch i {
				case 0:
					ui.Icon(c, icon.Minus).FontSize(13).TextColor(fg)
				case 1:
					ui.Icon(c, icon.Square).FontSize(11).TextColor(fg)
				default:
					ui.Icon(c, icon.Close).FontSize(13).TextColor(fg)
				}
			})
			if b.Clicked() {
				switch i {
				case 0:
					win.Minimize()
				case 1:
					win.ToggleMaximize()
				default:
					win.Close()
				}
			}
		}
	})
}
