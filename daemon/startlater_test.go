package daemon

import (
	"runtime"
	"testing"
	"time"

	"github.com/hustlex/hustlex/proto"
)

func TestSessionQueuedUntilAResetStartsThen(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	fakeAgent(t)
	dir := t.TempDir()
	srv, err := NewServer(dir)
	if err != nil {
		t.Fatal(err)
	}
	c := connect(t, srv, nil)
	at := time.Now().Add(4 * time.Second).Truncate(time.Second)
	held, err := c.CreateSession(proto.SessionCreateParams{Agent: "claude", Dir: t.TempDir(), Prompt: "fix the login redirect", StartAt: at, Cols: 80, Rows: 24})
	if err != nil {
		t.Fatal(err)
	}
	if !held.Queued || !held.StartAt.Equal(at) {
		t.Fatalf("queued %v until %v, want until %v", held.Queued, held.StartAt, at)
	}
	// Others don't wait behind it.
	now, err := c.CreateSession(proto.SessionCreateParams{Agent: "claude", Dir: t.TempDir(), Cols: 80, Rows: 24})
	if err != nil || now.Queued {
		t.Fatalf("a session for now: %+v, %v", now, err)
	}

	// It stays queued over a restart, then starts at its time.
	srv.Close()
	again, err := NewServer(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()
	info := func() proto.SessionInfo {
		sess, err := again.sessions.Get(held.ID)
		if err != nil {
			t.Fatal(err)
		}
		return sess.Info()
	}
	if got := info(); !got.Queued || !got.StartAt.Equal(at) || got.Prompt != "fix the login redirect" {
		t.Fatalf("after the restart: queued %v until %v, prompt %q", got.Queued, got.StartAt, got.Prompt)
	}
	waitFor(t, "it starts at its time", func() bool { return !info().Queued })
	if time.Now().Before(at) {
		t.Fatal("started before its time")
	}
	if got := info(); !got.StartAt.IsZero() {
		t.Fatalf("still says it starts at %v", got.StartAt)
	}
	waitFor(t, "its agent runs", func() bool { return info().Active })
}
