package ui

import (
	"testing"

	"github.com/hustlex/hustlex/proto"
)

// A path a program prints opens in a panel tab of its session, named as
// the Files tool names it.
func TestLinksOpenFilesInThePanel(t *testing.T) {
	a := New(Options{})
	a.daemonInfo.Home = "/home/hustlex"
	parent := proto.SessionInfo{ID: "s1", Dir: "/home/hustlex/test"}
	tab := proto.SessionInfo{ID: "s2", Dir: "/home/hustlex/test", Parent: "s1"}
	a.sessions = []proto.SessionInfo{parent, tab}

	for link, want := range map[string]string{
		"/home/hustlex/test/my-app/main.go:12:3": "my-app/main.go",
		"./my-app/index.html":                    "my-app/index.html",
		"~/notes/todo.md":                        "/home/hustlex/notes/todo.md",
		"/tmp/shots/08-fixed.png":                "/tmp/shots/08-fixed.png",
		// As a chat's Markdown links them.
		"sample-web/index.html":               "sample-web/index.html",
		"file:///home/hustlex/test/web/a.css": "web/a.css",
	} {
		if !a.openLink("s2", link) {
			t.Fatalf("%s: not opened", link)
		}
		if a.selected != "s1" || a.panelTab() != fileTabPrefix+want {
			t.Fatalf("%s: selected %q, panel %q, want tab %q", link, a.selected, a.panelTab(), want)
		}
	}
	if a.openLink("s1", "https://example.com/") {
		t.Fatal("a web page opened in the app")
	}
}
