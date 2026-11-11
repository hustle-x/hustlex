package ui

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/core/agent"
	"github.com/hustlex/hustlex/proto"
	"github.com/hustlex/hustlex/ui/icon"
	"github.com/hustlex/hustlex/ui/textfmt"
	"github.com/hustlex/hustlex/ui/theme"
	"github.com/hustlex/hustlex/ui/widget"
)

// accountBar says, under a session of an agent whose accounts hustleX
// reads, which account it runs as and what is left of its first limit,
// with a way to go on as another of the agent's accounts.
func (a *App) accountBar(c *ui.Context, s proto.SessionInfo) {
	base := agent.Base(s.Agent)
	ref := proto.RefDefault
	if s.Account != "" {
		ref = s.Account
	}
	var acc proto.CreditAccount
	var others []proto.CreditAccount
	found := false
	for _, x := range a.credits.Accounts() {
		if x.Agent != base {
			continue
		}
		if x.Ref == ref {
			acc, found = x, true
		} else {
			others = append(others, x)
		}
	}
	if !found {
		return
	}
	who := acc.Account
	if who == "" {
		who = a.agentName(base)
	}
	ui.Row(c.Key("account-bar")).Height(32).Shrink(0).FillWidth().PaddingX(panelInset).Gap(10).AlignItems(ui.Center).
		BorderWidth(1, 0, 0, 0).BorderColor(theme.Border).Children(func() {
		widget.Avatar(c, icon.Agent(base), 18, icon.AgentTint(base, theme.Text))
		ui.Row(c).Gap(4).Shrink(1).MinWidth(0).Children(func() {
			ui.Text(c, "This session uses").FontSize(11.5).TextColor(theme.Muted).SingleLine().Shrink(0)
			ui.Text(c, who).FontSize(11.5).TextColor(theme.Text).SingleLine().Shrink(1)
		})
		if acc.Plan != "" {
			ui.Text(c, acc.Plan).FontSize(11).TextColor(theme.TextSoft).SingleLine().Shrink(0).
				Padding(2, 7).Radius(6).Border(1, theme.BorderStrong)
		}
		if len(acc.Windows) > 0 {
			w := acc.Windows[0]
			text := fmt.Sprintf("%s %.0f%%", w.Short(), w.Used)
			if now := time.Now(); w.ResetsAt.After(now) {
				text += " · resets in " + textfmt.Until(w.ResetsAt, now)
			}
			col := theme.Muted
			if w.Used >= 60 {
				col = theme.Warning
			}
			ui.Text(c, text).FontSize(11.5).TextColor(col).SingleLine().Shrink(1)
		}
		ui.Spacer(c)
		if len(others) == 0 || s.Parent != "" || s.Project == "" {
			return
		}
		b := widget.GhostButton(c.Key("switch-account"), "Switch account").Height(22).PaddingX(9)
		key := "switch-account:" + s.ID
		a.menu.OpenBelow(b, key, 22)
		a.menu.Show(c, b, key, func(done func()) {
			for _, o := range others {
				name := o.Account
				if n := a.accountChipName(o.Agent, o.Ref); n != "" {
					name = n
				}
				if name == "" {
					name = o.Ref
				}
				if widget.MenuItemTip(c.Key(o.Ref), icon.Agent(base), name, "Go on with this conversation as it", false) {
					a.moveSession(base, o.Ref)
					done()
				}
			}
		})
	})
}
