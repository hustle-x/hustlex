package ui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/client"
	"github.com/hustlex/hustlex/core/screenmenu"
	"github.com/hustlex/hustlex/proto"
)

// The Inbox shows each waiting session's question and choices, the
// longest waiting first, and the nav bar counts them.
func TestInboxShowsWhatAgentsAsk(t *testing.T) {
	a := New(Options{})
	a.client = &client.Client{}
	a.inbox.polling = new(int) // no poller: the menus are set here
	a.sessions = []proto.SessionInfo{
		{ID: "s1", Agent: "codex", Project: "/p", Title: "Push notifications", State: proto.StateWaiting},
		{ID: "s2", Agent: "claude", Project: "/p", Title: "Refactor settings screen", State: proto.StateWaiting},
		{ID: "s3", Agent: "claude", Project: "/p", Title: "Add dark mode", State: proto.StateWorking},
	}
	a.waiting()
	a.inbox.since["s2"] = time.Now().Add(-12 * time.Minute)
	a.inbox.menus = map[string]inboxMenu{
		"s1": {screenmenu.Menu{Question: "$ npm install firebase-admin", Choices: []screenmenu.Choice{{Label: "Yes, proceed", Key: "y"}, {Label: "No", Key: "\x1b"}}}, true},
		"s2": {screenmenu.Menu{Question: "Keep the old settings route as a redirect?", Choices: []screenmenu.Choice{{Label: "Yes, keep a redirect", Key: "1"}}}, true},
	}
	if w := a.waiting(); len(w) != 2 || w[0].ID != "s2" {
		t.Fatalf("%+v", w)
	}
	a.page = pageInbox
	tt := ui.NewTester(func(c *ui.Context) { a.inboxPage(c) }, 900, 700)
	tt.Frame()
	tt.Frame()
	for _, want := range []string{"Keep the old settings route as a redirect?", "Yes, keep a redirect", "$ npm install firebase-admin", "Yes, proceed", "Open session", "12m ago"} {
		if !tt.HasText(want) {
			t.Fatalf("no %q in %q", want, tt.Texts())
		}
	}
	nav := ui.NewTester(func(c *ui.Context) { a.navBar(c) }, 1100, 60)
	nav.Frame()
	if !nav.HasText("Inbox") || !nav.HasText("2") {
		t.Fatalf("nav %q", nav.Texts())
	}
	a.inbox.answered = map[string]time.Time{"s1": time.Now()}
	if w := a.waiting(); len(w) != 1 {
		t.Fatal("an answered question still waits")
	}
}
