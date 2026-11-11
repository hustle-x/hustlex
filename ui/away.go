package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/proto"
	"github.com/hustlex/hustlex/ui/icon"
	"github.com/hustlex/hustlex/ui/textfmt"
	"github.com/hustlex/hustlex/ui/theme"
	"github.com/hustlex/hustlex/ui/widget"
)

// Back at the computer after a while, the user sees what moved on while
// they were away: sessions done, asking something, moved or started, and
// how the limits went. Away is the computer left without keyboard or
// mouse (the system's idle time), whatever the app showed.

const (
	awayAfter = 10 * time.Minute // idle this long, the user is away
	awayBack  = time.Minute      // idle less, they are here
	awayEvery = 15 * time.Second // how often the idle time is read
)

// idleTime is how long the computer had no keyboard or mouse; 0 where the
// system doesn't say. Tests replace it.
var idleTime = func() time.Duration { return mygo.Power.IdleTime() }

// awayState belongs to the main thread.
type awayState struct {
	snap  awaySnap  // things as they were while the user was last here
	since time.Time // when they left, zero while here
	open  bool      // the summary shows
	sum   awaySummary
}

// awaySnap is the sessions and limits at a moment.
type awaySnap struct {
	sessions map[string]proto.SessionInfo
	limits   map[string]proto.CreditWindow // agent+ref → its first limit
}

// awaySummary is what moved on while the user was away.
type awaySummary struct {
	away  time.Duration
	moved int // sessions with news
	rows  []awayRow
}

// awayRow is one piece of news.
type awayRow struct {
	kind          awayKind
	title, detail string
	session       string // the session it is about, "" for the limits
}

type awayKind int

const (
	awayDone awayKind = iota
	awayStarted
	awayMovedAccount
	awayNeeds
	awayLimits
)

// watchAway reads the idle time while the app runs.
func (a *App) watchAway() {
	for range time.Tick(awayEvery) {
		idle := idleTime()
		a.onMain(func() { a.awayTick(idle, time.Now()) })
	}
}

// awayTick notes whether the user is here or away, idle that long at
// now, and shows the summary when they come back with news.
func (a *App) awayTick(idle time.Duration, now time.Time) {
	st := &a.away
	switch {
	case idle < awayBack:
		if !st.since.IsZero() && now.Sub(st.since) >= awayAfter && a.client != nil {
			if sum := a.awaySummary(st.snap, st.since, now); sum.moved > 0 {
				st.sum, st.open = sum, true
			}
		}
		st.since = time.Time{}
		st.snap = a.awaySnapshot()
	case idle >= awayAfter && st.since.IsZero():
		st.since = now.Add(-idle)
	}
}

// awaySnapshot is the sessions and limits now.
func (a *App) awaySnapshot() awaySnap {
	snap := awaySnap{sessions: map[string]proto.SessionInfo{}, limits: map[string]proto.CreditWindow{}}
	for _, s := range a.sessions {
		snap.sessions[s.ID] = s
	}
	for _, acc := range a.credits.Accounts() {
		if len(acc.Windows) > 0 && acc.Error == "" {
			snap.limits[acc.Agent+" "+acc.Ref] = acc.Windows[0]
		}
	}
	return snap
}

// ownSession reports whether s is a session the user started an agent
// in: not a tab, a terminal, or a page's agent.
func ownSession(s proto.SessionInfo) bool {
	return s.Parent == "" && s.NotesFor == "" && s.CanvasFor == "" && s.Agent != "" && s.Agent != "shell" && s.Agent != "custom"
}

// awaySummary compares the sessions and limits now with snap, taken
// before the user left at since.
func (a *App) awaySummary(snap awaySnap, since, now time.Time) awaySummary {
	sum := awaySummary{away: now.Sub(since)}
	var done, started, moved, needs []awayRow
	for _, s := range a.sessions {
		if !ownSession(s) {
			continue
		}
		old, known := snap.sessions[s.ID]
		name := a.sessionName(s)
		switch {
		case waitsForYou(s) && !(known && waitsForYou(old)):
			detail := "waits for your answer"
			if m, ok := a.inbox.menus[s.ID]; ok && m.ok && m.menu.Question != "" {
				detail = a.agentName(s.Agent) + ": " + m.menu.Question
			}
			needs = append(needs, awayRow{awayNeeds, name + " needs you", detail, s.ID})
		case known && (isWorking(old) || old.Queued) && !isWorking(s) && !s.Queued && !s.Hibernated && !(s.Exited && s.ExitCode != 0):
			detail := "ready to review"
			if s.Note != "" {
				detail = textfmt.FirstLine(s.Note, 80)
			}
			done = append(done, awayRow{awayDone, name + " is done", detail, s.ID})
		case known && startsLater(old) && !s.Queued:
			started = append(started, awayRow{awayStarted, name + " started", "its account had room again", s.ID})
		case s.Moved != nil && s.Moved.At.After(since):
			moved = append(moved, awayRow{awayMovedAccount, name + " moved to " + a.accountName(s.Agent, s.Moved.To),
				a.accountName(s.Agent, s.Moved.From) + " ran out of its " + s.Moved.Limit, s.ID})
		}
	}
	for _, rows := range [][]awayRow{done, started, moved, needs} {
		sum.rows = append(sum.rows, rows...)
		sum.moved += len(rows)
	}
	var limits []string
	for _, acc := range a.credits.Accounts() {
		old, ok := snap.limits[acc.Agent+" "+acc.Ref]
		if !ok || len(acc.Windows) == 0 || acc.Error != "" || acc.Windows[0].Label != old.Label || acc.Windows[0].Used < old.Used+1 {
			continue
		}
		w := acc.Windows[0]
		text := fmt.Sprintf("%s %s %.0f%% → %.0f%%", a.accountShort(acc), strings.TrimSuffix(w.LimitName(), " limit"), old.Used, w.Used)
		if w.ResetsAt.After(now) {
			text += " · resets in " + textfmt.Until(w.ResetsAt, now)
		}
		limits = append(limits, text)
	}
	if len(limits) > 0 {
		sum.rows = append(sum.rows, awayRow{kind: awayLimits, title: "Limits", detail: strings.Join(limits, "; ")})
	}
	return sum
}

// awayLength says how long the user was away: "47 minutes", "2 hours 5
// minutes".
func awayLength(d time.Duration) string {
	m := int(d.Round(time.Minute) / time.Minute)
	if m < 60 {
		return count(m, "minute", "minutes")
	}
	h, m := m/60, m%60
	out := count(h, "hour", "hours")
	if m > 0 {
		out += " " + count(m, "minute", "minutes")
	}
	return out
}

// count is n and its noun: "1 session", "3 sessions".
func count(n int, one, many string) string {
	return fmt.Sprintf("%d %s", n, textfmt.Plural(n, one, many))
}

// awayDialog is the summary, while it shows.
func (a *App) awayDialog(c *ui.Context) {
	st := &a.away
	if !st.open {
		return
	}
	widget.Dialog(c, &st.open, 460, 0, func() {
		ui.Text(c, "While you were away").FontSize(15)
		ui.Text(c, awayLength(st.sum.away)+" · "+count(st.sum.moved, "session", "sessions")+" moved on").
			FontSize(12.5).TextColor(theme.Muted).Margin(2, 0, 12, 0)
		for i, r := range st.sum.rows {
			a.awayRowView(c.Key(i), r)
		}
		ui.Row(c).FillWidth().Margin(12, 0, 0, 0).Children(func() {
			ui.Spacer(c)
			if widget.PrimaryButton(c, nil, "Got it", "").Clicked() {
				st.open = false
			}
		})
	})
}

// awayRowView is one piece of news, with what to do about it.
func (a *App) awayRowView(c *ui.Context, r awayRow) {
	ui.Row(c).FillWidth().Padding(8, 0).Gap(10).AlignItems(ui.Center).BorderWidth(1, 0, 0, 0).BorderColor(theme.Border).Children(func() {
		tile := ui.Box(c).Size(22, 22).Radius(7).Center().Shrink(0)
		switch r.kind {
		case awayDone, awayStarted:
			tile.Background(theme.Fade(theme.Done, 0.14)).Children(func() { ui.Icon(c, icon.Check).FontSize(12).TextColor(theme.Done) })
		case awayNeeds:
			tile.Background(theme.Fade(theme.Warning, 0.14)).Children(func() { ui.Icon(c, icon.Alert).FontSize(12).TextColor(theme.Warning) })
		case awayMovedAccount:
			tile.Background(theme.Fade(theme.Text, 0.08)).Children(func() { ui.Icon(c, icon.Refresh).FontSize(12).TextColor(theme.TextSoft) })
		case awayLimits:
			tile.Background(theme.Fade(theme.Text, 0.06)).Children(func() { ui.Icon(c, icon.Clock).FontSize(12).TextColor(theme.Muted) })
		}
		ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
			ui.Text(c, r.title).FontSize(12.5)
			ui.Text(c, r.detail).FontSize(11.5).TextColor(theme.Muted)
		})
		switch r.kind {
		case awayNeeds:
			if widget.GhostButton(c.Key("answer"), "Answer").Clicked() {
				a.away.open = false
				a.page = pageInbox
			}
		case awayDone, awayStarted, awayMovedAccount:
			label := "Open"
			if r.kind == awayDone {
				label = "Review"
			}
			if widget.GhostButton(c.Key("open"), label).Clicked() {
				a.away.open = false
				a.openSessionByID(r.session)
			}
		}
	})
}

// openSessionByID shows session id, when it is still there.
func (a *App) openSessionByID(id string) {
	for _, s := range a.sessions {
		if s.ID == id {
			a.page = ""
			a.selectSession(s)
			return
		}
	}
}
