package daemon

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/hustlex/hustlex/core/git"
	"github.com/hustlex/hustlex/proto"
)

// An agent's turn, said by its hooks, can be undone and redone from the
// app: the files it changed go back, and forth.
func TestUndoAnAgentsTurn(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	fakeAgent(t)
	srv := startServer(t)
	c := connect(t, srv, nil)
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "t@example.com"}, {"config", "user.name", "t"}} {
		if _, err := git.Run(dir, args...); err != nil {
			t.Fatal(err)
		}
	}
	file := filepath.Join(dir, "app.go")
	os.WriteFile(file, []byte("v1"), 0o644)
	info, err := c.CreateSession(proto.SessionCreateParams{Agent: "claude", Dir: dir, Cols: 80, Rows: 24})
	if err != nil {
		t.Fatal(err)
	}
	state := func(st string) {
		if err := c.Call(proto.MethodSessionState, proto.SessionStateParams{ID: info.ID, State: st}, nil); err != nil {
			t.Fatal(err)
		}
	}
	turn := func() *proto.TurnChange {
		sess, _ := srv.sessions.Get(info.ID)
		return sess.Info().Turn
	}
	state(proto.StateWorking)
	waitFor(t, "the snapshot before the turn", func() bool {
		ts := srv.turns.of(info.ID)
		ts.mu.Lock()
		defer ts.mu.Unlock()
		return !ts.running && len(ts.queue) == 0 && ts.inTurn
	})
	os.WriteFile(file, []byte("v2"), 0o644)
	os.WriteFile(filepath.Join(dir, "new.go"), []byte("new"), 0o644)
	state(proto.StateDone)
	waitFor(t, "the turn's changes", func() bool { tc := turn(); return tc != nil && tc.Files == 2 })

	files, err := c.UndoTurn(info.ID, false)
	if err != nil || len(files) != 2 {
		t.Fatalf("undo: %v %v", files, err)
	}
	if b, _ := os.ReadFile(file); string(b) != "v1" {
		t.Fatalf("app.go = %q after undo", b)
	}
	if _, err := os.Stat(filepath.Join(dir, "new.go")); !os.IsNotExist(err) {
		t.Fatal("the new file stayed")
	}
	if tc := turn(); tc == nil || !tc.Undone {
		t.Fatalf("turn = %+v", tc)
	}
	if _, err := c.UndoTurn(info.ID, true); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(file); string(b) != "v2" {
		t.Fatalf("app.go = %q after redo", b)
	}
	if _, err := c.UndoTurn(info.ID, true); err == nil {
		t.Fatal("redid twice")
	}
}

// Two agents in one folder: a turn another agent worked during can't be
// undone, which would take the other's changes back too.
func TestNoUndoWhenAnotherAgentSharesTheFolder(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	fakeAgent(t)
	srv := startServer(t)
	c := connect(t, srv, nil)
	dir := t.TempDir()
	if _, err := git.Run(dir, "init", "-q"); err != nil {
		t.Fatal(err)
	}
	a, _ := c.CreateSession(proto.SessionCreateParams{Agent: "claude", Dir: dir, Cols: 80, Rows: 24})
	b, _ := c.CreateSession(proto.SessionCreateParams{Agent: "claude", Dir: dir, Cols: 80, Rows: 24})
	state := func(id, st string) {
		if err := c.Call(proto.MethodSessionState, proto.SessionStateParams{ID: id, State: st}, nil); err != nil {
			t.Fatal(err)
		}
	}
	state(a.ID, proto.StateWorking)
	waitFor(t, "a's snapshot", func() bool {
		ts := srv.turns.of(a.ID)
		ts.mu.Lock()
		defer ts.mu.Unlock()
		return !ts.running && len(ts.queue) == 0 && ts.inTurn
	})
	state(b.ID, proto.StateWorking)
	os.WriteFile(filepath.Join(dir, "b.go"), []byte("b's work"), 0o644)
	state(b.ID, proto.StateDone)
	state(a.ID, proto.StateDone)
	time.Sleep(time.Second)
	sess, _ := srv.sessions.Get(a.ID)
	if tc := sess.Info().Turn; tc != nil {
		t.Fatalf("a's turn can be undone: %+v", tc)
	}
	if _, err := c.UndoTurn(a.ID, false); err == nil {
		t.Fatal("undid a turn shared with another agent")
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "b.go")); string(b) != "b's work" {
		t.Fatal("b's work went")
	}
}
