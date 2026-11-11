package daemon

import (
	"runtime"
	"slices"
	"testing"

	"github.com/hustlex/hustlex/proto"
)

func TestAgentsNameSessionsUntilTheUserDoes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	dir := t.TempDir()
	srv, err := NewServer(dir)
	if err != nil {
		t.Fatal(err)
	}
	c := connect(t, srv, nil)
	info, err := c.CreateSession(proto.SessionCreateParams{Command: []string{"sh", "-c", "sleep 30"}, Dir: t.TempDir(), Cols: 80, Rows: 10})
	if err != nil {
		t.Fatal(err)
	}
	for _, title := range []string{"Fix login", "  Fix the login   redirect "} {
		if err := c.SetTitle(info.ID, title, false); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.SetTitle(info.ID, "Login bug", true); err != nil {
		t.Fatal(err)
	}
	if err := c.SetTitle(info.ID, "Something else", false); err == nil {
		t.Fatal("an agent renamed a session the user named")
	}
	srv.Close()
	again, err := NewServer(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()
	list, _ := connect(t, again, nil).Sessions()
	if len(list) != 1 || list[0].Name != "Login bug" || !list[0].Named ||
		!slices.Equal(list[0].Earlier, []string{"Fix login", "Fix the login redirect"}) {
		t.Fatalf("after the restart: %+v", list)
	}
}
