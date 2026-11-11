package ui

import (
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/core/agent"
	"github.com/hustlex/hustlex/proto"
	"github.com/hustlex/hustlex/ui/icon"
	"github.com/hustlex/hustlex/ui/theme"
	"github.com/hustlex/hustlex/ui/widget"
)

// The daemon moves a session whose account ran out of a limit to another
// account with room (daemon/accountmove.go); the app says so, once as a
// notification and above the session's account bar for a day.

// movedShown is how long a session says it moved.
const movedShown = 24 * time.Hour

// accountName is what the app calls session account ref (a session's
// Account) of agentID: its email, else the user's name for it.
func (a *App) accountName(agentID, account string) string {
	ref := proto.RefDefault
	if account != "" {
		ref = account
	}
	for _, acc := range a.credits.Accounts() {
		if acc.Agent == agent.Base(agentID) && acc.Ref == ref && acc.Account != "" {
			return acc.Account
		}
	}
	if n := a.accountChipName(agent.Base(agentID), ref); n != "" {
		return n
	}
	if account == "" {
		return "its own account"
	}
	return account
}

// noteMove tells the user session info just moved, when it did.
func (a *App) noteMove(old, info proto.SessionInfo) {
	if info.Moved == nil || movedAt(old) == movedAt(info) || time.Since(info.Moved.At) > time.Minute {
		return
	}
	m := info.Moved
	a.notify("Moved to " + a.accountName(info.Agent, m.To) + ": " + a.accountName(info.Agent, m.From) + " ran out of its " + m.Limit)
}

// movedNotice says, above session s's account bar, that it moved to
// another account at its limit, until dismissed or a day passed.
func (a *App) movedNotice(c *ui.Context, s proto.SessionInfo) {
	m := s.Moved
	if m == nil || time.Since(m.At) > movedShown || a.movedSeen[s.ID] == movedAt(s) {
		return
	}
	ui.Row(c.Key("moved")).Shrink(0).FillWidth().Padding(7, panelInset).Gap(9).AlignItems(ui.Center).
		BorderWidth(1, 0, 0, 0).BorderColor(theme.Border).Background(theme.Sunken).Children(func() {
		ui.Box(c).Size(20, 20).Radius(6).Center().Background(theme.Fade(theme.Text, 0.08)).Shrink(0).Children(func() {
			ui.Icon(c, icon.Refresh).FontSize(11).TextColor(theme.TextSoft)
		})
		ui.Row(c).Grow(1).MinWidth(0).Gap(4).Wrap().Children(func() {
			ui.Text(c, a.accountName(s.Agent, m.From)+" hit its").FontSize(12).TextColor(theme.TextSoft)
			ui.Text(c, m.Limit+".").FontSize(12).TextColor(theme.Text)
			ui.Text(c, "Continued on").FontSize(12).TextColor(theme.TextSoft)
			ui.Text(c, a.accountName(s.Agent, m.To)+",").FontSize(12).TextColor(theme.Text)
			ui.Text(c, "same conversation.").FontSize(12).TextColor(theme.TextSoft)
		})
		if widget.IconButton(c, icon.Close, "Dismiss", false).Clicked() {
			if a.movedSeen == nil {
				a.movedSeen = map[string]int64{}
			}
			a.movedSeen[s.ID] = movedAt(s)
		}
	})
}
