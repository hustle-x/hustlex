package ui

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/client"
	"github.com/hustlex/hustlex/proto"
	"github.com/hustlex/hustlex/ui/icon"
	"github.com/hustlex/hustlex/ui/theme"
	"github.com/hustlex/hustlex/ui/widget"
)

// Agents name their sessions (hustlex title self); a name the user gives,
// by double-clicking it in the header, stays theirs.

// sessionRename is the header's name being edited, on the main thread.
type sessionRename struct {
	id      string // the session, "" while none is edited
	buf     string
	focused bool // the field had the focus, so losing it saves
}

// movedAt is when the daemon last moved s to another account, zero when
// it never did.
func movedAt(s proto.SessionInfo) int64 {
	if s.Moved == nil {
		return 0
	}
	return s.Moved.At.UnixNano()
}

// sessionCrumb is the header's path to session s, its name last: double
// click to rename it. narrow shows the name alone.
func (a *App) sessionCrumb(c *ui.Context, s proto.SessionInfo, narrow bool, parts ...string) {
	if a.rename.id != "" && a.rename.id != s.ID {
		a.rename = sessionRename{} // another session shows: the edit is dropped
	}
	ui.Row(c).Gap(7).MinWidth(0).Shrink(1).AlignItems(ui.Center).Children(func() {
		if !narrow {
			// The parts before the name give way first: it says the most.
			for _, p := range parts {
				ui.Text(c, p).FontSize(12).TextColor(theme.Muted).SingleLine().MinWidth(0).Shrink(20)
				ui.Text(c, "/").FontSize(12).TextColor(theme.Idle)
			}
		}
		if a.rename.id == s.ID {
			a.renameField(c, s)
			return
		}
		name := a.sessionName(s)
		tip := "Double-click to rename"
		if len(s.Earlier) > 0 {
			tip = "Earlier: " + strings.Join(s.Earlier, " → ") + "\n" + tip
		}
		t := ui.ButtonBase(c.Key("title")).Gap(5).MinWidth(0).Shrink(1).AlignItems(ui.Center).Label(name).Tooltip(tip)
		t.Children(func() {
			ui.Text(c, name).FontSize(12).TextColor(theme.Text).SingleLine().Shrink(1)
			if s.Named {
				ui.Icon(c, icon.Pin).FontSize(10).TextColor(theme.Muted)
			}
		})
		if t.DoubleClicked() {
			a.rename = sessionRename{id: s.ID, buf: name}
		}
	})
}

// renameField edits session s's name: Enter or leaving it saves, Escape
// leaves it as it was.
func (a *App) renameField(c *ui.Context, s proto.SessionInfo) {
	in := widget.Input(c.Key("rename"), &a.rename.buf, "Session name").Height(24).Width(260).Shrink(1).AutoFocus()
	switch {
	case c.Shortcut(0, ui.KeyEscape):
		a.rename = sessionRename{}
	case in.Submitted(), a.rename.focused && !in.Focused():
		name := a.rename.buf
		a.rename = sessionRename{}
		a.renameSession(s, name)
	default:
		a.rename.focused = a.rename.focused || in.Focused()
	}
}

// renameSession gives session s the user's name; "" gives naming back
// to its agent.
func (a *App) renameSession(s proto.SessionInfo, name string) {
	name = strings.Join(strings.Fields(name), " ")
	cl := a.client
	if cl == nil || name == a.sessionName(s) || name == "" && !s.Named {
		return // unchanged, or emptied with nothing to give back
	}
	a.setNameHere(s.ID, name, name != "")
	go func() {
		err := cl.SetTitle(s.ID, name, true)
		a.onMain(func() {
			if err != nil {
				a.setNameHere(s.ID, s.Name, s.Named)
				a.status = "Couldn't rename the session: " + client.Explain(err)
				return
			}
			if name == "" {
				a.notify("Agents can name this session again.")
			} else {
				a.notify("Renamed. Agents won't change this title.")
			}
		})
	}()
}

// setNameHere shows session id under name until the daemon says.
func (a *App) setNameHere(id, name string, named bool) {
	for i := range a.sessions {
		if a.sessions[i].ID == id {
			a.sessions[i].Name, a.sessions[i].Named = name, named
		}
	}
}
