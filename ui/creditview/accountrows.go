package creditview

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/proto"
	"github.com/hustlex/hustlex/ui/icon"
	"github.com/hustlex/hustlex/ui/textfmt"
	"github.com/hustlex/hustlex/ui/theme"
	"github.com/hustlex/hustlex/ui/widget"
)

// accountRows are the accounts on the Credits page, a row each: who, each
// limit with a meter, whether its pace lasts, and which is in use.
func (t *Tracker) accountRows(c *ui.Context, h Host) {
	box := ui.Column(c).FillWidth().Radius(theme.RadiusCard).Background(theme.Raised).Border(1, theme.Border).Clip()
	narrow := box.Bounds().W > 0 && box.Bounds().W < 760
	box.Children(func() {
		for i, a := range t.report.Accounts {
			t.accountRow(c.Key(a.Agent+" "+a.Ref+" "+strconv.Itoa(i)), h, a, i > 0, narrow)
		}
	})
}

// accountRow is one account; narrow, its limits go under who it is.
func (t *Tracker) accountRow(c *ui.Context, h Host, a proto.CreditAccount, divider, narrow bool) {
	inUse := a.Ref != "" && h.Active != nil && h.Active(a.Agent) == a.Ref
	row := ui.Column(c).FillWidth().Padding(10, 12).Gap(8)
	if divider {
		row.BorderWidth(1, 0, 0, 0).BorderColor(theme.Border)
	}
	if inUse {
		row.LinearGradient(ui.LinearGradient{Angle: 90, From: theme.Fade(theme.Text, 0.05), To: theme.Fade(theme.Text, 0), End: 0.4})
	}
	row.Children(func() {
		ui.Row(c).FillWidth().Gap(16).AlignItems(ui.Center).Children(func() {
			who := ui.Row(c).Gap(9).AlignItems(ui.Center)
			if narrow {
				who.Grow(1).MinWidth(0)
			} else {
				who.Width(200).Shrink(0)
			}
			who.Children(func() {
				widget.Avatar(c, icon.Agent(a.Agent), 26, icon.AgentTint(a.Agent, theme.Text))
				ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
					who := a.Account
					if who == "" {
						who = h.AgentName(a.Agent)
					}
					ui.Text(c, who).FontSize(12.5).SingleLine()
					plan := a.Plan
					if plan == "" {
						plan = h.AgentName(a.Agent)
					}
					ui.Text(c, plan).FontSize(11).TextColor(theme.Muted).SingleLine()
				})
			})
			if !narrow {
				limits(c, a)
			}
			ui.Row(c).MinWidth(150).Shrink(0).Gap(8).AlignItems(ui.Center).Justify(ui.End).Children(func() {
				if ok, text, known := pace(a, time.Now()); known {
					mark := ui.Box(c).Size(20, 20).Radius(6).Center().Tooltip(text)
					if ok {
						mark.Background(theme.Fade(theme.Done, 0.12)).TextColor(theme.Done).Children(func() { ui.Icon(c, icon.Check).FontSize(12) })
					} else {
						mark.Background(theme.Fade(theme.Warning, 0.13)).TextColor(theme.Warning).Children(func() { ui.Icon(c, icon.Alert).FontSize(12) })
					}
				}
				switch {
				case inUse:
					ui.Box(c).PaddingX(7).Height(20).Radius(6).Background(theme.Fade(theme.Text, 0.1)).Center().Children(func() {
						ui.Text(c, "In use").FontSize(10.5).TextColor(theme.Accent).SingleLine()
					})
				case a.Ref != "" && h.Switch != nil && len(a.Windows) > 0:
					b := ui.ButtonBase(c.Key("use")).Height(24).PaddingX(9).Radius(7).Border(1, theme.BorderStrong).
						Cursor(ui.CursorPointer).Label("Use for new sessions")
					col := theme.TextSoft
					if b.Hovered() {
						col = theme.Text
					}
					b.Children(func() { ui.Text(c, "Use for new sessions").FontSize(11.5).TextColor(col).SingleLine() })
					if b.Clicked() {
						h.Switch(a.Agent, a.Ref, accountShortName(h, a))
					}
				}
			})
		})
		if narrow && len(a.Windows) > 0 {
			ui.Row(c).FillWidth().Padding(0, 0, 0, 35).Children(func() { limits(c, a) })
		}
		problem(c, h, a, true)
		if a.Note != "" {
			ui.Text(c, a.Note).FontSize(11).TextColor(theme.Muted).Margin(0, 0, 0, 35)
		}
	})
}

// limits are an account's first three limits, side by side.
func limits(c *ui.Context, a proto.CreditAccount) {
	ui.Row(c).Grow(1).MinWidth(0).Gap(14).Children(func() {
		for i, w := range a.Windows[:min(len(a.Windows), 3)] {
			limit(c.Key(i), w)
		}
		for i := len(a.Windows); i < 3; i++ {
			ui.Box(c.Key("pad" + strconv.Itoa(i))).Grow(1).Basis(0)
		}
	})
}

// limit is one of an account's limits: its name and share used, a meter,
// when it starts over.
func limit(c *ui.Context, w proto.CreditWindow) {
	ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(4).Children(func() {
		ui.Row(c).FillWidth().Gap(5).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, windowIcon(w)).FontSize(11).TextColor(theme.Muted)
			ui.Text(c, w.Label).FontSize(11).TextColor(theme.Muted).SingleLine().Shrink(1)
			ui.Spacer(c)
			ui.Text(c, fmt.Sprintf("%.0f%%", w.Used)).FontSize(11).SingleLine()
		})
		widget.Meter(c, widget.Segment{Fraction: w.Used / 100, Color: level(w.Used)})
		if !w.ResetsAt.IsZero() {
			ui.Text(c, "resets "+resetsWhen(w.ResetsAt, time.Now())).FontSize(10.5).TextColor(theme.Muted).SingleLine()
		} else if w.Detail != "" {
			ui.Text(c, w.Detail).FontSize(10.5).TextColor(theme.Muted).SingleLine()
		}
	})
}

func windowIcon(w proto.CreditWindow) *ui.SVG {
	switch {
	case strings.HasSuffix(w.Label, "hours"):
		return icon.Clock
	case w.Label == "Week" || w.Label == "Month" || strings.HasSuffix(w.Label, "days"):
		return icon.Week
	}
	return icon.Sparkle // a model's own limit: "Week · Fable"
}

// resetsWhen says when a limit starts over: in how long when soon, the
// day and time otherwise.
func resetsWhen(at, now time.Time) string {
	if at.Sub(now) < 24*time.Hour {
		return "in " + textfmt.Until(at, now)
	}
	return at.Local().Format("Mon 15:04")
}

// windowLength is how long a limit lasts before it starts over, from its
// label; 0 when it doesn't say.
func windowLength(label string) time.Duration {
	switch {
	case label == "Week" || strings.HasPrefix(label, "Week ·"):
		return 7 * 24 * time.Hour
	case label == "Month":
		return 30 * 24 * time.Hour
	}
	for _, unit := range []struct {
		suffix string
		d      time.Duration
	}{{" hours", time.Hour}, {" days", 24 * time.Hour}} {
		if n, ok := strings.CutSuffix(label, unit.suffix); ok {
			if v, err := strconv.Atoi(n); err == nil {
				return time.Duration(v) * unit.d
			}
		}
	}
	return 0
}

// pace says whether account a lasts its longest limit at the pace it is
// used: ok, and a sentence why; known is false when it can't tell yet.
func pace(a proto.CreditAccount, now time.Time) (ok bool, text string, known bool) {
	var w proto.CreditWindow
	var length time.Duration
	for _, x := range a.Windows {
		if l := windowLength(x.Label); l > length && !x.ResetsAt.IsZero() && !strings.Contains(x.Label, "·") {
			w, length = x, l
		}
	}
	if length == 0 {
		return false, "", false
	}
	name := "the " + strings.ToLower(w.Label)
	if w.Label == "Week" {
		name = "the week"
	}
	if w.Used >= 100 {
		return false, "Used up until " + name + " resets " + resetsWhen(w.ResetsAt, now), true
	}
	elapsed := length - w.ResetsAt.Sub(now)
	if elapsed < length/20 || elapsed <= 0 {
		return true, "Just started: " + fmt.Sprintf("%.0f%%", w.Used) + " of " + name + " used", true
	}
	projected := w.Used * float64(length) / float64(elapsed)
	if projected <= 100 {
		return true, fmt.Sprintf("On track: about %.0f%% of %s by its reset", projected, name), true
	}
	out := now.Add(time.Duration(float64(elapsed) * (100 - w.Used) / w.Used))
	return false, "At this pace " + name + " runs out " + partOfDay(out, now), true
}

// partOfDay names when t is: "Thursday evening", "this afternoon".
func partOfDay(t, now time.Time) string {
	t, now = t.Local(), now.Local()
	part := "night"
	switch h := t.Hour(); {
	case h >= 5 && h < 12:
		part = "morning"
	case h >= 12 && h < 17:
		part = "afternoon"
	case h >= 17 && h < 22:
		part = "evening"
	}
	y1, m1, d1 := t.Date()
	y2, m2, d2 := now.Date()
	if y1 == y2 && m1 == m2 && d1 == d2 {
		if part == "night" {
			return "tonight"
		}
		return "this " + part
	}
	return t.Weekday().String() + " " + part
}

// accountShortName is what the Spending filter and messages call an
// account: the user's name for it, else its email before the @.
func accountShortName(h Host, a proto.CreditAccount) string {
	if h.AccountName != nil {
		if n := h.AccountName(a.Agent, a.Ref); n != "" {
			return n
		}
	}
	if who, _, ok := strings.Cut(a.Account, "@"); ok && who != "" {
		return who
	}
	if a.Account != "" {
		return a.Account
	}
	return h.AgentName(a.Agent)
}
