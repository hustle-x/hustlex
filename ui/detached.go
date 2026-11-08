package ui

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/proto"
	"github.com/hustlex/hustlex/ui/icon"
	"github.com/hustlex/hustlex/ui/theme"
	"github.com/hustlex/hustlex/ui/widget"
)

// A session can leave the main window for one of its own, dragged out of
// the Sessions list or from its menu: the window shows the session's
// terminal under a small header. The session is the same one; closing the
// window only brings it back to the main window, it ends nothing.

// sessionDrag is what a session row carries while dragged.
type sessionDrag struct{ id string }

// openSessionWindow shows session s in a window of its own, at the screen
// point at when given (the pointer let go there), or focuses the one it has.
func (a *App) openSessionWindow(s proto.SessionInfo, at *mygo.Point) {
	if w := a.detached[s.ID]; w != nil {
		w.Show()
		w.Focus()
		return
	}
	if a.detached == nil {
		a.detached = map[string]*mygo.Window{}
	}
	id := s.ID
	o := mygo.WindowOptions{
		Title:           a.sessionName(s),
		Width:           900,
		Height:          600,
		MinWidth:        480,
		MinHeight:       320,
		BackgroundColor: theme.Hex(theme.Background),
		Content:         ui.View(func(c *ui.Context) { a.detachedView(c, id) }),
	}
	if at != nil {
		// The title bar under the pointer, as a tab torn off a browser.
		o.X, o.Y = max(at.X-120, 0), max(at.Y-16, 0)
	}
	w := mygo.NewWindow(o)
	w.OnClosed(func() {
		if a.detached[id] == w {
			delete(a.detached, id)
		}
	})
	a.detached[id] = w
}

// closeSessionWindow brings session id back to the main window.
func (a *App) closeSessionWindow(id string) {
	if w := a.detached[id]; w != nil {
		delete(a.detached, id)
		w.Close()
	}
}

// detachedView is the window of one detached session.
func (a *App) detachedView(c *ui.Context, id string) {
	c.SetTheme(a.theme)
	s, ok := a.sessionByID(id)
	p := a.panes[id]
	if !ok {
		// Deleted in the main window.
		a.onMain(func() { a.closeSessionWindow(id) })
	}
	ui.Column(c).Fill().Background(theme.Background).TextColor(theme.Text).Children(func() {
		ui.Row(c).Height(headerHeight).Shrink(0).FillWidth().PaddingX(14).Gap(8).AlignItems(ui.Center).
			BorderWidth(0, 0, 1, 0).BorderColor(theme.Border).Background(theme.Panel).Children(func() {
			if !ok {
				return
			}
			st := statusOf(s)
			widget.Dot(c, st.color, st.hollow, 7)
			ui.Text(c, a.sessionName(s)).FontSize(12.5).FontWeight(600).SingleLine().Shrink(1)
			ui.Text(c, st.label).FontSize(11.5).TextColor(theme.Muted).SingleLine().Shrink(0)
			ui.Spacer(c)
			if widget.GhostButton(c, "Bring back").Clicked() {
				a.closeSessionWindow(id)
			}
		})
		if p != nil {
			terminal.View(c.Key("term"), p.term).Grow(1).FillWidth()
			return
		}
		ui.Column(c).Grow(1).FillWidth().Center().Children(func() {
			ui.Text(c, "Connecting…").TextColor(theme.Muted)
		})
	})
}

// detachedNote stands where a session's terminal was, while it shows in
// its own window.
func (a *App) detachedNote(c *ui.Context, s proto.SessionInfo) {
	ui.Column(c).Grow(1).FillWidth().Center().Gap(12).Children(func() {
		ui.Icon(c, icon.AppWindow).FontSize(26).TextColor(theme.Muted)
		ui.Text(c, "This session is open in its own window.").TextColor(theme.Muted)
		ui.Row(c).Gap(8).Children(func() {
			if widget.Button(c, widget.Secondary, nil, "Show window", "").Clicked() {
				a.openSessionWindow(s, nil)
			}
			if widget.Button(c, widget.Secondary, nil, "Bring back", "").Clicked() {
				a.closeSessionWindow(s.ID)
			}
		})
	})
}

// dropOutsideWindow opens s in a window where the pointer let go if that is
// outside the main window.
func (a *App) dropOutsideWindow(s proto.SessionInfo) {
	pt := mygo.Screen.CursorScreenPoint()
	b := a.win.Bounds()
	if pt.X >= b.X && pt.X < b.X+b.Width && pt.Y >= b.Y && pt.Y < b.Y+b.Height {
		return
	}
	a.openSessionWindow(s, &pt)
}
