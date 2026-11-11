package ui

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/proto"
	"github.com/hustlex/hustlex/ui/icon"
	"github.com/hustlex/hustlex/ui/textfmt"
	"github.com/hustlex/hustlex/ui/theme"
	"github.com/hustlex/hustlex/ui/widget"
)

// The Sessions panel's width: as wide as the user dragged it, within these.
const (
	defaultSessionsWidth = 252
	minSessionsWidth     = 200
	maxSessionsWidth     = 460
	sessionsGrip         = 6 // the strip along its edge by the workspace that resizes it
)

// sessionsWidth is the Sessions panel's width.
func (a *App) sessionsWidth() float32 {
	if w := a.state.SessionsWidth; w > 0 {
		return min(max(w, minSessionsWidth), maxSessionsWidth)
	}
	return defaultSessionsWidth
}

// sessionList is the column of the host's projects, each with its
// sessions: on the left, or on the right past the tools
// (state.SessionsRight). It shows unless the user folded it away
// (toggleSessionsPanel) or a session took the window.
func (a *App) sessionList(c *ui.Context) {
	width := a.sessionsWidth()
	slide := float32(24) // in from the side it is on
	if a.state.sessionsLeft() {
		slide = -24
	}
	widget.Bento(ui.Column(c.Key("sessions")).Width(width).Shrink(0).FillHeight(), theme.Panel).
		Transition(widget.PanelMotion(slide)).Children(func() {
		// A header as tall as the workspace's and the tools', so their
		// lines meet across the window.
		ui.Row(c).FillWidth().Height(headerHeight).Shrink(0).PaddingX(panelInset).Gap(6).
			BorderWidth(0, 0, 1, 0).BorderColor(theme.Border).Children(func() {
			ui.Row(c).Gap(6).Grow(1).MinWidth(0).AlignItems(ui.Baseline).Children(func() {
				ui.Text(c, "Sessions").FontSize(13).FontWeight(600).SingleLine().Shrink(0)
				_, state := a.connectionState()
				if a.client != nil {
					state = fmt.Sprintf("%d %s", len(a.sessions), textfmt.Plural(len(a.sessions), "session", "sessions"))
				}
				ui.Text(c, state).FontSize(11.5).TextColor(theme.Muted).SingleLine().Shrink(1)
			})
			add := widget.IconButton(c, icon.FolderPlus, "Add a project: open a folder or clone from Git", false)
			a.menu.OpenBelow(add, "add-project", 28)
			a.menu.Show(c, add, "add-project", func(done func()) {
				if widget.MenuItem(c, icon.FolderPlus, "Open folder… ("+modKey+"O)", false) {
					a.openFolder()
					done()
				}
				if widget.MenuItem(c, icon.Git, "Clone from Git…", false) {
					a.openClone()
					done()
				}
			})
			p, ok := a.currentProject()
			b := widget.Button(c, widget.Primary, icon.Plus, "New", "").Height(28).PaddingX(10).Tooltip("New session (" + modKey + "N)")
			if b.Clicked() {
				if ok {
					a.openNewSession(p)
				} else {
					a.openFolder() // no project yet: pick one first
				}
			}
		})
		ui.Column(c).FillWidth().Shrink(0).Padding(panelInset, panelInset, 4, panelInset).Children(func() {
			widget.SearchField(c, &a.filter, "Filter sessions", icon.Search)
		})

		ui.Scroll(c).Grow(1).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Padding(8).Gap(2).Children(func() {
				if len(a.projects) == 0 {
					msg := "No projects yet. Open a folder to start."
					if a.client == nil {
						msg = "Not connected."
					}
					ui.Text(c, msg).FontSize(12).TextColor(theme.Muted).Padding(6)
				}
				shown := 0
				a.queuedLater(c)
				a.needsYou(c)
				for _, e := range a.projectEntries() {
					if e.group != "" {
						shown += a.groupSection(c.Key("group:"+e.group), e)
					} else if a.projectGroup(c.Key(e.projects[0].Path), e.projects[0]) {
						shown++
					}
				}
				if a.filter != "" && shown == 0 && len(a.projects) > 0 {
					ui.Text(c, "Nothing matches.").FontSize(12).TextColor(theme.Muted).Padding(6)
				}
				a.otherHostsSessions(c)
			})
		})
		a.sessionsGrip(c, width)
	})
}

// sessionsGrip is the Sessions panel's edge by the workspace, which the
// pointer drags to resize it; it shows a line only while hovered or
// dragged, and a double click puts the width back.
func (a *App) sessionsGrip(c *ui.Context, width float32) {
	grip := ui.Box(c.Key("sessions-grip")).Absolute().Top(0).Bottom(0).Width(sessionsGrip).Cursor(ui.CursorResizeEW).Label("Resize the Sessions panel")
	dir := float32(-1) // on the right, its left edge: dragging left widens it
	if a.state.sessionsLeft() {
		grip.Right(0)
		dir = 1
	} else {
		grip.Left(0)
	}
	if grip.DoubleClicked() {
		a.state.SessionsWidth = 0
	}
	dx, _, held := grip.Dragged()
	if held && dx != 0 {
		a.state.SessionsWidth = min(max(width+dir*dx, minSessionsWidth), maxSessionsWidth)
	}
	if held || grip.Hovered() {
		grip.Children(func() {
			ui.Box(c).Width(2).FillHeight().Background(theme.Accent)
		})
	}
}

// projectGroup draws a project and its sessions, and reports whether it
// drew anything: while filtering, projects without a match are hidden.
func (a *App) projectGroup(c *ui.Context, p proto.ProjectInfo) bool {
	sessions := a.projectSessions(p.Path)
	filtering := strings.TrimSpace(a.filter) != ""
	if filtering {
		sessions = a.matching(p, sessions)
		if len(sessions) == 0 {
			return false
		}
	}
	open := filtering || !a.collapsed[p.Path]
	selected := p.Path == a.project

	ui.Row(c).Height(28).FillWidth().Gap(4).Margin(6, 0, 0, 0).Children(func() {
		row := ui.ButtonBase(c).Grow(1).Height(28).PaddingX(4).Gap(6).Radius(6).Cursor(ui.CursorPointer).Label(p.Name)
		row.Drag(listDrag{path: p.Path})
		a.dropOnProject(row, p)
		if row.Hovered() {
			row.Background(theme.Hover)
		}
		row.Children(func() {
			chev := icon.ChevronRight
			if open {
				chev = icon.ChevronDown
			}
			ui.Icon(c, chev).FontSize(11).TextColor(theme.Muted)
			name := widget.SectionLabel(c, strings.ToUpper(p.Name), theme.Muted).Shrink(1)
			if selected {
				name.TextColor(theme.TextSoft)
			}
			ui.Spacer(c)
			ui.Text(c, summary(sessions)).FontSize(10.5).TextColor(theme.Muted).SingleLine()
		})
		if row.Clicked() {
			if selected || filtering {
				a.collapsed[p.Path] = open
			} else {
				a.project = p.Path
				a.selected = ""
				a.page = ""
				delete(a.collapsed, p.Path)
			}
		}
		a.menu.Open(row, "project:"+p.Path)
		a.menu.Show(c, row, "project:"+p.Path, func(done func()) { a.projectMenuItems(c, p, done) })
		if n := a.linkConflictCount(p.Path); n > 0 {
			warn := widget.IconButton(c, icon.Alert, fmt.Sprintf("%d file(s) in conflict with its other copies: pick versions", n), false)
			warn.TextColor(theme.Warning)
			if warn.Clicked() {
				a.openLinks(p)
			}
		}
		more := widget.IconButton(c, icon.More, "More for "+p.Name, false)
		a.menu.OpenBelow(more, "project-more:"+p.Path, 28)
		a.menu.Show(c, more, "project-more:"+p.Path, func(done func()) { a.projectMenuItems(c, p, done) })
		if widget.IconButton(c, icon.Plus, "New session in "+p.Name, false).Clicked() {
			a.openNewSession(p)
		}
	})
	if !open {
		return true
	}
	if !filtering {
		a.pipelineCards(c, p)
		a.projectNotesRow(c.Key("notes:"+p.Path), p)
	}
	if len(sessions) == 0 {
		ui.Text(c, "No sessions yet").FontSize(11).TextColor(theme.Muted).Padding(4, 8, 4, 22)
	}
	for i, s := range sessions {
		if !filtering && (waitsForYou(s) || startsLater(s)) {
			continue // in Needs you or Queued, above
		}
		a.sessionRow(c.Key(s.ID), i, s)
	}
	return true
}

// waitsForYou reports whether s's agent asks something of the user.
func waitsForYou(s proto.SessionInfo) bool {
	return s.State == proto.StateWaiting && !s.Exited && s.Parent == ""
}

// needsYou lists, above the projects, the sessions whose agent waits for
// the user: what to look at first.
func (a *App) needsYou(c *ui.Context) {
	if strings.TrimSpace(a.filter) != "" {
		return
	}
	var waiting []proto.SessionInfo
	for _, p := range a.projects {
		for _, s := range a.projectSessions(p.Path) {
			if waitsForYou(s) {
				waiting = append(waiting, s)
			}
		}
	}
	if len(waiting) == 0 {
		return
	}
	ui.Row(c.Key("needs-you")).Height(28).FillWidth().Gap(6).PaddingX(4).Margin(6, 0, 0, 0).AlignItems(ui.Center).Children(func() {
		widget.Dot(c, theme.Warning, false, 6)
		widget.SectionLabel(c, "NEEDS YOU", theme.Warning)
		ui.Spacer(c)
		ui.Text(c, strconv.Itoa(len(waiting))).FontSize(10.5).TextColor(theme.Warning)
	})
	for i, s := range waiting {
		a.sessionRow(c.Key("wait:"+s.ID), i, s)
	}
}

// projectMenuItems are a project's menu: its right-click menu and its
// More button's.
func (a *App) projectMenuItems(c *ui.Context, p proto.ProjectInfo, done func()) {
	if widget.MenuItem(c, icon.Plus, "New session", false) {
		a.openNewSession(p)
		done()
	}
	if a.client != nil && widget.MenuItemTip(c, icon.Chat, "History…", "The agents' past conversations in this project, to resume", false) {
		a.openHistory(p)
		done()
	}
	if a.client != nil && p.IsGit && widget.MenuItemTip(c, icon.List, "New pipeline…", "Agents one after another on a branch: plan, build, review", false) {
		a.openPipeline(p)
		done()
	}
	if a.client != nil && widget.MenuItemTip(c, icon.Notebook, "New note", "A note attached to this project", false) {
		a.newNoteFor(p)
		done()
	}
	if n := len(a.notesOfProject(p.Path)); a.client != nil && n > 0 && widget.MenuItem(c, icon.Notebook, fmt.Sprintf("Notes (%d)", n), false) {
		a.showProjectNotes(p.Path)
		done()
	}
	if a.client != nil && p.IsGit && len(a.allHosts()) > 1 && widget.MenuItemTip(c, icon.Git, "Clone to host…", "Clone this repository on another host", false) {
		a.openCloneTo(p)
		done()
	}
	if a.client != nil && widget.MenuItemTip(c, icon.Server, "Copy to host…", "Copy the project and its sessions to another host", false) {
		a.openCopy(p)
		done()
	}
	if a.client != nil && len(a.allHosts()) > 1 && widget.MenuItemTip(c, icon.Sync, "Sync with host…", "Sync its files with its copy on another host, now", false) {
		a.openProjectSync(p)
		done()
	}
	if a.client != nil && widget.MenuItemTip(c, icon.ArrowUp, "Auto sync & push…", "Keep its copies on other hosts in sync, and push to a branch of its own, on a schedule", false) {
		a.openLinks(p)
		done()
	}
	if a.client != nil && widget.MenuItemTip(c, icon.Moon, "Hibernate idle", "Hibernate the project's idle sessions, freeing their memory", false) {
		a.hibernateIdle(p.Path)
		done()
	}
	if a.client != nil {
		widget.MenuSeparator(c)
		a.projectArrangeItems(c, p, done)
	}
	widget.MenuSeparator(c)
	if widget.MenuItemTip(c, icon.Pencil, "Rename…", "Its name in hustleX; the folder keeps its own", false) {
		a.projectRename = &projectRenameForm{open: true, path: p.Path, name: p.Name}
		done()
	}
	if widget.MenuItemTip(c, icon.Trash, "Remove…", "Remove the project from hustleX; its files stay", true) {
		a.removing = &p
		done()
	}
}

// removeProjectDialog confirms removing a project from the list. Its
// folder stays on disk; a project with sessions still running can't go,
// as they would be left out of the list.
func (a *App) removeProjectDialog(c *ui.Context) {
	if a.removing == nil {
		return
	}
	p := *a.removing
	running := 0
	for _, s := range a.projectSessions(p.Path) {
		if !s.Exited {
			running++
		}
	}
	open := true
	if running > 0 {
		msg := fmt.Sprintf("%d %s still running in it. Delete %s first.", running,
			textfmt.Plural(running, "session is", "sessions are"), textfmt.Plural(running, "it", "them"))
		if ui.AlertDialog(c, &open, "Can't remove “"+p.Name+"” yet", msg, "OK") != -1 || !open {
			a.removing = nil
		}
		return
	}
	msg := "The folder and its worktrees stay on disk. Open it again to bring it back."
	switch ui.AlertDialog(c, &open, "Remove “"+p.Name+"” from hustleX?", msg, "Cancel", "Remove") {
	case 1:
		a.removeProject(p)
		a.removing = nil
	case 0, ui.AlertDismissed:
		a.removing = nil
	}
}

// removeProject forgets p on the host; the projects event updates the
// list.
func (a *App) removeProject(p proto.ProjectInfo) {
	cl := a.client
	if cl == nil {
		return
	}
	if a.project == p.Path {
		a.project, a.selected = "", ""
	}
	go func() {
		if err := cl.RemoveProject(p.Path); err != nil {
			a.onMain(func() { a.status = err.Error() })
		}
	}()
}

// matching is the sessions whose name, branch, agent or project contain
// the filter.
func (a *App) matching(p proto.ProjectInfo, sessions []proto.SessionInfo) []proto.SessionInfo {
	q := strings.ToLower(strings.TrimSpace(a.filter))
	var out []proto.SessionInfo
	for _, s := range sessions {
		for _, field := range []string{a.sessionName(s), s.Branch, a.agentName(s.Agent), p.Name} {
			if strings.Contains(strings.ToLower(field), q) {
				out = append(out, s)
				break
			}
		}
	}
	return out
}

func (a *App) sessionRow(c *ui.Context, i int, s proto.SessionInfo) {
	st := statusOf(s)
	active := s.ID == a.selected
	row := ui.ButtonBase(c).Label(a.sessionName(s)).Transition(widget.RowMotion(i))
	if a.picked[s.ID] && !active {
		row.Border(1, theme.AccentLine)
	}
	widget.ListRow(c, row, active, "sessions", func() {
		agentTile(c, s.Agent, 26)
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Row(c).Gap(4).MinWidth(0).AlignItems(ui.Center).Children(func() {
				name := ui.Text(c, a.sessionName(s)).FontSize(12.5).SingleLine().Shrink(1)
				switch {
				case startsLater(s):
					name.TextColor(theme.TextSoft)
				case active || a.unread[s.ID]:
					name.TextColor(theme.TextStrong)
				}
				if a.unread[s.ID] {
					name.FontWeight(650)
				}
				// The agent's note stays out of the row, which keeps every row
				// two lines high; hovering the name shows it, and the names
				// the session had.
				var tip []string
				if s.Note != "" {
					tip = append(tip, s.Note)
				}
				if len(s.Earlier) > 0 {
					tip = append(tip, "Earlier: "+strings.Join(s.Earlier, " → "))
				}
				if len(tip) > 0 {
					name.Tooltip(strings.Join(tip, "\n"))
				}
				if s.Named {
					ui.Box(c.Key("pin")).Shrink(0).Tooltip("Named by you: agents leave it").Children(func() {
						ui.Icon(c, icon.Pin).FontSize(10).TextColor(theme.Muted)
					})
				}
			})
			ui.Row(c).Gap(5).MinWidth(0).AlignItems(ui.Center).Children(func() {
				if startsLater(s) {
					ui.Icon(c, icon.Clock).FontSize(10.5).TextColor(theme.Muted)
					ui.Text(c, a.startsDetail(s)).FontSize(11).TextColor(theme.Muted).SingleLine().Shrink(1)
					return
				}
				widget.Dot(c, st.color, st.hollow, 6)
				detail := st.label
				switch {
				case s.Branch != "" && s.Branch != a.sessionName(s):
					detail += " · " + s.Branch
				case !s.Worktree && s.Project != "":
					detail += " · project folder"
				}
				// What the agent says it is at replaces the branch while it works,
				// so the row stays two lines high.
				if s.Note != "" && !s.Exited {
					detail = textfmt.FirstLine(s.Note, 60)
				}
				ui.Text(c, detail).FontSize(11).TextColor(theme.Muted).SingleLine().Shrink(1)
			})
		})
		a.conflictMark(c, s)
		if a.unread[s.ID] {
			ui.Box(c).Size(7, 7).Radius(4).Background(theme.Accent).Shrink(0).Tooltip("Unread: its agent finished or asks something")
		}
		if u, ok := a.usage.Session(s.ID); ok && u.RSS > 0 && !a.compact() {
			ui.Text(c, textfmt.Bytes(u.RSS)).FontSize(10.5).TextColor(theme.Muted).SingleLine().Shrink(0)
		}
	})
	a.sessionMenu(c, row, s)
	// Dragged out of the window, the session gets one of its own.
	row.Drag(sessionDrag{s.ID})
	if _, _, ended := row.DragEnded(); ended {
		a.dropOutsideWindow(s)
	}
	if row.Clicked() {
		if row.ClickModifiers()&(ui.Cmd|ui.Shift) != 0 {
			a.pick(s)
		} else {
			a.picked = nil
			a.selectSession(s)
		}
	}
}

// sessionMenu is a session's right-click menu, also opened by the
// focused session's More button.
func (a *App) sessionMenu(c *ui.Context, anchor ui.Element, s proto.SessionInfo) {
	key := "session:" + s.ID
	a.menu.Open(anchor, key)
	a.menu.Show(c, anchor, key, func(done func()) { a.sessionMenuItems(c, s, done) })
}

func (a *App) sessionMenuItems(c *ui.Context, s proto.SessionInfo, done func()) {
	if canChat(s) {
		label, svg := "Show as chat", icon.Chat
		if a.showsChat(s.ID) {
			label, svg = "Show the terminal", icon.Terminal
		}
		if widget.MenuItem(c, svg, label, false) {
			a.toggleChat(s)
			done()
		}
	}
	if widget.MenuItemTip(c, icon.AppWindow, "Open in new window", "Or drag it out of the window", false) {
		a.openSessionWindow(s, nil)
		done()
	}
	label := "Maximize"
	if a.maximized != "" {
		label = "Restore"
	}
	if widget.MenuItem(c, icon.Maximize, label, false) {
		a.selectSession(s)
		a.toggleMaximized(s.ID)
		done()
	}
	if a.canReview(s) && !(a.panelShows(panelChanges) && a.selected == s.ID) && widget.MenuItem(c, icon.Git, "Show changes", false) {
		a.selectSession(s)
		a.openPanel(panelChanges)
		done()
	}
	if s.Named && widget.MenuItemTip(c, icon.Pin, "Let agents name it", "Your name for it goes; its agent names it again", false) {
		a.renameSession(s, "")
		done()
	}
	if !a.unread[s.ID] && widget.MenuItem(c, icon.Chat, "Mark as unread", false) {
		if a.unread == nil {
			a.unread = map[string]bool{}
		}
		a.unread[s.ID] = true
		if a.selected == s.ID {
			a.selected = "" // shown, it would read it again
		}
		done()
	}
	muted := slices.Contains(a.notifySettings().MutedSessions, s.ID)
	if label := map[bool]string{false: "Mute sounds", true: "Unmute sounds"}[muted]; s.Agent != "" && widget.MenuItem(c, icon.VolumeDown, label, false) {
		a.muteSession(s.ID, !muted)
		done()
	}
	if canContinue(s) && widget.MenuItem(c, icon.Play, "Continue", false) {
		a.continueSession(s)
		done()
	}
	a.hibernateItems(c, s, done)
	if !s.Exited && widget.MenuItem(c, icon.Gauge, "Set time budget…", false) {
		a.askBudget(s)
		done()
	}
	for _, ed := range editors {
		if widget.MenuItem(c, icon.Pencil, "Open in "+ed.name, false) {
			a.openInEditor(ed, a.sessionFolder(s), 0)
			done()
		}
	}
	if a.splitOn() && s.Project == a.project && len(a.splitSessions(a.projectSessions(a.project))) > 1 &&
		widget.MenuItemTip(c, icon.Split, "Unsplit", "Leave it out of the sessions side by side", false) {
		a.hideFromSplit(s.ID, true)
		done()
	}
	if i := slices.IndexFunc(a.projects, func(p proto.ProjectInfo) bool { return p.Path == s.Project }); i >= 0 &&
		widget.MenuItemTip(c, icon.Notebook, "New note", "A note attached to its project", false) {
		a.newNoteFor(a.projects[i])
		done()
	}
	widget.MenuSeparator(c)
	if widget.MenuItem(c, icon.Trash, "Delete session…", true) {
		a.askDelete(s)
		done()
	}
}

func summary(sessions []proto.SessionInfo) string {
	working, waiting := 0, 0
	for _, s := range sessions {
		if isWorking(s) {
			working++
		}
		if s.State == proto.StateWaiting && !s.Exited {
			waiting++
		}
	}
	switch {
	case waiting > 0:
		return fmt.Sprintf("%d need input", waiting)
	case working > 0:
		return fmt.Sprintf("%d working", working)
	case len(sessions) > 0:
		return fmt.Sprintf("%d", len(sessions))
	}
	return ""
}
