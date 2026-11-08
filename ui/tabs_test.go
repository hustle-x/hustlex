package ui

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/proto"
)

// A session's tabs show in its strip, not in the session list, and
// picking one from elsewhere brings it to the front of its session.
func TestTabsShowInTheirSession(t *testing.T) {
	a := New(Options{})
	parent := proto.SessionInfo{ID: "s1", Project: "/code/app", Agent: "claude"}
	tab := proto.SessionInfo{ID: "s2", Project: "/code/app", Agent: "shell", Parent: "s1"}
	a.sessions = []proto.SessionInfo{parent, tab}
	a.agents = []proto.AgentInfo{{ID: "claude", Name: "Claude Code"}, {ID: "shell", Name: "Shell"}}

	if got := a.projectSessions("/code/app"); len(got) != 1 || got[0].ID != "s1" {
		t.Fatalf("project sessions %+v", got)
	}
	tt := ui.NewTester(func(c *ui.Context) { a.tabStrip(c, parent) }, 600, 40)
	tt.Frame()
	if !tt.HasText("Claude Code") || !tt.HasText("Shell") {
		t.Fatalf("strip %q", tt.Texts())
	}
	if a.frontTab(parent).ID != "s1" {
		t.Fatal("a tab in front before one was picked")
	}
	a.selectSession(tab)
	if a.selected != "s1" || a.frontTab(parent).ID != "s2" {
		t.Fatalf("selected %q, front %q", a.selected, a.frontTab(parent).ID)
	}
	a.remove("s2")
	if a.frontTab(parent).ID != "s1" {
		t.Fatal("a removed tab stays in front")
	}
}

// The split icon puts a session's second tab beside the front one; picking
// the one beside swaps them, and removing it goes back to one.
func TestTabsSplitBesideEachOther(t *testing.T) {
	a := New(Options{})
	parent := proto.SessionInfo{ID: "s1", Project: "/code/app", Agent: "claude"}
	tab := proto.SessionInfo{ID: "s2", Project: "/code/app", Agent: "shell", Parent: "s1"}
	a.agents = []proto.AgentInfo{{ID: "claude", Name: "Claude Code"}, {ID: "shell", Name: "Shell"}}

	a.sessions = []proto.SessionInfo{parent}
	tt := ui.NewTester(func(c *ui.Context) { a.tabStrip(c, parent) }, 600, 40)
	tt.Frame()
	if tt.HasText("Show two tabs side by side") {
		t.Fatal("a split button with a single tab")
	}

	a.sessions = []proto.SessionInfo{parent, tab}
	tt.Frame()
	a.toggleTabSplit(parent)
	if side, ok := a.sideTab(parent); !ok || side.ID != "s2" || a.frontTab(parent).ID != "s1" {
		t.Fatalf("side %+v %v, front %q", side, ok, a.frontTab(parent).ID)
	}
	a.showTab("s1", "s2")
	if side, ok := a.sideTab(parent); !ok || side.ID != "s1" || a.frontTab(parent).ID != "s2" {
		t.Fatalf("after picking the side tab: side %+v %v, front %q", side, ok, a.frontTab(parent).ID)
	}
	a.toggleTabSplit(parent)
	if _, ok := a.sideTab(parent); ok {
		t.Fatal("still split after turning it off")
	}
	a.toggleTabSplit(parent)
	a.remove("s2")
	if _, ok := a.sideTab(parent); ok {
		t.Fatal("a removed tab stays beside")
	}
}
