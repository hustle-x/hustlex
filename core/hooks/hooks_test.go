package hooks

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/hustlex/hustlex/proto"
)

func TestCheckRefusesWhatCannotRun(t *testing.T) {
	for _, bad := range [][]proto.ProjectHook{
		{{Event: "finished", Run: "x"}},
		{{Event: EventDone}},
		{{Event: EventDone, Webhook: "ftp://x"}},
	} {
		if Check(bad) == nil {
			t.Errorf("%+v passed", bad)
		}
	}
	if err := Check([]proto.ProjectHook{{Event: EventDone, Run: "echo", Webhook: "https://example.com/h"}}); err != nil {
		t.Error(err)
	}
}

func TestSaveLoadAndAllowed(t *testing.T) {
	dir := t.TempDir()
	want := []proto.ProjectHook{{Event: EventInput, Run: "say hi"}}
	if err := Save(dir, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil || len(got) != 1 || got[0] != want[0] {
		t.Fatalf("%+v, %v", got, err)
	}
	a := OpenAllowed(t.TempDir())
	if a.Is(dir) {
		t.Error("allowed before the user said so")
	}
	a.Set(dir, true)
	if !a.Is(dir) {
		t.Error("not allowed after")
	}
	a.Set(dir, false)
	if a.Is(dir) {
		t.Error("still allowed")
	}
}

func TestTrackerFiresOncePerChange(t *testing.T) {
	var tr Tracker
	now := time.Now()
	s := proto.SessionInfo{ID: "a", Created: now}
	steps := []struct {
		mut  func(*proto.SessionInfo)
		want string
	}{
		{func(*proto.SessionInfo) {}, EventStart},
		{func(*proto.SessionInfo) {}, ""},
		{func(i *proto.SessionInfo) { i.State = proto.StateWaiting }, EventInput},
		{func(i *proto.SessionInfo) { i.State = proto.StateWaiting }, ""},
		{func(i *proto.SessionInfo) { i.State = proto.StateDone }, EventDone},
		{func(i *proto.SessionInfo) { i.Exited, i.ExitCode = true, 2 }, EventError},
	}
	for i, st := range steps {
		st.mut(&s)
		if got := tr.Event(s, now); got != st.want {
			t.Errorf("step %d: %q, want %q", i, got, st.want)
		}
	}
	old := proto.SessionInfo{ID: "b", Created: now.Add(-time.Hour)}
	if tr.Event(old, now) != "" {
		t.Error("a session running before the host started counted as a start")
	}
}

func TestRunHookGetsTheSessionInItsEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the hook is a sh command")
	}
	dir := t.TempDir()
	out := filepath.Join(dir, "out")
	done := make(chan struct{})
	Fire([]proto.ProjectHook{{Event: EventDone, Run: "echo $HUSTLEX_EVENT $HUSTLEX_SESSION > out"}}, EventDone,
		proto.SessionInfo{ID: "s1", Dir: dir}, func(string, ...any) {})
	go func() {
		for range 100 {
			if b, _ := os.ReadFile(out); len(b) > 0 {
				close(done)
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()
	select {
	case <-done:
	case <-time.After(6 * time.Second):
		t.Fatal("the hook did not run")
	}
	if b, _ := os.ReadFile(out); string(b) != "done s1\n" {
		t.Errorf("got %q", b)
	}
}
