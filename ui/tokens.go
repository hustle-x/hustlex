package ui

import (
	"fmt"
	"slices"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/proto"
	"github.com/hustlex/hustlex/ui/icon"
	"github.com/hustlex/hustlex/ui/textfmt"
	"github.com/hustlex/hustlex/ui/theme"
	"github.com/hustlex/hustlex/ui/widget"
)

// tokenStats is the tokens of the host's sessions, read while the Resource
// usage page is open.
type tokenStats struct {
	report  proto.TokenReport
	asked   time.Time
	loading bool
}

const tokenEvery = 10 * time.Second

// refreshTokens asks the daemon for the sessions' tokens, at most every
// tokenEvery, on a goroutine of its own: the first reading scans the files.
func (a *App) refreshTokens() {
	t := &a.tokens
	cl := a.client
	if cl == nil || t.loading || time.Since(t.asked) < tokenEvery {
		return
	}
	t.loading, t.asked = true, time.Now()
	gen := a.gen
	go func() {
		rep, err := cl.Tokens()
		a.onMain(func() {
			t.loading = false
			if err == nil && gen == a.gen {
				t.report = rep
			}
		})
	}()
}

// compact writes n as 1.2k, 3.4M.
func compact(n int64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1e3)
	}
	return fmt.Sprint(n)
}

// tokensSection is the tokens each session's agent used: by session, with
// the share of the input that came from the cache.
func (a *App) tokensSection(c *ui.Context) {
	a.refreshTokens()
	rows := slices.Clone(a.tokens.report.Sessions)
	slices.SortFunc(rows, func(x, y proto.SessionTokens) int {
		return int((y.Input + y.CacheRead + y.CacheCreate + y.Output) - (x.Input + x.CacheRead + x.CacheCreate + x.Output))
	})
	var total proto.SessionTokens
	for _, r := range rows {
		total.Input += r.Input
		total.CacheRead += r.CacheRead
		total.CacheCreate += r.CacheCreate
		total.Output += r.Output
	}
	ui.Column(c).FillWidth().Gap(6).Margin(22, 0, 0, 0).Children(func() {
		widget.SectionLabel(c, "TOKENS", theme.Muted)
		if len(rows) == 0 {
			text := "No Claude Code session has used tokens yet."
			if a.tokens.loading || a.tokens.asked.IsZero() {
				text = "Counting…"
			}
			ui.Text(c, text).FontSize(12).TextColor(theme.Muted)
			return
		}
		line := func(key any, label func(), r proto.SessionTokens, strong bool) {
			ui.Row(c.Key(key)).FillWidth().Gap(10).PaddingX(10).Height(30).AlignItems(ui.Center).Children(func() {
				ui.Row(c).Grow(1).MinWidth(0).Gap(8).AlignItems(ui.Center).Children(label)
				for _, n := range []int64{r.Input, r.CacheRead + r.CacheCreate, r.Output} {
					t := ui.Text(c, compact(n)).FontSize(12).Width(70).TextAlign(ui.End)
					if strong {
						t.FontWeight(600)
					} else {
						t.TextColor(theme.TextSoft)
					}
				}
			})
		}
		head := func(s string) {
			ui.Text(c, s).FontSize(11).TextColor(theme.Muted).Width(70).TextAlign(ui.End)
		}
		ui.Row(c).FillWidth().Gap(10).PaddingX(10).Children(func() {
			ui.Spacer(c)
			head("Input")
			head("Cache")
			head("Output")
		})
		for _, r := range rows {
			s, _ := a.sessionByID(r.ID)
			name := a.sessionName(s)
			line(r.ID, func() {
				ui.Icon(c, icon.Agent(s.Agent)).FontSize(13)
				ui.Text(c, name).FontSize(12).SingleLine().Shrink(1)
				if r.Model != "" {
					ui.Text(c, r.Model).FontSize(11).TextColor(theme.Muted).SingleLine().Shrink(0)
				}
				if s.Project != "" {
					ui.Text(c, textfmt.LastPathPart(s.Project)).FontSize(11).TextColor(theme.Muted).SingleLine().Shrink(0)
				}
			}, r, false)
		}
		widget.Divider(c)
		line("total", func() { ui.Text(c, "All sessions").FontSize(12) }, total, true)
		if in := total.Input + total.CacheRead + total.CacheCreate; in > 0 {
			ui.Text(c, fmt.Sprintf("%.0f%% of what the agents read came from the cache. Counted from the agents' own conversation files, so it covers what they recorded, not other tools.",
				float64(total.CacheRead)/float64(in)*100)).FontSize(11).TextColor(theme.Muted)
		}
	})
}
