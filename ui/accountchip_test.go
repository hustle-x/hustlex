package ui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/proto"
)

// The account chip outlined is the one the session in view runs as; on
// another page, or a session of another agent, there is none to follow.
func TestChipFollowsTheSessionInView(t *testing.T) {
	a := New(Options{})
	a.sessions = []proto.SessionInfo{
		{ID: "s1", Agent: "claude", Project: "/p"},
		{ID: "s2", Agent: "claude", Project: "/p", Account: "acc2"},
		{ID: "s3", Agent: "codex", Project: "/p"},
	}
	for _, c := range []struct {
		selected, page, want string
		ok                   bool
	}{
		{"s1", "", proto.RefDefault, true},
		{"s2", "", "acc2", true},
		{"s3", "", "", false},
		{"s2", pageCredits, "", false},
	} {
		a.selected, a.page = c.selected, c.page
		if ref, ok := a.showingRef("claude"); ref != c.want || ok != c.ok {
			t.Errorf("%s on %q: %q %v, want %q %v", c.selected, c.page, ref, ok, c.want, c.ok)
		}
	}
}

// Under a session, its account and what is left of it, with a way to go
// on as the agent's other account.
func TestAccountBarSaysTheSessionsAccount(t *testing.T) {
	a := New(Options{})
	a.credits.SetReport(proto.CreditsReport{Accounts: []proto.CreditAccount{
		{Agent: "claude", Ref: proto.RefDefault, Account: "alex@example.com", Plan: "Max 20x", Windows: []proto.CreditWindow{{Label: "5 hours", Used: 13}}},
		{Agent: "claude", Ref: "acc2", Account: "sam@example.com", Plan: "Pro", Windows: []proto.CreditWindow{{Label: "5 hours", Used: 71}}},
	}})
	s := proto.SessionInfo{ID: "s2", Agent: "claude", Project: "/p", Account: "acc2"}
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Width(700).Children(func() { a.accountBar(c, s) }) }, 700, 40)
	tt.Frame()
	for _, want := range []string{"This session uses", "sam@example.com", "Pro", "5h 71%", "Switch account"} {
		if !tt.HasText(want) {
			t.Fatalf("no %q in %q", want, tt.Texts())
		}
	}
	// A codex session: hustleX reads none of its accounts here, so no bar.
	s = proto.SessionInfo{ID: "s3", Agent: "codex", Project: "/p"}
	tt = ui.NewTester(func(c *ui.Context) { ui.Column(c).Width(700).Children(func() { a.accountBar(c, s) }) }, 700, 40)
	tt.Frame()
	if tt.HasText("This session uses") {
		t.Fatal("a bar for an account hustleX doesn't read")
	}
}

// A session whose agent asks something rises to Needs you, above the
// projects, and leaves its project's rows.
func TestNeedsYouComesFirst(t *testing.T) {
	a := New(Options{})
	a.projects = []proto.ProjectInfo{{Path: "/p", Name: "acme-web"}}
	a.sessions = []proto.SessionInfo{
		{ID: "s1", Agent: "claude", Project: "/p", Name: "Fix login", State: proto.StateWorking},
		{ID: "s2", Agent: "claude", Project: "/p", Name: "Push notifications", State: proto.StateWaiting},
	}
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Width(300).Height(500).Children(func() { a.sessionList(c) }) }, 300, 500)
	tt.Frame()
	tt.Frame()
	texts := strings.Join(tt.Texts(), "|")
	need, push, proj := strings.Index(texts, "NEEDS YOU"), strings.Index(texts, "Push notifications"), strings.Index(texts, "ACME-WEB")
	if need < 0 || push < need || proj < push || strings.Count(texts, "Push notifications") > 2 {
		t.Fatalf("order: %s", texts)
	}
}
