package ui

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/client"
	"github.com/hustlex/hustlex/proto"
)

// A new user sees the steps a first session needs, Claude Code offered
// while no agent is installed and the others to pick, until the first
// session; Start needs every step done; Skip hides them.
func TestWelcomeLeadsToTheFirstSession(t *testing.T) {
	a := New(Options{})
	a.client = &client.Client{}
	claude := proto.AgentInfo{ID: "claude", Name: "Claude Code", Command: "claude", Install: "curl … | bash"}
	codex := proto.AgentInfo{ID: "codex", Name: "Codex", Command: "codex", Install: "npm i -g @openai/codex"}
	a.agents = []proto.AgentInfo{claude, codex, {ID: "shell", Name: "Shell", Installed: true}}
	frame := func() *ui.Tester {
		tt := ui.NewTester(func(c *ui.Context) { a.workspaceView(c) }, 1100, 700)
		tt.Frame()
		tt.Frame()
		return tt
	}
	tt := frame()
	for _, want := range []string{"Welcome to hustleX", "Install Claude Code", "Codex", "After step 1", "Open folder", "3 steps left"} {
		if !tt.HasText(want) {
			t.Fatalf("no %q in %q", want, tt.Texts())
		}
	}
	if tt.HasText("Start your first session") {
		t.Fatal("Start before the steps are done")
	}
	if err := tt.Click("Codex"); err != nil {
		t.Fatal(err)
	}
	if tt = frame(); !tt.HasText("Install Codex") || !tt.HasText("Opens your browser to sign in with your ChatGPT account.") {
		t.Fatalf("Codex picked: %q", tt.Texts())
	}
	a.welcomePick = ""
	if a.defaultAgent() != "claude" {
		t.Fatalf("a new session would start %q", a.defaultAgent())
	}

	claude.Installed = true
	a.agents[0] = claude
	a.setProjects([]proto.ProjectInfo{{Path: "/code/site", Name: "site"}})
	if tt = frame(); !tt.HasText("Checking for an account signed in already…") || tt.HasText("Sign in to Claude Code") {
		t.Fatalf("accounts not read yet: %q", tt.Texts())
	}
	a.credits.SetReport(proto.CreditsReport{})
	if tt = frame(); !tt.HasText("Sign in to Claude Code") || !tt.HasText("1 step left") || tt.HasText("Start your first session") {
		t.Fatalf("not signed in: %q", tt.Texts())
	}
	a.credits.SetReport(proto.CreditsReport{Accounts: []proto.CreditAccount{{Agent: "claude", Ref: proto.RefDefault, Account: "alex@example.com"}}})
	if tt = frame(); tt.HasText("Install Claude Code") || tt.HasText("Open folder") || !tt.HasText("Start your first session") {
		t.Fatalf("%q", tt.Texts())
	}

	a.sessions = []proto.SessionInfo{{ID: "s1", Project: "/code/site", Agent: "claude"}}
	if a.welcomeShown() {
		t.Fatal("still welcomed after the first session")
	}
	a.sessions = nil
	if err := frame().Click("Skip"); err != nil {
		t.Fatal(err)
	}
	if a.welcomeShown() {
		t.Fatal("still welcomed after Skip")
	}
}
