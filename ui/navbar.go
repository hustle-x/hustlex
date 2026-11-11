package ui

import (
	"strconv"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/core/hosts"
	"github.com/hustlex/hustlex/ui/icon"
	"github.com/hustlex/hustlex/ui/theme"
	"github.com/hustlex/hustlex/ui/widget"
)

// navBarHeight is the height of the row of pages under the title bar.
const navBarHeight = 44

// panelInset is how far a panel's content sits from its edge, so the
// logo, every panel's title and the status bar start on one line.
const panelInset = 12

// navBar is the row of pages under the title bar: a labeled button per
// page, the open one lit. The buttons that fold the Sessions panel and the
// tools panel sit at its ends, on the side of each panel.
func (a *App) navBar(c *ui.Context) {
	if a.maximized != "" {
		return
	}
	left := a.state.sessionsLeft()
	remote := a.client != nil && a.host.ID != hosts.LocalID
	ui.Row(c.Key("nav-pad")).FillWidth().Shrink(0).Padding(0, theme.BentoGap, theme.BentoGap, theme.BentoGap).Children(func() {
		widget.Bento(ui.Row(c.Key("nav")).FillWidth().Height(navBarHeight).Padding(0, panelInset).Gap(2).AlignItems(ui.Center), theme.Panel).Children(func() {
			ui.Box(c).Size(28, 28).Shrink(0).Radius(theme.RadiusControl).Background(theme.Accent).TextColor(theme.OnAccent).Center().
				Margin(0, 6, 0, 0).Children(func() { ui.Icon(c, icon.Logo).FontSize(20) })
			// Each panel's toggle on its panel's side, the logo keeping the
			// left edge.
			compact := a.compact()
			if left && !compact {
				a.sessionsToggle(c, icon.Sidebar)
				ui.Box(c).Size(1, 18).Shrink(0).Background(theme.Border).Margin(0, 4)
			}
			ui.Row(c).Grow(1).MinWidth(0).Gap(2).Clip().Children(func() {
				if a.navItem(c, icon.Terminal, "Sessions", "sessions", a.page == "", true) {
					a.page = ""
				}
				if a.navItem(c, icon.Inbox, "Inbox", pageInbox, a.page == pageInbox, a.client != nil) {
					a.togglePage(pageInbox)
				}
				if a.navItem(c, icon.Notebook, "Notes", pageNotes, a.page == pageNotes, a.client != nil) {
					a.togglePage(pageNotes)
				}
				if compact {
					return
				}
				if a.navItem(c, icon.Canvas, "Canvas", pageCanvas, a.page == pageCanvas, a.client != nil) {
					a.togglePage(pageCanvas)
				}
				if a.navItem(c, icon.Key, "Credentials", pageVault, a.page == pageVault, true) {
					a.openVault()
				}
				if a.navItem(c, icon.Credits, "Credits", pageCredits, a.page == pageCredits, a.client != nil) {
					a.togglePage(pageCredits)
				}
				if a.navItem(c, icon.Usage, "Usage", pageUsage, a.page == pageUsage, a.client != nil) {
					a.togglePage(pageUsage)
				}
				if a.navItem(c, icon.Sync, "Sync", pageSync, a.page == pageSync, remote) {
					a.openSync()
				}
			})
			if compact {
				ui.Spacer(c)
			}
			a.searchButton(c)
			a.hostMenu(c, false)
			mode, label := icon.Sun, "Switch to the light mode"
			if theme.Current().Light {
				mode, label = icon.Moon, "Switch to the dark mode"
			}
			if widget.IconButton(c.Key("nav-mode"), mode, label, false).Clicked() {
				a.toggleMode()
			}
			if widget.IconButton(c.Key("nav-appearance"), icon.Palette, "Appearance", a.page == pageAppearance).Clicked() {
				a.accentInput = a.state.Accent
				a.togglePage(pageAppearance)
			}
			if widget.IconButton(c.Key("nav-phone"), icon.Phone, "Pair a phone", a.phoneOf().open).Clicked() {
				a.pairPhone()
			}
			if widget.IconButton(c.Key("nav-settings"), icon.Settings, "Settings", a.page == pageSettings).Clicked() && a.client != nil {
				a.togglePage(pageSettings)
			}
			ui.Box(c).Size(1, 18).Shrink(0).Background(theme.Border).Margin(0, 4)
			if widget.IconButton(c.Key("nav-tools"), icon.PanelRight, "Show or hide the tools ("+modKey+"⇧D)", a.panelShown()).Clicked() {
				a.togglePanel()
			}
			if !left {
				a.sessionsToggle(c, icon.Sidebar)
			}
		})
	})
}

// sessionsToggle is the button that folds the Sessions panel away or
// brings it back.
func (a *App) sessionsToggle(c *ui.Context, svg *ui.SVG) {
	if widget.IconButton(c.Key("nav-sessions"), svg, "Show or hide the Sessions panel ("+modKey+"B)", !a.state.HideSessions).Clicked() {
		a.toggleSessionsPanel()
	}
}

// navItem is one page of the nav bar: its icon and label, lit while the
// page is open. It reports a click on a page that is available.
func (a *App) navItem(c *ui.Context, svg *ui.SVG, label, id string, on, enabled bool) bool {
	b := ui.ButtonBase(c.Key("nav:" + id)).Height(30).Shrink(0).PaddingX(11).Gap(7).Radius(theme.RadiusControl).
		Label(label).Cursor(ui.CursorPointer)
	bg, fg := theme.Fade(theme.Hover, 0), theme.TextSoft
	switch {
	case on:
		fg = theme.TextStrong
	case b.Hovered():
		bg, fg = theme.Hover, theme.Text
	}
	b.Background(widget.Tint(b, "bg", bg)).TextColor(widget.Tint(b, "fg", fg))
	if !enabled {
		b.Opacity(0.4)
	}
	widget.Pressable(b).Children(func() {
		if on {
			widget.SelectionUnder(c, "nav-page", theme.RadiusControl)
		}
		ui.Icon(c, svg).FontSize(15)
		ui.Text(c, label).FontSize(12.5).SingleLine()
		if id == pageInbox {
			if n := len(a.waiting()); n > 0 {
				ui.Box(c).MinWidth(16).Height(16).PaddingX(4).Radius(8).Background(theme.Warning).Center().Shrink(0).Children(func() {
					ui.Text(c, strconv.Itoa(n)).FontSize(10).TextColor(theme.Mix(theme.Warning, ui.Hex("#000000"), 0.9)).SingleLine()
				})
			}
		}
	})
	return enabled && b.Clicked()
}

// togglePage shows a page, or the sessions again if it is showing.
func (a *App) togglePage(id string) {
	if a.page == id {
		a.page = ""
	} else {
		a.page = id
	}
}

// searchButton opens the command palette: every page, session and
// command, a click away for whoever doesn't know its shortcut.
func (a *App) searchButton(c *ui.Context) {
	b := ui.ButtonBase(c.Key("nav-search")).Height(30).Shrink(0).PaddingX(10).Gap(8).Radius(theme.RadiusControl).Margin(0, 6, 0, 8).
		Label("Search & commands").Tooltip("Search & commands (" + modKey + "K)").Cursor(ui.CursorPointer).Background(theme.Sunken)
	border, fg := theme.BorderStrong, theme.Muted
	if b.Hovered() {
		border, fg = theme.AccentLine, theme.TextSoft
	}
	widget.Pressable(b.Border(1, widget.Tint(b, "border", border)).TextColor(widget.Tint(b, "fg", fg))).Children(func() {
		ui.Icon(c, icon.Search).FontSize(13)
		ui.Text(c, "Search & commands").FontSize(12).SingleLine()
		ui.Text(c, modKey+"K").FontSize(10.5).Padding(1, 5).Radius(5).Border(1, theme.BorderStrong).SingleLine()
	})
	if b.Clicked() {
		a.openPalette()
	}
}
