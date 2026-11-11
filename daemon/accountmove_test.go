package daemon

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hustlex/hustlex/core/agent"
	"github.com/hustlex/hustlex/proto"
)

func TestPickMoveGoesToTheAccountWithMostRoom(t *testing.T) {
	week := func(used float64) []proto.CreditWindow {
		return []proto.CreditWindow{{Label: "5 hours", Used: used}, {Label: "Week", Used: 10}}
	}
	accounts := []proto.CreditAccount{
		{Agent: "claude", Ref: proto.RefDefault, Account: "alex@example.com", Windows: week(100)},
		{Agent: "claude", Ref: "work", Account: "sam@example.com", Windows: week(60)},
		{Agent: "claude", Ref: "spare", Account: "kim@example.com", Windows: week(20)},
		{Agent: "claude", Ref: "broken", Error: "signed out"},
		{Agent: "claude", Account: "unmanaged@example.com", Windows: week(0)},
		{Agent: "codex", Ref: proto.RefDefault, Windows: week(0)},
	}
	from, to, limit, ok := pickMove(accounts, "claude", proto.RefDefault)
	if !ok || from.Account != "alex@example.com" || to.Ref != "spare" || limit != "5-hour limit" {
		t.Fatalf("pickMove = %v %v %q %v", from.Account, to.Ref, limit, ok)
	}
	if _, _, limit, ok := pickMove(accounts, "claude", "work"); ok || limit != "" {
		t.Fatalf("moved an account with room left: %q %v", limit, ok)
	}
	accounts[2].Windows = week(90)
	accounts[1].Windows = week(85)
	if _, _, limit, ok := pickMove(accounts, "claude", proto.RefDefault); ok || limit == "" {
		t.Fatalf("moved to a nearly full account: %q %v", limit, ok)
	}
}

func TestMovedSessionGoesOnInPlace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	bin := t.TempDir()
	// A "claude" that says its limit ran out, then echoes what it's told.
	script := "#!/bin/sh\necho '5-hour limit reached ∙ resets 6:50pm'\nwhile read l; do echo \"told: $l\"; done\n"
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	srv := startServer(t)
	c := connect(t, srv, nil)
	info, err := c.CreateSession(proto.SessionCreateParams{Agent: "claude", Dir: t.TempDir(), Cols: 120, Rows: 30})
	if err != nil {
		t.Fatal(err)
	}
	sess, _ := srv.sessions.Get(info.ID)
	waitFor(t, "it says its limit ran out", func() bool { return agent.LimitReached(sess.Terminal().Text()) })
	srv.persist.setConversation(info.ID, "conv-1")
	from := proto.CreditAccount{Agent: "claude", Ref: "work", Account: "alex@example.com"}
	to := proto.CreditAccount{Agent: "claude", Ref: "", Account: "sam@example.com"}
	if err := srv.moveAccount(sess, to, from, "5-hour limit"); err != nil {
		t.Fatal(err)
	}
	next, err := srv.sessions.Get(info.ID)
	if err != nil || next == sess {
		t.Fatalf("not replaced in place: %v", err)
	}
	got := next.Info()
	if got.Moved == nil || got.Moved.Limit != "5-hour limit" || got.Moved.To != "" {
		t.Fatalf("moved = %+v", got.Moved)
	}
	waitFor(t, "the notice and the agent told to go on", func() bool {
		text := next.Terminal().Text()
		return strings.Contains(text, "alex@example.com hit its 5-hour limit. Continued on sam@example.com") && strings.Contains(text, "told: Your usage limit ran out")
	})
	if list, _ := c.Sessions(); len(list) != 1 {
		t.Fatalf("sessions = %d", len(list))
	}
}
