package creditview

import (
	"cmp"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/hustlex/hustlex/proto"
)

// usage is tokens by kind with what they would cost.
type usage struct {
	input, cacheRead, cacheCreate, output int64
	cost                                  float64
	unpriced                              int64 // tokens of models without a known price
	saved                                 float64
}

func (u usage) total() int64 { return u.input + u.cacheRead + u.cacheCreate + u.output }

func (u *usage) add(model string, r proto.UsageRow) {
	t := usage{input: r.Input, cacheRead: r.CacheRead, cacheCreate: r.CacheCreate, output: r.Output}
	u.input += t.input
	u.cacheRead += t.cacheRead
	u.cacheCreate += t.cacheCreate
	u.output += t.output
	if c, ok := cost(model, t); ok {
		u.cost += c
		u.saved += cacheSaving(model, t)
	} else {
		u.unpriced += t.total()
	}
}

func (u *usage) merge(o usage) {
	u.input += o.input
	u.cacheRead += o.cacheRead
	u.cacheCreate += o.cacheCreate
	u.output += o.output
	u.cost += o.cost
	u.unpriced += o.unpriced
	u.saved += o.saved
}

// cacheHit is how much of what the model read came from the cache.
func (u usage) cacheHit() float64 {
	read := u.cacheRead + u.input + u.cacheCreate
	if read == 0 {
		return 0
	}
	return float64(u.cacheRead) / float64(read)
}

// dayUsage is one day's tokens.
type dayUsage struct {
	day time.Time
	usage
	conversations int
}

// share is one project, session or model and its tokens.
type share struct {
	key, name string
	agent     string // its agent, for the logo; "" for a session
	usage
}

// spending is what the Spending section shows for a range and account.
type spending struct {
	total, prev                 usage
	conversations, prevConvs    int
	active                      int        // days with tokens
	chart                       []dayUsage // the bars: the range, two weeks at least
	byProject, bySession, byMod []share
	week                        map[string]float64 // each project's cost in the last seven days, for budgets
}

// dayKey is a day as the daemon names it.
func dayKey(t time.Time) string { return t.Format(time.DateOnly) }

// accountKey is an account as the Spending filter names it.
func accountKey(agent, ref string) string { return agent + " " + ref }

// names are what the Spending section calls projects and sessions.
type names struct {
	project func(path string) string
	session func(id string) (string, bool)
}

// spend sums history h for the days days long range ending today, of the
// account filter ("" for all).
func spend(h proto.UsageHistory, today time.Time, days int, filter string, n names) spending {
	keep := func(c proto.UsageConversation) bool {
		return filter == "" || accountKey(c.Agent, c.Account) == filter
	}
	from := dayKey(today.AddDate(0, 0, -(days - 1)))
	prevFrom := dayKey(today.AddDate(0, 0, -(2*days - 1)))
	weekFrom := dayKey(today.AddDate(0, 0, -6))
	chartDays := max(days, 14)
	chartFrom := today.AddDate(0, 0, -(chartDays - 1))

	var s spending
	s.week = map[string]float64{}
	byDay := map[string]*dayUsage{}
	dayConvs := map[string]map[int]bool{}
	convs, prevConvs := map[int]bool{}, map[int]bool{}
	projects, sessions, models := map[string]*share{}, map[string]*share{}, map[string]*share{}
	for _, r := range h.Rows {
		if r.Conversation < 0 || r.Conversation >= len(h.Conversations) {
			continue
		}
		c := h.Conversations[r.Conversation]
		if !keep(c) {
			continue
		}
		var u usage
		u.add(r.Model, r)
		if r.Day >= weekFrom {
			s.week[c.Project] += u.cost
		}
		if r.Day >= dayKey(chartFrom) {
			d := byDay[r.Day]
			if d == nil {
				d = &dayUsage{}
				byDay[r.Day] = d
				dayConvs[r.Day] = map[int]bool{}
			}
			d.merge(u)
			dayConvs[r.Day][r.Conversation] = true
		}
		switch {
		case r.Day >= from:
			s.total.merge(u)
			convs[r.Conversation] = true
			addShare(projects, c.Project, n.project(c.Project), c.Agent, u)
			sk := c.Session
			if sk == "" {
				sk = c.Agent + ":" + strconv.Itoa(r.Conversation)
			}
			title, ok := "", false
			if c.Session != "" {
				title, ok = n.session(c.Session)
			}
			if !ok {
				title = cmp.Or(c.Title, "Conversation in "+filepath.Base(c.Project))
			}
			addShare(sessions, sk, title, "", u)
			addShare(models, r.Model, r.Model, c.Agent, u)
		case r.Day >= prevFrom:
			s.prev.merge(u)
			prevConvs[r.Conversation] = true
		}
	}
	s.conversations, s.prevConvs = len(convs), len(prevConvs)
	for i := range chartDays {
		day := chartFrom.AddDate(0, 0, i)
		d := dayUsage{day: day}
		if b := byDay[dayKey(day)]; b != nil {
			d.usage, d.conversations = b.usage, len(dayConvs[dayKey(day)])
		}
		if i >= chartDays-days && d.total() > 0 {
			s.active++
		}
		s.chart = append(s.chart, d)
	}
	s.byProject, s.bySession, s.byMod = sorted(projects), sorted(sessions), sorted(models)
	return s
}

func addShare(m map[string]*share, key, name, agent string, u usage) {
	sh := m[key]
	if sh == nil {
		sh = &share{key: key, name: name, agent: agent}
		m[key] = sh
	}
	sh.merge(u)
}

func sorted(m map[string]*share) []share {
	out := make([]share, 0, len(m))
	for _, s := range m {
		out = append(out, *s)
	}
	slices.SortFunc(out, func(a, b share) int {
		return cmp.Or(cmp.Compare(b.total(), a.total()), cmp.Compare(a.name, b.name))
	})
	return out
}

// activity is the tokens of each of the last weeks' days, all accounts,
// oldest first, starting on a Sunday so the weeks line up.
func activity(h proto.UsageHistory, today time.Time, weeks int) []dayUsage {
	start := today.AddDate(0, 0, -(weeks*7 - 1))
	start = start.AddDate(0, 0, -int(start.Weekday()))
	totals := map[string]int64{}
	for _, r := range h.Rows {
		totals[r.Day] += r.Input + r.CacheRead + r.CacheCreate + r.Output
	}
	var out []dayUsage
	for d := start; !d.After(today); d = d.AddDate(0, 0, 1) {
		out = append(out, dayUsage{day: d, usage: usage{cacheRead: totals[dayKey(d)]}})
	}
	return out
}

// streak is how many days up to today have tokens, without a gap; today
// without any yet doesn't break it.
func streak(days []dayUsage) int {
	n := 0
	for i := len(days) - 1; i >= 0; i-- {
		if days[i].total() == 0 {
			if i == len(days)-1 {
				continue
			}
			break
		}
		n++
	}
	return n
}
