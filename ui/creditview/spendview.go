package creditview

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/client"
	"github.com/hustlex/hustlex/proto"
	"github.com/hustlex/hustlex/ui/icon"
	"github.com/hustlex/hustlex/ui/theme"
	"github.com/hustlex/hustlex/ui/widget"
)

// historyDays is how far back the page reads: the activity's 13 weeks.
const historyDays = 91

// spendState is the Spending section's: the history read and what the
// user picked. It lives in the Tracker, on the main thread.
type spendState struct {
	history   proto.UsageHistory
	loaded    bool
	err       string
	following *int // the poller's token while one runs
	days      int  // the range: 1, 7 or 30
	filter    string
	by        int    // Where it went: projects, sessions, models
	budgetFor string // the project whose budget is being typed
	budgetBuf string
}

// FollowHistory reads the host's token history while wanted (called on
// the main thread) reports true: often while the host reads it the first
// time, then every little while.
func (t *Tracker) FollowHistory(cl *client.Client, update func(func()), wanted func() bool) {
	if t.spend.following != nil {
		return
	}
	tok := new(int)
	t.spend.following = tok
	go func() {
		for {
			h, err := cl.UsageHistory(historyDays)
			stop := make(chan bool, 1)
			update(func() {
				if !wanted() || t.spend.following != tok {
					if t.spend.following == tok {
						t.spend.following = nil
					}
					stop <- true
					return
				}
				stop <- false
				t.spend.loaded = true
				t.spend.err = ""
				if err != nil {
					t.spend.err = describe(err)
					return
				}
				t.spend.history = h
			})
			if <-stop {
				return
			}
			wait := 15 * time.Second
			if err == nil && h.Reading {
				wait = 2 * time.Second
			}
			select {
			case <-cl.Done():
				update(func() {
					if t.spend.following == tok {
						t.spend.following = nil
					}
				})
				return
			case <-time.After(wait):
			}
		}
	}()
}

// rangeDays is the range picked, a week at first.
func (t *Tracker) rangeDays() int {
	if t.spend.days == 0 {
		return 7
	}
	return t.spend.days
}

// RangeControl picks the range the Spending section counts, for the
// page's header.
func (t *Tracker) RangeControl(c *ui.Context) {
	days := []int{1, 7, 30}
	on := 1
	for i, d := range days {
		if d == t.rangeDays() {
			on = i
		}
	}
	if i := segmented(c.Key("range"), []segItem{{label: "Today"}, {label: "7 days"}, {label: "30 days"}}, on); i >= 0 {
		t.spend.days = days[i]
	}
}

// segItem is one choice of a segmented control.
type segItem struct {
	label string
	svg   *ui.SVG
	tint  ui.Color
}

// segmented is a row of choices in a sunken track, the one on raised;
// it returns the one clicked, -1 for none.
func segmented(c *ui.Context, items []segItem, on int) int {
	clicked := -1
	ui.Row(c).Padding(2).Gap(2).Radius(8).Background(theme.Sunken).Border(1, theme.BorderStrong).Shrink(0).Children(func() {
		for i, it := range items {
			b := ui.ButtonBase(c.Key(i)).Height(22).PaddingX(10).Gap(5).Radius(6).Cursor(ui.CursorPointer).Label(it.label)
			col := theme.Muted
			switch {
			case i == on:
				b.Background(theme.Selected)
				col = theme.Text
			case b.Hovered():
				col = theme.TextSoft
			}
			b.Children(func() {
				if it.svg != nil {
					ui.Icon(c, it.svg).FontSize(12).TextColor(it.tint)
				}
				ui.Text(c, it.label).FontSize(12).TextColor(col).SingleLine()
			})
			if b.Clicked() {
				clicked = i
			}
		}
	})
	return clicked
}

// spending section: what the agents used, by day, kind and where.
func (t *Tracker) spending(c *ui.Context, h Host) {
	days := t.rangeDays()
	accs := t.spendAccounts()
	period := "last " + strconv.Itoa(days) + " days"
	if days == 1 {
		period = "today"
	}
	who := "all accounts"
	for _, a := range accs {
		if accountKey(a.Agent, a.Ref) == t.spend.filter {
			who = a.Account
		}
	}
	if !slicesContainsAccount(accs, t.spend.filter) {
		t.spend.filter, who = "", "all accounts"
	}
	ui.Row(c).FillWidth().Gap(8).Margin(18, 2, 8, 2).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icon.Wallet).FontSize(12).TextColor(theme.Muted)
		ui.Text(c, "SPENDING").FontSize(11).TextColor(theme.Muted).LetterSpacing(0.6).SingleLine()
		ui.Text(c, period+" · "+who).FontSize(11.5).TextColor(theme.TextSoft).SingleLine().Shrink(1)
		ui.Spacer(c)
		if len(accs) > 1 {
			items := []segItem{{label: "All accounts"}}
			on := 0
			for i, a := range accs {
				label := accountShortName(h, a)
				if a.Agent != "claude" {
					label += " · " + h.AgentName(a.Agent)
				}
				items = append(items, segItem{label: label, svg: icon.Agent(a.Agent), tint: icon.AgentTint(a.Agent, theme.Text)})
				if accountKey(a.Agent, a.Ref) == t.spend.filter {
					on = i + 1
				}
			}
			if i := segmented(c.Key("accounts"), items, on); i == 0 {
				t.spend.filter = ""
			} else if i > 0 {
				t.spend.filter = accountKey(accs[i-1].Agent, accs[i-1].Ref)
			}
		}
	})
	switch {
	case t.spend.err != "":
		ui.Text(c, t.spend.err).FontSize(12).TextColor(theme.Error)
		return
	case !t.spend.loaded || t.spend.history.Reading && len(t.spend.history.Rows) == 0:
		ui.Row(c).Gap(8).Padding(4, 2).Children(func() {
			widget.Spinner(c, 14)
			ui.Text(c, "Reading the agents' conversations on "+h.Name+"…").FontSize(12).TextColor(theme.Muted)
		})
		return
	}
	today := time.Now()
	s := spend(t.spend.history, today, days, t.spend.filter, names{project: h.projectName, session: h.sessionTitle})
	statsRow(c, s, days)
	pairs(c, "charts", func() { dailyBars(c, s) }, func() { tokenMix(c, s) })
	pairs(c, "where", func() { t.whereItWent(c, h, s) }, func() { activityCard(c, t.spend.history, today) })
	ui.Text(c, "Read from each agent's own conversation files on "+h.Name+". Costs are estimates at API prices; a plan bills differently.").
		FontSize(11).TextColor(theme.Muted).Margin(12, 2, 0, 2)
}

func slicesContainsAccount(accs []proto.CreditAccount, key string) bool {
	if key == "" {
		return true
	}
	for _, a := range accs {
		if accountKey(a.Agent, a.Ref) == key {
			return true
		}
	}
	return false
}

// spendAccounts are the accounts the Spending filter offers: those whose
// conversations hustleX reads.
func (t *Tracker) spendAccounts() []proto.CreditAccount {
	var out []proto.CreditAccount
	for _, a := range t.managed() {
		if a.Agent == "claude" || a.Agent == "codex" {
			out = append(out, a)
		}
	}
	return out
}

// pairs lays two cards side by side, the first wider, or one above the
// other in a narrow window.
func pairs(c *ui.Context, key string, first, second func()) {
	box := ui.Column(c.Key(key)).FillWidth().Margin(10, 0, 0, 0)
	wide := box.Bounds().W == 0 || box.Bounds().W >= 820
	box.Children(func() {
		if wide {
			ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Stretch).Children(func() {
				ui.Column(c).Grow(1.55).Basis(0).MinWidth(0).Children(first)
				ui.Column(c).Grow(1).Basis(0).MinWidth(0).Children(second)
			})
			return
		}
		ui.Column(c).FillWidth().Gap(10).Children(func() {
			ui.Column(c).FillWidth().Children(first)
			ui.Column(c).FillWidth().Children(second)
		})
	})
}

func card(c *ui.Context, svg *ui.SVG, title string, extra func(), body func()) {
	ui.Column(c).FillWidth().Grow(1).Padding(12, 14).Radius(theme.RadiusCard).Background(theme.Raised).Border(1, theme.Border).Children(func() {
		ui.Row(c).FillWidth().Gap(8).Margin(0, 0, 10, 0).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, svg).FontSize(14).TextColor(theme.Muted)
			ui.Text(c, title).FontSize(12.5).SingleLine()
			ui.Spacer(c)
			if extra != nil {
				extra()
			}
		})
		body()
	})
}

func pill(c *ui.Context, svg *ui.SVG, label string) {
	ui.Row(c).Height(18).PaddingX(7).Gap(3).Radius(6).Border(1, theme.BorderStrong).AlignItems(ui.Center).Shrink(0).Children(func() {
		if svg != nil {
			ui.Icon(c, svg).FontSize(10).TextColor(theme.TextSoft)
		}
		ui.Text(c, label).FontSize(10.5).TextColor(theme.TextSoft).SingleLine()
	})
}

// statsRow is the range's totals beside the range before it.
func statsRow(c *ui.Context, s spending, days int) {
	type stat struct {
		svg    *ui.SVG
		value  string
		label  string
		change string
		down   bool
		tip    string
	}
	change := func(now, before float64) (string, bool) {
		if before == 0 {
			return "", false
		}
		p := math.Round((now - before) / before * 100)
		if p == 0 {
			return "", false
		}
		if p < 0 {
			return fmt.Sprintf("▼%.0f%%", -p), true
		}
		return fmt.Sprintf("▲%.0f%%", p), false
	}
	tc, td := change(float64(s.total.total()), float64(s.prev.total()))
	cc, cd := change(s.total.cost, s.prev.cost)
	sc, sd := change(float64(s.conversations), float64(s.prevConvs))
	costTip := "What these tokens would cost at API prices. Your plan bills differently."
	if s.total.unpriced > 0 {
		costTip += " " + tokens(s.total.unpriced) + " tokens of models without a known price aren't counted."
	}
	stats := []stat{
		{icon.Tokens, tokens(s.total.total()), "Tokens", tc, td, ""},
		{icon.Dollar, dollars(s.total.cost), "Est. cost", cc, cd, costTip},
		{icon.Terminal, strconv.Itoa(s.conversations), "Sessions", sc, sd, "Conversations with the agents in the range"},
		{icon.Calendar, strconv.Itoa(s.active) + "/" + strconv.Itoa(days), "Active days", "", false, ""},
		{icon.Database, fmt.Sprintf("%.0f%%", s.total.cacheHit()*100), "Cache hit", "", false, "Of what the models read, how much came from the cache"},
	}
	grid := ui.Grid(c).FillWidth().Radius(theme.RadiusCard).Background(theme.Raised).Border(1, theme.Border).Clip()
	cols := 5
	if w := grid.Bounds().W; w > 0 && w < 760 {
		cols = 2
	}
	grid.Columns(cols).Children(func() {
		for i, st := range stats {
			cell := ui.Row(c.Key(i)).Padding(11, 14).Gap(11).AlignItems(ui.Center).MinWidth(0)
			if i%cols > 0 {
				cell.BorderWidth(0, 0, 0, 1).BorderColor(theme.Border)
			}
			if i >= cols {
				cell.BorderWidth(1, 0, 0, max(0, float32(min(1, i%cols)))).BorderColor(theme.Border)
			}
			if st.tip != "" {
				cell.Tooltip(st.tip)
			}
			cell.Children(func() {
				ui.Box(c).Size(32, 32).Radius(9).Background(theme.Selected).Border(1, theme.BorderStrong).Center().Shrink(0).Children(func() {
					ui.Icon(c, st.svg).FontSize(15).TextColor(theme.TextSoft)
				})
				ui.Column(c).MinWidth(0).Gap(1).Children(func() {
					ui.Text(c, st.value).FontSize(18).SingleLine()
					ui.Row(c).Gap(6).Children(func() {
						ui.Text(c, st.label).FontSize(11).TextColor(theme.Muted).SingleLine()
						if st.change != "" {
							col := theme.Done
							if st.down {
								col = theme.Error
							}
							ui.Text(c, st.change).FontSize(11).TextColor(col).SingleLine()
						}
					})
				})
			})
		}
	})
}

// kinds are the kinds of tokens, in the charts' order and colors.
func kinds() []struct {
	name string
	col  ui.Color
	of   func(usage) int64
} {
	return []struct {
		name string
		col  ui.Color
		of   func(usage) int64
	}{
		{"Cache read", theme.Mix(theme.Text, theme.Raised, 0.08), func(u usage) int64 { return u.cacheRead }},
		{"Cache write", theme.Mix(theme.Text, theme.Raised, 0.45), func(u usage) int64 { return u.cacheCreate }},
		{"Input", theme.Working, func(u usage) int64 { return u.input }},
		{"Output", theme.Done, func(u usage) int64 { return u.output }},
	}
}

// dayTip is a day's tokens by kind, for its bar and its square.
func dayTip(d dayUsage, withCost bool) string {
	lines := []string{d.day.Format("Mon, Jan 2")}
	if d.total() == 0 {
		return lines[0] + "\nNo tokens"
	}
	if !withCost {
		return lines[0] + "\n" + tokens(d.total()) + " tokens"
	}
	for _, k := range kinds() {
		lines = append(lines, k.name+"  "+tokens(k.of(d.usage)))
	}
	lines = append(lines, "Est. cost  "+dollars(d.cost))
	return strings.Join(lines, "\n")
}

// dailyBars are the tokens of each day, a bar each split by kind.
func dailyBars(c *ui.Context, s spending) {
	var most dayUsage
	for _, d := range s.chart {
		if d.total() > most.total() {
			most = d
		}
	}
	extra := func() {
		if most.total() > 0 {
			pill(c, icon.Flame, "Busiest "+most.day.Format("Jan 2")+" · "+tokens(most.total()))
		}
	}
	card(c, icon.Chart, "Tokens per day", extra, func() {
		const height = 92
		ui.Row(c).FillWidth().Height(height).Gap(3).AlignItems(ui.End).Children(func() {
			for i, d := range s.chart {
				h := float32(2)
				if most.total() > 0 && d.total() > 0 {
					h = max(3, float32(d.total())/float32(most.total())*height)
				}
				bar := ui.Column(c.Key(i)).Grow(1).Basis(0).MinWidth(0).Height(h).Radius(3).Clip().Tooltip(dayTip(d, true))
				if d.total() == 0 {
					bar.Background(theme.Selected)
					continue
				}
				bar.Children(func() {
					ks := kinds()
					for j := len(ks) - 1; j >= 0; j-- {
						if n := ks[j].of(d.usage); n > 0 {
							ui.Box(c.Key(j)).FillWidth().Grow(float32(n)).Basis(0).Background(ks[j].col)
						}
					}
				})
			}
		})
		ui.Row(c).FillWidth().Margin(6, 0, 0, 0).Children(func() {
			if len(s.chart) > 0 {
				ui.Text(c, s.chart[0].day.Format("Jan 2")).FontSize(10.5).TextColor(theme.Muted)
			}
			ui.Spacer(c)
			ui.Text(c, "Today").FontSize(10.5).TextColor(theme.Muted)
		})
	})
}

// tokenMix is the range's tokens by kind.
func tokenMix(c *ui.Context, s spending) {
	card(c, icon.Pie, "Token mix", nil, func() {
		total := s.total.total()
		ui.Row(c).FillWidth().Height(8).Radius(4).Clip().Background(theme.Selected).Margin(0, 0, 10, 0).Children(func() {
			for i, k := range kinds() {
				if n := k.of(s.total); n > 0 {
					ui.Box(c.Key(i)).FillHeight().Grow(float32(n)).Basis(0).Background(k.col)
				}
			}
		})
		ui.Grid(c).FillWidth().Columns(2).GapX(14).GapY(6).Children(func() {
			for i, k := range kinds() {
				ui.Row(c.Key(i)).Gap(7).AlignItems(ui.Center).MinWidth(0).Children(func() {
					ui.Box(c).Size(7, 7).Radius(2).Background(k.col).Shrink(0)
					ui.Text(c, k.name).FontSize(11.5).TextColor(theme.TextSoft).SingleLine().Shrink(1)
					ui.Spacer(c)
					ui.Text(c, tokens(k.of(s.total))).FontSize(11.5).SingleLine()
				})
			}
		})
		if total > 0 && s.total.saved >= 1 {
			ui.Row(c).Margin(10, 0, 0, 0).Gap(4).Wrap().Children(func() {
				ui.Text(c, "Cache saved about").FontSize(11).TextColor(theme.Muted)
				ui.Text(c, dollars(s.total.saved)).FontSize(11).TextColor(theme.Done)
				ui.Text(c, "vs reading it all as new input.").FontSize(11).TextColor(theme.Muted)
			})
		}
	})
}

// whereItWent is the range's tokens by project, session or model, with
// each project's weekly budget.
func (t *Tracker) whereItWent(c *ui.Context, h Host, s spending) {
	tabs := func() {
		ui.Row(c).Gap(2).Children(func() {
			for i, label := range []string{"Projects", "Sessions", "Models"} {
				b := ui.ButtonBase(c.Key(i)).Height(22).PaddingX(8).Radius(6).Cursor(ui.CursorPointer).Label(label)
				col := theme.Muted
				if i == t.spend.by {
					b.Background(theme.Selected)
					col = theme.Text
				} else if b.Hovered() {
					col = theme.TextSoft
				}
				b.Children(func() { ui.Text(c, label).FontSize(11.5).TextColor(col).SingleLine() })
				if b.Clicked() {
					t.spend.by = i
				}
			}
		})
	}
	card(c, icon.Route, "Where it went", tabs, func() {
		rows := [][]share{s.byProject, s.bySession, s.byMod}[t.spend.by]
		if len(rows) == 0 {
			ui.Text(c, "No tokens in this range.").FontSize(12).TextColor(theme.Muted).Padding(6, 4)
			return
		}
		top := rows[0].total()
		total := s.total.total()
		for i, r := range rows[:min(len(rows), 6)] {
			row := ui.Row(c.Key(r.key)).FillWidth().Gap(8).Padding(6, 4).AlignItems(ui.Center)
			if i > 0 {
				row.BorderWidth(1, 0, 0, 0).BorderColor(theme.Border)
			}
			row.Children(func() {
				ui.Row(c).Grow(1).MinWidth(0).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(17, 17).Radius(5).Background(theme.Selected).Center().Shrink(0).Children(func() {
						switch {
						case r.agent != "":
							ui.Icon(c, icon.Agent(r.agent)).FontSize(11).TextColor(icon.AgentTint(r.agent, theme.Text))
						default:
							ui.Icon(c, icon.Chat).FontSize(11).TextColor(theme.Muted)
						}
					})
					ui.Text(c, r.name).FontSize(12).SingleLine().Shrink(1)
				})
				ui.Row(c).Width(96).Shrink(0).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Width(56).Children(func() {
						widget.Meter(c, widget.Segment{Fraction: float64(r.total()) / float64(max(top, 1)), Color: theme.Accent})
					})
					ui.Text(c, fmt.Sprintf("%.0f%%", float64(r.total())/float64(max(total, 1))*100)).FontSize(12).TextColor(theme.Muted).SingleLine()
				})
				ui.Text(c, tokens(r.total())).FontSize(12).TextColor(theme.TextSoft).Width(48).TextAlign(ui.End).Shrink(0).SingleLine()
				ui.Text(c, dollars(r.cost)).FontSize(12).TextColor(theme.TextSoft).Width(54).TextAlign(ui.End).Shrink(0).SingleLine()
				if t.spend.by == 0 && h.Budget != nil {
					t.budgetCell(c, h, r.key, s.week[r.key])
				}
			})
		}
	})
}

// budgetCell is a project's weekly budget: how much of it the last seven
// days used, and a way to set it.
func (t *Tracker) budgetCell(c *ui.Context, h Host, project string, week float64) {
	ui.Row(c).Width(122).Shrink(0).Gap(6).AlignItems(ui.Center).Justify(ui.End).Children(func() {
		if t.spend.budgetFor == project {
			in := widget.Input(c.Key("budget"), &t.spend.budgetBuf, "$ a week").Width(80).Height(24).AutoFocus()
			save := func() {
				v, err := strconv.ParseFloat(strings.TrimPrefix(strings.TrimSpace(t.spend.budgetBuf), "$"), 64)
				if err != nil || v < 0 {
					v = 0
				}
				h.SetBudget(project, v)
				t.spend.budgetFor = ""
			}
			if in.Submitted() {
				save()
			}
			if smallButton(c.Key("save"), "Save") {
				save()
			}
			return
		}
		budget := h.Budget(project)
		if budget > 0 {
			pct := week / budget * 100
			col := theme.Muted
			switch {
			case pct >= 100:
				col = theme.Error
			case pct >= 80:
				col = theme.Warning
			}
			ui.Text(c, fmt.Sprintf("%.0f%% of %s", pct, dollars(budget))).FontSize(11.5).TextColor(col).SingleLine().
				Tooltip("Of its weekly budget, in the last seven days")
		}
		label := "Set budget"
		if budget > 0 {
			label = "Edit"
		}
		if smallButton(c.Key("edit"), label) {
			t.spend.budgetFor = project
			t.spend.budgetBuf = ""
			if budget > 0 {
				t.spend.budgetBuf = strconv.FormatFloat(budget, 'f', -1, 64)
			}
		}
	})
}

func smallButton(c *ui.Context, label string) bool {
	b := ui.ButtonBase(c).Height(20).PaddingX(7).Radius(6).Border(1, theme.BorderStrong).Cursor(ui.CursorPointer).Label(label).Shrink(0)
	col := theme.Muted
	if b.Hovered() {
		col = theme.Text
	}
	b.Children(func() { ui.Text(c, label).FontSize(11).TextColor(col).SingleLine() })
	return b.Clicked()
}

// activityCard is the last weeks' days as squares, darker the more tokens.
func activityCard(c *ui.Context, h proto.UsageHistory, today time.Time) {
	days := activity(h, today, 13)
	var most int64
	for _, d := range days {
		most = max(most, d.total())
	}
	levels := []ui.Color{theme.Selected, theme.Fade(theme.Text, 0.16), theme.Fade(theme.Text, 0.32), theme.Fade(theme.Text, 0.58), theme.Fade(theme.Text, 0.9)}
	levelOf := func(n int64) ui.Color {
		switch f := float64(n) / float64(max(most, 1)); {
		case n == 0:
			return levels[0]
		case f < 0.2:
			return levels[1]
		case f < 0.45:
			return levels[2]
		case f < 0.75:
			return levels[3]
		}
		return levels[4]
	}
	extra := func() {
		if n := streak(days); n > 0 {
			pill(c, icon.Flame, strconv.Itoa(n)+"-day streak")
		}
	}
	card(c, icon.Calendar, "Activity", extra, func() {
		ui.Row(c).Gap(3).Children(func() {
			for w := 0; w*7 < len(days); w++ {
				ui.Column(c.Key(w)).Gap(3).Children(func() {
					for _, d := range days[w*7 : min(len(days), w*7+7)] {
						ui.Box(c.Key(dayKey(d.day))).Size(10, 10).Radius(2).Background(levelOf(d.total())).Tooltip(dayTip(d, false))
					}
				})
			}
		})
		ui.Row(c).Gap(3).Margin(8, 0, 0, 0).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Less").FontSize(10.5).TextColor(theme.Muted)
			for i, l := range levels {
				ui.Box(c.Key(i)).Size(9, 9).Radius(2).Background(l)
			}
			ui.Text(c, "More").FontSize(10.5).TextColor(theme.Muted)
		})
	})
}

// tokens reads a count of tokens short: 1.8B, 452.5M, 12.3k.
func tokens(n int64) string {
	f := float64(n)
	switch {
	case f >= 1e9:
		return fmt.Sprintf("%.1fB", f/1e9)
	case f >= 1e6:
		return fmt.Sprintf("%.1fM", f/1e6)
	case f >= 1e3:
		return fmt.Sprintf("%.1fk", f/1e3)
	}
	return strconv.FormatInt(n, 10)
}

// dollars reads an amount: cents below a hundred.
func dollars(v float64) string {
	if v >= 100 {
		return "$" + strconv.FormatFloat(math.Round(v), 'f', 0, 64)
	}
	return fmt.Sprintf("$%.2f", v)
}

// SetHistory shows h as the host's token history, without reading it.
func (t *Tracker) SetHistory(h proto.UsageHistory) {
	t.spend.history, t.spend.loaded = h, true
	t.spend.following = new(int)
}
