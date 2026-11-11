package ui

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/client"
	"github.com/hustlex/hustlex/proto"
)

// A new user picks a layout first; closing the picker keeps Expert, as
// before layouts.
func TestLayoutPickerAsksANewUser(t *testing.T) {
	a := New(Options{})
	a.client = &client.Client{}
	a.agents = []proto.AgentInfo{{ID: "claude", Name: "Claude Code", Command: "claude"}}
	frame := func() *ui.Tester {
		tt := ui.NewTester(func(c *ui.Context) { a.layoutPicker(c) }, 1100, 700)
		tt.Frame()
		tt.Frame()
		return tt
	}
	tt := frame()
	if !tt.HasText("Choose your layout") || !tt.HasText("Continue with Compact") {
		t.Fatalf("%q", tt.Texts())
	}
	if err := tt.Click("Expert"); err != nil {
		t.Fatal(err)
	}
	tt = frame()
	if err := tt.Click("Continue with Expert"); err != nil {
		t.Fatalf("%v: %q", err, tt.Texts())
	}
	if a.state.Layout != layoutExpert {
		t.Fatalf("layout %q", a.state.Layout)
	}
	if frame().HasText("Choose your layout") {
		t.Fatal("asked again")
	}
}

// Compact shows a session as a chat, or its terminal once picked, and
// leaves the other views, the side panel and most pages to ⌘K.
func TestCompactShowsLess(t *testing.T) {
	a := New(Options{})
	a.client = &client.Client{}
	s := proto.SessionInfo{ID: "s1", Agent: "claude", Project: "/p"}
	a.state.Layout = layoutCompact
	a.state.Panel.Open = true
	if views := a.sessionViews(s); slices.ContainsFunc(views, func(v string) bool { return v != viewTerminal && v != viewChat }) {
		t.Fatalf("views %q", views)
	}
	if a.panelShown() {
		t.Fatal("side panel in Compact")
	}
	if canChat(s) {
		a.compactChat(s)
		if a.currentView(s) != viewChat {
			t.Fatalf("view %q, want the chat", a.currentView(s))
		}
		a.showView(s, viewTerminal)
		a.compactChat(s)
		if a.currentView(s) != viewTerminal {
			t.Fatal("the picked terminal didn't stay")
		}
	}
	tt := ui.NewTester(func(c *ui.Context) { a.navBar(c) }, 1100, 60)
	tt.Frame()
	if tt.HasText("Canvas") || !tt.HasText("Notes") {
		t.Fatalf("nav %q", tt.Texts())
	}

	a.page = pageCanvas
	a.setWindowLayout(layoutExpert)
	if a.page != pageCanvas {
		t.Fatal("Expert lost the page")
	}
	a.setWindowLayout(layoutCompact)
	if a.page != "" {
		t.Fatalf("Compact kept page %q", a.page)
	}
	a.setWindowLayout(layoutExpert)
	if !a.panelShown() {
		t.Fatal("no side panel in Expert")
	}
}
