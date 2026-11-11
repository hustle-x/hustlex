package ui

import (
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/core/agent"
	"github.com/hustlex/hustlex/proto"
	"github.com/hustlex/hustlex/ui/icon"
	"github.com/hustlex/hustlex/ui/textfmt"
	"github.com/hustlex/hustlex/ui/theme"
	"github.com/hustlex/hustlex/ui/widget"
)

// A new session can start when its account has room again: the New
// session form's Start picks the reset of the account's fullest limit,
// and the daemon holds the session until then (SessionCreateParams.StartAt).

// startMargin is how long after a limit's reset a queued session starts,
// so the account's reading has turned over.
const startMargin = time.Minute

// startsLater reports whether s is queued until a time.
func startsLater(s proto.SessionInfo) bool {
	return s.Queued && !s.StartAt.IsZero() && s.Parent == ""
}

// queuedLater lists, above the projects, the sessions queued until a
// limit resets.
func (a *App) queuedLater(c *ui.Context) {
	if strings.TrimSpace(a.filter) != "" {
		return
	}
	var later []proto.SessionInfo
	for _, p := range a.projects {
		for _, s := range a.projectSessions(p.Path) {
			if startsLater(s) {
				later = append(later, s)
			}
		}
	}
	if len(later) == 0 {
		return
	}
	ui.Row(c.Key("queued-later")).Height(28).FillWidth().Gap(6).PaddingX(4).Margin(6, 0, 0, 0).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icon.Clock).FontSize(11).TextColor(theme.Muted)
		widget.SectionLabel(c, "QUEUED", theme.Muted)
		ui.Spacer(c)
		ui.Text(c, strconv.Itoa(len(later))).FontSize(10.5).TextColor(theme.Muted)
	})
	for i, s := range later {
		a.sessionRow(c.Key("later:"+s.ID), i, s)
	}
}

// startsDetail says when queued session s starts, and after what: "starts
// 15:40 · after alex's 5-hour reset".
func (a *App) startsDetail(s proto.SessionInfo) string {
	text := "starts " + clockOrDay(s.StartAt, time.Now())
	ref := proto.RefDefault
	if s.Account != "" {
		ref = s.Account
	}
	for _, acc := range a.credits.Accounts() {
		if acc.Agent != agent.Base(s.Agent) || acc.Ref != ref {
			continue
		}
		for _, w := range acc.Windows {
			if d := s.StartAt.Sub(w.ResetsAt); d >= 0 && d <= 3*startMargin {
				return text + " · after " + a.accountShort(acc) + "'s " + strings.TrimSuffix(w.LimitName(), " limit") + " reset"
			}
		}
	}
	return text
}

// clockOrDay is t as a time today, else with its day.
func clockOrDay(t, now time.Time) string {
	t, now = t.Local(), now.Local()
	if y, m, d := t.Date(); y == now.Year() && m == now.Month() && d == now.Day() {
		return t.Format("15:04")
	}
	return t.Format("Mon 15:04")
}

// accountShort is what short texts call an account: the user's name for
// its chip, else its email before the @.
func (a *App) accountShort(acc proto.CreditAccount) string {
	if n := a.accountChipName(acc.Agent, acc.Ref); n != "" {
		return n
	}
	if who, _, ok := strings.Cut(acc.Account, "@"); ok && who != "" {
		return who
	}
	if acc.Account != "" {
		return acc.Account
	}
	return a.agentName(acc.Agent)
}

// resetToWait is the reset a session of agentID as account (a session's
// Account) would wait for: that of its fullest limit with a reset ahead;
// ok false when the account has no such reading.
func (a *App) resetToWait(agentID, account string) (acc proto.CreditAccount, w proto.CreditWindow, ok bool) {
	ref := proto.RefDefault
	if account != "" {
		ref = account
	}
	now := time.Now()
	for _, x := range a.credits.Accounts() {
		if x.Agent != agent.Base(agentID) || x.Ref != ref || x.Error != "" {
			continue
		}
		for _, y := range x.Windows {
			if y.ResetsAt.After(now) && (!ok || y.Used > w.Used || y.Used == w.Used && y.ResetsAt.Before(w.ResetsAt)) {
				acc, w, ok = x, y, true
			}
		}
	}
	return acc, w, ok
}

// startField is the New session form's Start: now, or when the account's
// fullest limit resets, for an agent whose account has a reading.
func (a *App) startField(c *ui.Context) {
	f := &a.newSession
	if _, isAgent := agent.Find(f.agent); !isAgent || !a.daemonInfo.StartLater {
		return
	}
	acc, w, ok := a.resetToWait(f.agent, f.account)
	if !ok {
		f.later = false
		return
	}
	widget.Field(c, "Start", func() {
		later := "When a limit resets · " + a.accountShort(acc) + " in " + textfmt.Until(w.ResetsAt, time.Now())
		cur := 0
		if f.later {
			cur = 1
		}
		if i := widget.Segmented(c.Key("start"), []string{"Now", later}, cur, nil); i >= 0 {
			f.later = i == 1
		}
	})
}
