package ui

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/client"
	"github.com/hustlex/hustlex/proto"
)

// Before any project, an agent's install or sign-in session still shows,
// not the start screen.
func TestSessionWithoutProjectShows(t *testing.T) {
	a := New(Options{})
	a.client = &client.Client{}
	s := proto.SessionInfo{ID: "s1", Agent: "claude", Name: "Sign in to Claude Code", Dir: "/home/u"}
	a.sessions = []proto.SessionInfo{s}
	a.selectSession(s)
	tt := ui.NewTester(func(c *ui.Context) { a.view(c) }, 1300, 800)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("Sign in to Claude Code") || tt.HasText("Each folder is a project. Sessions run in it or in their own git worktrees.") {
		t.Fatalf("%q", tt.Texts())
	}
}
