package ui

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/ui/theme"
)

type fakeWindow struct{ did []string }

func (w *fakeWindow) Close()          { w.did = append(w.did, "close") }
func (w *fakeWindow) Minimize()       { w.did = append(w.did, "minimize") }
func (w *fakeWindow) ToggleMaximize() { w.did = append(w.did, "maximize") }

// The title bar's own buttons close, minimize and maximize the window:
// macOS's dots, and Windows' and Linux's −, □ and ×.
func TestWindowButtons(t *testing.T) {
	for _, mac := range []bool{true, false} {
		was := macWindowButtons
		macWindowButtons = mac
		testWindowButtons(t, map[bool]string{true: "title-bar.png", false: "title-bar-windows.png"}[mac])
		macWindowButtons = was
	}
}

func testWindowButtons(t *testing.T, name string) {
	a := New(Options{})
	w := &fakeWindow{}
	a.winCtl = w
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(900).Height(60).Background(theme.Ground).Children(func() { a.titleBar(c) })
	}, 900, 60)
	tt.Frame()
	for _, b := range []string{"Zoom", "Minimize", "Close"} {
		if err := tt.Click(b); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
	}
	shot(t, tt, name)
	if got := w.did; len(got) != 3 || got[0] != "maximize" || got[1] != "minimize" || got[2] != "close" {
		t.Fatalf("did %q", got)
	}
}
