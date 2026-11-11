package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/client"
	"github.com/hustlex/hustlex/proto"
)

// Back after a while, the user sees what moved on: sessions done, asking,
// moved to another account, and the limits.
func TestAwaySummarySaysWhatMovedOn(t *testing.T) {
	a := New(Options{})
	a.client = &client.Client{}
	a.accts = accountsState{host: a.host.ID, asked: true, loaded: true} // nothing to load
	start := time.Now()
	account := func(used float64) proto.CreditsReport {
		return proto.CreditsReport{Accounts: []proto.CreditAccount{{Agent: "claude", Ref: proto.RefDefault, Account: "alex@example.com",
			Windows: []proto.CreditWindow{{Label: "5 hours", Used: used, ResetsAt: start.Add(2 * time.Hour)}}}}}
	}
	a.credits.SetReport(account(13))
	a.sessions = []proto.SessionInfo{
		{ID: "s1", Agent: "claude", Project: "/p", Name: "Add dark mode", State: proto.StateWorking},
		{ID: "s2", Agent: "codex", Project: "/p", Name: "Push notifications", State: proto.StateWorking},
		{ID: "s3", Agent: "claude", Project: "/p", Name: "Fix the login redirect", State: proto.StateWorking},
		{ID: "s4", Agent: "claude", Project: "/p", Name: "Untouched", State: proto.StateDone},
	}
	a.awayTick(0, start)
	a.awayTick(11*time.Minute, start.Add(11*time.Minute))
	if a.away.open {
		t.Fatal("the summary shows while the user is away")
	}

	a.sessions[0].State, a.sessions[0].Note = proto.StateDone, "6 files changed · 24 tests passed"
	a.sessions[1].State = proto.StateWaiting
	a.sessions[2].Moved = &proto.AccountMove{From: "", To: "work", Limit: "5-hour limit", At: start.Add(20 * time.Minute)}
	a.credits.SetReport(account(29))
	a.awayTick(5*time.Second, start.Add(47*time.Minute))
	if !a.away.open {
		t.Fatal("no summary on coming back")
	}
	var titles []string
	for _, r := range a.away.sum.rows {
		titles = append(titles, r.title+": "+r.detail)
	}
	got := strings.Join(titles, "\n")
	for _, want := range []string{
		"Add dark mode is done: 6 files changed · 24 tests passed",
		"Push notifications needs you: waits for your answer",
		"Fix the login redirect moved to work: alex@example.com ran out of its 5-hour limit",
		"Limits: alex 5-hour 13% → 29% · resets in",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in\n%s", want, got)
		}
	}
	if strings.Contains(got, "Untouched") || a.away.sum.moved != 3 {
		t.Errorf("%d sessions moved on:\n%s", a.away.sum.moved, got)
	}
	tt := ui.NewTester(func(c *ui.Context) { a.awayDialog(c) }, 900, 700)
	tt.Frame()
	tt.Frame()
	for _, want := range []string{"While you were away", "3 sessions moved on", "Review", "Answer", "Got it"} {
		if !tt.HasText(want) {
			t.Fatalf("no %q in %q", want, tt.Texts())
		}
	}

	// Away and back with nothing new: no summary.
	a.away.open = false
	a.awayTick(0, start.Add(50*time.Minute))
	a.awayTick(20*time.Minute, start.Add(70*time.Minute))
	a.awayTick(0, start.Add(71*time.Minute))
	if a.away.open {
		t.Fatalf("a summary of nothing: %+v", a.away.sum)
	}
}

func TestAwayLength(t *testing.T) {
	for d, want := range map[time.Duration]string{
		47 * time.Minute:  "47 minutes",
		61 * time.Minute:  "1 hour 1 minute",
		125 * time.Minute: "2 hours 5 minutes",
		3 * time.Hour:     "3 hours",
	} {
		if got := awayLength(d); got != want {
			t.Errorf("awayLength(%v) = %q, want %q", d, got, want)
		}
	}
}

// A session queued until a reset says when it starts and after what, and
// the form waits for the account's fullest limit.
func TestQueuedSessionSaysWhenItStarts(t *testing.T) {
	a := New(Options{})
	reset := time.Now().Add(2*time.Hour + 14*time.Minute).Truncate(time.Minute)
	a.credits.SetReport(proto.CreditsReport{Accounts: []proto.CreditAccount{{Agent: "claude", Ref: proto.RefDefault, Account: "alex@example.com",
		Windows: []proto.CreditWindow{{Label: "5 hours", Used: 97, ResetsAt: reset}, {Label: "Week", Used: 40, ResetsAt: reset.Add(72 * time.Hour)}}}}})
	_, w, ok := a.resetToWait("claude", "")
	if !ok || w.Label != "5 hours" {
		t.Fatalf("waits for %+v, %v", w, ok)
	}
	s := proto.SessionInfo{ID: "q", Agent: "claude", Project: "/p", Queued: true, StartAt: reset.Add(startMargin)}
	if !startsLater(s) {
		t.Fatal("not queued for later")
	}
	want := "starts " + clockOrDay(s.StartAt, time.Now()) + " · after alex's 5-hour reset"
	if got := a.startsDetail(s); got != want {
		t.Fatalf("startsDetail = %q, want %q", got, want)
	}
}
