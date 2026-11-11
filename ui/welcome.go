package ui

import (
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/core/agent"
	"github.com/hustlex/hustlex/proto"
	"github.com/hustlex/hustlex/ui/icon"
	"github.com/hustlex/hustlex/ui/theme"
	"github.com/hustlex/hustlex/ui/widget"
)

// Until the first session, the workspace walks a new user through what a
// session needs: an agent, signed in, and a folder. Installing and
// signing in run in sessions of the home folder; once one is done, the
// welcome comes back with its step ticked.

// welcomeShown reports whether the workspace shows the welcome instead of
// an empty state.
func (a *App) welcomeShown() bool {
	if a.state.WelcomeDone || a.client == nil || len(a.agents) == 0 {
		return false
	}
	for _, s := range a.sessions {
		if s.Project != "" && s.ID != a.welcomeRun {
			return false // used already
		}
	}
	return true
}

// welcomeAgents are the agents the welcome offers first, in this order;
// the others are under More.
var welcomeAgents = []string{"claude", "codex", "gemini", "opencode"}

// welcomeAgent is the agent the welcome sets up: the one picked, else the
// first installed, else Claude Code.
func (a *App) welcomeAgent() (proto.AgentInfo, bool) {
	if a.welcomePick != "" {
		ag := a.agentInfo(a.welcomePick)
		return ag, ag.Installed
	}
	for _, ag := range a.agents {
		if ag.Installed && ag.Command != "" {
			return ag, true
		}
	}
	return a.agentInfo("claude"), false
}

// signedIn reports whether agent id has an account hustleX reads, or, for
// an agent whose accounts it doesn't read, whether its sign-in ran.
func (a *App) signedIn(id string) bool {
	base := agent.Base(id)
	if base != "claude" && base != "codex" {
		return a.agentInfo(base).SignIn == "" || a.welcomeSigned[base] // nothing to sign in to, or its sign-in ran
	}
	for _, acc := range a.credits.Accounts() {
		if agent.Base(acc.Agent) == base {
			return true
		}
	}
	return false
}

// signInLabel is the sign-in button's words for agent id.
func signInLabel(ag proto.AgentInfo) (button, detail string) {
	switch agent.Base(ag.ID) {
	case "claude":
		return "Sign in to Claude Code", "Opens your browser to sign in to your Claude account."
	case "codex":
		return "Sign in with ChatGPT", "Opens your browser to sign in with your ChatGPT account."
	case "gemini":
		return "Sign in with Google", "Opens your browser to sign in with your Google account."
	case "opencode":
		return "Connect a provider", "Connect a model provider: an API key, or a sign-in it offers."
	}
	return "Sign in to " + ag.Name, ag.Name + " signs in in a session of its own."
}

// welcomeTick goes back to the welcome once the install or sign-in it
// started is done.
func (a *App) welcomeTick() {
	if a.welcomeRun == "" || a.selected != a.welcomeRun {
		return
	}
	s, ok := a.sessionByID(a.welcomeRun)
	switch {
	case !ok:
	case s.Exited:
		if s.Agent != "" && s.Agent != agent.Custom {
			if a.welcomeSigned == nil {
				a.welcomeSigned = map[string]bool{}
			}
			a.welcomeSigned[agent.Base(s.Agent)] = true // its sign-in ran
		}
	case a.signedIn(s.Agent) && agent.Base(s.Agent) != "gemini" && agent.Base(s.Agent) != "opencode":
		a.killSession(s.ID) // the shell its sign-in drops to
	default:
		return
	}
	a.selected, a.welcomeRun = "", ""
}

// welcome draws the steps; they can be taken in any order, but signing in
// needs the agent installed.
func (a *App) welcome(c *ui.Context) {
	ag, installed := a.welcomeAgent()
	checking := installed && !a.credits.Loaded() && (agent.Base(ag.ID) == "claude" || agent.Base(ag.ID) == "codex")
	signedIn := installed && a.signedIn(ag.ID)
	hasProject := len(a.projects) > 0
	// The next step's button is the accent one; the others are outlined.
	next := 3
	switch {
	case !installed:
		next = 1
	case !signedIn:
		next = 2
	}
	kind := func(n int) widget.ButtonKind {
		if n == next {
			return widget.Primary
		}
		return widget.Secondary
	}
	ui.Scroll(c).Grow(1).FillWidth().Children(func() {
		ui.Column(c).FillWidth().Center().Padding(48, 24).Children(func() {
			ui.Column(c).Width(540).Gap(12).Children(func() {
				ui.Text(c, "Welcome to hustleX").FontSize(20)
				ui.Text(c, "Three steps to your first agent session.").TextColor(theme.Muted)
				ui.Box(c).Height(4)
				a.welcomeStep(c, 1, installed, "Choose your agent", "The AI that writes code in your sessions. You can add others later.", func() {
					if ag.Install != "" && widget.Button(c, kind(1), icon.Agent(ag.ID), "Install "+ag.Name, "").Disabled(a.newSession.busy).Clicked() {
						a.newSession.project = proto.ProjectDetail{}
						a.installAgent(ag)
					} else if ag.Install == "" && ag.Homepage != "" {
						ui.Link(c, "How to install "+ag.Name, ag.Homepage).FontSize(12)
					}
				}, func() {
					if !installed {
						a.welcomeAgentChips(c, ag.ID)
					}
				})
				button, detail := signInLabel(ag)
				if checking {
					detail = "Checking for an account signed in already…"
				}
				a.welcomeStep(c, 2, signedIn, "Sign in", detail, func() {
					switch {
					case !installed:
						ui.Text(c, "After step 1").FontSize(12).TextColor(theme.Muted)
					case !checking && widget.Button(c, kind(2), icon.Agent(ag.ID), button, "").Clicked():
						a.signInFromCredits(proto.ProjectInfo{}, ag.ID, proto.RefDefault)
					}
				}, nil)
				a.welcomeStep(c, 3, hasProject, "Open a project folder", "The folder the agent works in: an app, a site, any code.", func() {
					if widget.Button(c, kind(3), icon.FolderPlus, "Open folder", "").Clicked() {
						a.openFolder()
					}
				}, nil)
				a.formError(c)
				ui.Row(c).FillWidth().Gap(8).Margin(8, 0, 0, 0).AlignItems(ui.Center).Children(func() {
					left := 0
					for _, done := range []bool{installed, signedIn, hasProject} {
						if !done {
							left++
						}
					}
					if left == 0 {
						if widget.PrimaryButton(c, icon.Plus, "Start your first session", "").Clicked() {
							p, ok := a.currentProject()
							if !ok {
								p = a.projects[0]
							}
							a.openNewSession(p)
							a.newSession.agent = ag.ID
						}
					} else {
						ui.Text(c, map[bool]string{true: "1 step left", false: strconv.Itoa(left) + " steps left"}[left == 1]).FontSize(12.5).TextColor(theme.Muted)
					}
					ui.Spacer(c)
					if widget.GhostButton(c, "Skip").Clicked() {
						a.state.WelcomeDone = true
						a.saveState()
					}
				})
			})
		})
	})
}

// welcomeAgentChips pick the agent to set up: the usual ones, then More.
func (a *App) welcomeAgentChips(c *ui.Context, current string) {
	ui.Row(c).FillWidth().Gap(6).Wrap().Children(func() {
		for _, id := range welcomeAgents {
			ag := a.agentInfo(id)
			if ag.Command == "" {
				continue // this daemon doesn't know it
			}
			a.welcomeChip(c, ag, current == id)
		}
		var others []string
		for _, ag := range a.agents {
			if ag.Command != "" && !slices.Contains(welcomeAgents, ag.ID) && agent.Base(ag.ID) == ag.ID {
				others = append(others, ag.Name)
			}
		}
		if len(others) == 0 {
			return
		}
		more := ui.ButtonBase(c.Key("more-agents")).Height(30).PaddingX(10).Gap(7).Radius(theme.RadiusControl).
			Border(1, theme.Border).Cursor(ui.CursorPointer).Label("More agents")
		more.Children(func() {
			ui.Icon(c, icon.More).FontSize(13)
			ui.Text(c, "More").FontSize(12.5)
			ui.Text(c, strings.Join(others[:min(len(others), 3)], ", ")+"…").FontSize(11).TextColor(theme.Muted).SingleLine()
		})
		a.menu.OpenBelow(more, "welcome-agents", 30)
		a.menu.Show(c, more, "welcome-agents", func(done func()) {
			for _, ag := range a.agents {
				if ag.Command == "" || slices.Contains(welcomeAgents, ag.ID) || agent.Base(ag.ID) != ag.ID {
					continue
				}
				if widget.MenuItem(c.Key(ag.ID), icon.Agent(ag.ID), ag.Name, current == ag.ID) {
					a.welcomePick = ag.ID
					done()
				}
			}
		})
	})
}

// welcomeChip is one agent to pick.
func (a *App) welcomeChip(c *ui.Context, ag proto.AgentInfo, on bool) {
	b := ui.ButtonBase(c.Key(ag.ID)).Height(30).PaddingX(10).Gap(7).Radius(theme.RadiusControl).
		Border(1, theme.Border).Cursor(ui.CursorPointer).Label(ag.Name)
	if on {
		b.Border(1, theme.AccentLine).Background(theme.Selected)
	} else if b.Hovered() {
		b.Background(theme.Hover)
	}
	b.Children(func() {
		ui.Icon(c, icon.Agent(ag.ID)).FontSize(13).TextColor(icon.AgentTint(ag.ID, theme.Text))
		ui.Text(c, ag.Name).FontSize(12.5)
	})
	if b.Clicked() {
		a.welcomePick = ag.ID
	}
}

// welcomeStep is one step: its number, or a tick once done, and what to
// do while it isn't.
func (a *App) welcomeStep(c *ui.Context, n int, done bool, title, detail string, action, below func()) {
	widget.Card(c).FillWidth().Children(func() {
		ui.Row(c).FillWidth().Gap(14).AlignItems(ui.Center).Children(func() {
			mark := ui.Box(c).Size(26, 26).Radius(13).Shrink(0).Center()
			if done {
				mark.Background(theme.Done).Children(func() { ui.Icon(c, icon.Check).FontSize(13).TextColor(theme.Background) })
			} else {
				mark.Border(1, theme.BorderStrong).Children(func() { ui.Text(c, string(rune('0'+n))).FontSize(12).TextColor(theme.Muted) })
			}
			ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
				col := theme.Text
				if done {
					col = theme.Muted
				}
				ui.Text(c, title).FontSize(13.5).TextColor(col)
				if !done {
					ui.Text(c, detail).FontSize(12).TextColor(theme.Muted)
				}
			})
			if !done {
				action()
			}
		})
		if !done && below != nil {
			ui.Box(c).FillWidth().Padding(10, 0, 0, 40).Children(below)
		}
	})
}
