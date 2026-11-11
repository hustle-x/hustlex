package creditview

import (
	"strings"
	"testing"
	"time"

	"github.com/hustlex/hustlex/proto"
)

// The range counts its days against the range before, by account; each
// project, session and model is ranked by tokens.
func TestSpendSumsTheRange(t *testing.T) {
	today := time.Date(2026, 11, 11, 15, 0, 0, 0, time.Local)
	day := func(back int) string { return dayKey(today.AddDate(0, 0, -back)) }
	h := proto.UsageHistory{
		Conversations: []proto.UsageConversation{
			{Agent: "claude", Account: proto.RefDefault, Project: "/code/acme-web", Session: "s1"},
			{Agent: "claude", Account: "acc2", Project: "/code/acme-web", Title: "Add dark mode"},
			{Agent: "codex", Account: proto.RefDefault, Project: "/code/mobile-app", Title: "Push notifications"},
		},
		Rows: []proto.UsageRow{
			{Day: day(0), Conversation: 0, Model: "claude-opus-5-5", Input: 1e6, Output: 1e6},
			{Day: day(3), Conversation: 1, Model: "claude-opus-5-5", CacheRead: 10e6},
			{Day: day(1), Conversation: 2, Model: "gpt-5.5-codex", Input: 5e5},
			{Day: day(9), Conversation: 0, Model: "claude-opus-5-5", Input: 1e6},
			{Day: day(40), Conversation: 0, Model: "claude-opus-5-5", Input: 1e6},
		},
	}
	n := names{project: func(p string) string { return p[strings.LastIndex(p, "/")+1:] },
		session: func(id string) (string, bool) { return "Fix login redirect", id == "s1" }}
	s := spend(h, today, 7, "", n)
	if s.total.total() != 12.5e6 || s.prev.total() != 1e6 || s.conversations != 3 || s.prevConvs != 1 || s.active != 3 {
		t.Fatalf("%+v", s)
	}
	// 1M input and 1M output of Opus 5.5 ($4 + $20), 10M cache read ($2); Codex unpriced.
	if got := dollars(s.total.cost); got != "$26.00" || s.total.unpriced != 5e5 {
		t.Fatalf("cost %s, unpriced %d", got, s.total.unpriced)
	}
	if len(s.chart) != 14 || s.chart[13].total() != 2e6 {
		t.Fatalf("chart %d days, today %d", len(s.chart), s.chart[13].total())
	}
	if s.byProject[0].name != "acme-web" || s.bySession[0].name != "Add dark mode" || s.bySession[1].name != "Fix login redirect" || s.byMod[0].key != "claude-opus-5-5" {
		t.Fatalf("%+v %+v %+v", s.byProject, s.bySession, s.byMod)
	}
	if mine := spend(h, today, 7, accountKey("claude", "acc2"), n); mine.total.total() != 10e6 || mine.conversations != 1 {
		t.Fatalf("one account: %+v", mine.total)
	}
	if w := spend(h, today, 1, "", n); w.total.total() != 2e6 || len(w.chart) != 14 {
		t.Fatalf("today: %+v", w.total)
	}
}

// The pace says whether the week lasts, and when it runs out if not.
func TestPace(t *testing.T) {
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.Local) // a Monday
	week := func(used float64, left time.Duration) proto.CreditAccount {
		return proto.CreditAccount{Windows: []proto.CreditWindow{{Label: "5 hours", Used: 90, ResetsAt: now.Add(time.Hour)}, {Label: "Week", Used: used, ResetsAt: now.Add(left)}}}
	}
	if ok, text, known := pace(week(20, 5*24*time.Hour), now); !known || !ok || text != "On track: about 70% of the week by its reset" {
		t.Fatal(ok, text, known)
	}
	if ok, text, _ := pace(week(60, 5*24*time.Hour), now); ok || text != "At this pace the week runs out Tuesday evening" {
		t.Fatal(ok, text)
	}
	if _, _, known := pace(proto.CreditAccount{}, now); known {
		t.Fatal("no limits, yet a pace")
	}
}
