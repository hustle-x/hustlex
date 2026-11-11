package ui

import (
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

// welcomeAgent is the agent the welcome offers: the first installed, else
// Claude Code.
func (a *App) welcomeAgent() (proto.AgentInfo, bool) {
	for _, ag := range a.agents {
		if ag.Installed && ag.Command != "" {
			return ag, true
		}
	}
	return a.agentInfo("claude"), false
}

// claudeSignedIn reports whether hustleX reads an account of Claude Code.
func (a *App) claudeSignedIn() bool {
	for _, acc := range a.credits.Accounts() {
		if agent.Base(acc.Agent) == "claude" {
			return true
		}
	}
	return false
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
	case s.Agent == "claude" && a.claudeSignedIn():
		a.killSession(s.ID) // the shell its sign-in drops to
	default:
		return
	}
	a.selected, a.welcomeRun = "", ""
}

// welcome draws the steps.
func (a *App) welcome(c *ui.Context) {
	ag, installed := a.welcomeAgent()
	signedIn := ag.ID != "claude" || a.claudeSignedIn()
	hasProject := len(a.projects) > 0
	ui.Scroll(c).Grow(1).FillWidth().Children(func() {
		ui.Column(c).FillWidth().Center().Padding(48, 24).Children(func() {
			ui.Column(c).Width(520).Gap(14).Children(func() {
				ui.Text(c, "Welcome to hustleX").FontSize(20)
				ui.Text(c, "Three steps to your first agent session.").TextColor(theme.Muted)
				ui.Box(c).Height(4)
				a.welcomeStep(c, 1, installed, "Install an agent", "Claude Code runs in hustleX; Codex, Gemini CLI and others can be picked in New session.", func() {
					if ag.Install != "" && widget.PrimaryButton(c, icon.Agent(ag.ID), "Install "+ag.Name, "").Disabled(a.newSession.busy).Clicked() {
						a.newSession.project = proto.ProjectDetail{}
						a.installAgent(ag)
					} else if ag.Install == "" && ag.Homepage != "" {
						ui.Link(c, "How to install "+ag.Name, ag.Homepage).FontSize(12)
					}
				})
				signDetail := "It opens your browser to sign in to your Claude account."
				checking := ag.ID == "claude" && installed && !signedIn && !a.credits.Loaded()
				if checking {
					signDetail = "Checking for an account signed in already…"
				}
				if ag.ID != "claude" {
					signDetail = ag.Name + " asks you to sign in on its first session."
				}
				a.welcomeStep(c, 2, installed && signedIn, "Sign in", signDetail, func() {
					if installed && !signedIn && !checking && widget.PrimaryButton(c, icon.Agent(ag.ID), "Sign in to "+ag.Name, "").Clicked() {
						a.signInFromCredits(proto.ProjectInfo{}, ag.ID, proto.RefDefault)
					}
				})
				a.welcomeStep(c, 3, hasProject, "Open a project folder", "The folder the agent works in: an app, a site, any code.", func() {
					if !hasProject && widget.PrimaryButton(c, icon.FolderPlus, "Open folder", "").Clicked() {
						a.openFolder()
					}
				})
				a.formError(c)
				ui.Row(c).FillWidth().Gap(8).Margin(8, 0, 0, 0).Children(func() {
					ready := installed && hasProject
					b := widget.PrimaryButton(c, icon.Plus, "Start your first session", "")
					if !ready {
						b.Disabled(true).Opacity(0.45)
					} else if b.Clicked() {
						p, ok := a.currentProject()
						if !ok {
							p = a.projects[0]
						}
						a.openNewSession(p)
						a.newSession.agent = ag.ID
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

// welcomeStep is one step: its number, or a tick once done, and what to
// do while it isn't.
func (a *App) welcomeStep(c *ui.Context, n int, done bool, title, detail string, action func()) {
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
	})
}
