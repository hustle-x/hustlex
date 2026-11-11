package ui

import (
	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/proto"
	"github.com/hustlex/hustlex/ui/icon"
	"github.com/hustlex/hustlex/ui/theme"
	"github.com/hustlex/hustlex/ui/widget"
)

// The window's two layouts. Compact shows what a session needs: Sessions
// and Notes, a session as a chat or a terminal; the other pages, views
// and tools stay a ⌘K away. Expert shows everything, as hustleX always
// did, and is what a user from before the layouts has.
const (
	layoutCompact = "compact"
	layoutExpert  = "expert"
)

// compact reports whether the window is in the Compact layout.
func (a *App) compact() bool { return a.state.Layout == layoutCompact }

// setWindowLayout switches the layout and keeps it.
func (a *App) setWindowLayout(layout string) {
	a.state.Layout = layout
	if layout == layoutCompact {
		switch a.page {
		case pageCanvas, pageVault, pageCredits, pageUsage, pageSync:
			a.page = ""
		}
	}
	a.saveState()
}

// compactChat makes session s show as a chat in Compact, unless the user
// picked its terminal: a chat reads more easily to someone new.
func (a *App) compactChat(s proto.SessionInfo) {
	if !a.compact() || !canChat(s) || a.state.Chats[s.ID] || a.state.TerminalViews[s.ID] {
		return
	}
	if a.state.Chats == nil {
		a.state.Chats = map[string]bool{}
	}
	a.state.Chats[s.ID] = true
}

// layoutPicker asks a new user, before anything else, how much the window
// should show.
func (a *App) layoutPicker(c *ui.Context) {
	if a.state.Layout != "" || !a.welcomeShown() {
		return
	}
	if a.layoutPick == "" {
		a.layoutPick = layoutCompact
	}
	open := true
	widget.Sheet(c, &open, 640, func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
			ui.Text(c, "Choose your layout").FontSize(18)
			ui.Text(c, "Pick how much hustleX shows. You can switch anytime in Settings.").FontSize(12).TextColor(theme.Muted)
		})
	}, func() {
		a.layoutCards(c, a.layoutPick, func(l string) { a.layoutPick = l })
	}, func() {
		ui.Text(c, "Not sure? Start with Compact; everything is still a "+modKey+"K away.").FontSize(11.5).TextColor(theme.Muted).Shrink(1)
		ui.Spacer(c)
		label := "Continue with Compact"
		if a.layoutPick == layoutExpert {
			label = "Continue with Expert"
		}
		if widget.PrimaryButton(c, nil, label, "").Clicked() {
			a.setWindowLayout(a.layoutPick)
		}
	})
	if !open {
		a.setWindowLayout(layoutExpert) // closed without picking: everything, as before
	}
}

// layoutCards are the two layouts side by side, current one picked.
func (a *App) layoutCards(c *ui.Context, current string, pick func(string)) {
	ui.Row(c).FillWidth().Gap(12).Children(func() {
		for _, l := range []string{layoutCompact, layoutExpert} {
			on := current == l
			b := ui.ButtonBase(c.Key("layout:" + l)).Grow(1).Basis(0).Padding(12).Gap(10).Radius(theme.RadiusCard).
				Background(theme.Raised).Cursor(ui.CursorPointer).Label(map[string]string{layoutCompact: "Compact", layoutExpert: "Expert"}[l])
			border := theme.BorderStrong
			switch {
			case on:
				border = theme.Accent
			case b.Hovered():
				border = theme.AccentLine
			}
			b.Border(1, widget.Tint(b, "border", border))
			widget.Pressable(b).Children(func() {
				ui.Column(c).FillWidth().Gap(10).Children(func() {
					layoutSketch(c, l == layoutExpert)
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						if l == layoutCompact {
							ui.Text(c, "Compact").FontSize(13.5)
							ui.Text(c, "Recommended").FontSize(10.5).Padding(2, 7).Radius(6).Background(theme.Selected).TextColor(theme.TextSoft)
						} else {
							ui.Text(c, "Expert").FontSize(13.5)
						}
						ui.Spacer(c)
						if on {
							ui.Icon(c, icon.Check).FontSize(14).TextColor(theme.Accent)
						}
					})
					about := "Sessions and Notes, terminal or chat. Tools show up when a session needs them."
					if l == layoutExpert {
						about = "Every tool in view: canvas, credentials, usage, sync, browser, traffic and the side panel."
					}
					ui.Text(c, about).FontSize(12).TextColor(theme.Muted)
				})
			})
			if b.Clicked() {
				pick(l)
			}
		}
	})
}

// layoutSketch is a small drawing of a layout: the nav bar, the Sessions
// panel, the session, and in Expert the tools.
func layoutSketch(c *ui.Context, expert bool) {
	ui.Column(c).FillWidth().Height(120).Padding(7).Gap(5).Radius(9).Background(theme.Sunken).Border(1, theme.Border).Children(func() {
		ui.Row(c).FillWidth().Height(12).Gap(3).Padding(3, 4).Radius(4).Background(theme.Hover).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(6, 5).Radius(2).Background(theme.Accent)
			n := 2
			if expert {
				n = 7
			}
			for range n {
				ui.Box(c).Size(16, 5).Radius(2).Background(theme.BorderStrong)
			}
		})
		ui.Row(c).FillWidth().Grow(1).Gap(5).Children(func() {
			ui.Box(c).Width(46).FillHeight().Radius(5).Background(theme.Panel).Border(1, theme.Border)
			ui.Column(c).Grow(1).FillHeight().Padding(6).Gap(4).Radius(5).Background(theme.Panel).Border(1, theme.Border).Children(func() {
				for _, w := range []float32{70, 45, 60} {
					ui.Box(c).WidthPercent(w).Height(4).Radius(2).Background(theme.BorderStrong)
				}
			})
			if expert {
				ui.Box(c).Width(38).FillHeight().Radius(5).Background(theme.Panel).Border(1, theme.Border)
			}
		})
	})
}
