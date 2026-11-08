package ui

import (
	"encoding/json"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/proto"
	"github.com/hustlex/hustlex/ui/icon"
	"github.com/hustlex/hustlex/ui/theme"
	"github.com/hustlex/hustlex/ui/widget"
)

// hooksForm edits the hooks of the current project: a command or a webhook
// for when an agent's session starts, finishes, needs you or fails.
type hooksForm struct {
	open    bool
	project proto.ProjectInfo
	text    string // the hooks as JSON, what is edited
	allowed bool
	loaded  bool
	err     string
}

const hooksExample = `[
  {"event": "done", "run": "say 'agent finished'"},
  {"event": "needs-input", "webhook": "https://example.com/hook"}
]`

func (a *App) openHooks() {
	p, ok := a.currentProject()
	cl := a.client
	if !ok || cl == nil {
		return
	}
	a.hooks = hooksForm{open: true, project: p}
	go func() {
		st, err := cl.Hooks(p.Path)
		a.onMain(func() {
			if !a.hooks.open || a.hooks.project.Path != p.Path {
				return
			}
			if err != nil {
				a.hooks.err = err.Error()
				return
			}
			text := hooksExample
			if len(st.Hooks) > 0 {
				b, _ := json.MarshalIndent(st.Hooks, "", "  ")
				text = string(b)
			}
			a.hooks.text, a.hooks.allowed, a.hooks.loaded = text, st.Allowed, true
		})
	}()
}

func (a *App) saveHooks() {
	f := &a.hooks
	var list []proto.ProjectHook
	if err := json.Unmarshal([]byte(f.text), &list); err != nil {
		f.err = "That isn't a list of hooks: " + err.Error()
		return
	}
	cl := a.client
	if cl == nil {
		f.err = "Not connected."
		return
	}
	go func() {
		_, err := cl.SetHooks(proto.HooksState{Project: f.project.Path, Hooks: list, Allowed: f.allowed})
		a.onMain(func() {
			if err != nil {
				f.err = strings.TrimPrefix(err.Error(), "daemon: ")
				return
			}
			f.open = false
			a.notify("Saved the hooks of " + f.project.Name)
		})
	}()
}

func (a *App) hooksDialog(c *ui.Context) {
	f := &a.hooks
	if !f.open {
		return
	}
	widget.Sheet(c, &f.open, 560, func() {
		ui.Column(c).Gap(5).Grow(1).MinWidth(0).Children(func() {
			widget.Breadcrumb(c, a.host.Name, f.project.Name, "Hooks")
			ui.Text(c, f.project.Path+"/.hustlex/hooks.json").Font(theme.MonoFont).FontSize(11).TextColor(theme.Muted).SingleLine()
		})
		if widget.IconButton(c, icon.Close, "Close", false).Clicked() {
			f.open = false
		}
	}, func() {
		ui.Text(c, "Run a command or call a webhook when an agent's session starts, is done, needs you (needs-input) or fails (error). A command runs in the session's folder, with HUSTLEX_EVENT, HUSTLEX_SESSION and HUSTLEX_PROJECT in its environment.").
			FontSize(12).TextColor(theme.TextSoft)
		ui.TextArea(c, &f.text).Height(190).Padding(9, 11).Radius(theme.RadiusControl).Font(theme.MonoFont).
			Background(theme.Sunken).Border(1, theme.BorderStrong)
		widget.Card(c).FillWidth().Children(func() {
			ui.Row(c).FillWidth().Gap(12).Children(func() {
				ui.Column(c).Gap(3).Grow(1).MinWidth(0).Children(func() {
					ui.Text(c, "Run these hooks on "+a.host.Name).FontSize(12.5)
					ui.Text(c, "A project's hooks don't run until you allow them: opening a repository must not run its commands.").
						FontSize(11.5).TextColor(theme.Muted)
				})
				widget.Switch(c, &f.allowed, "Run these hooks")
			})
		})
		if f.err != "" {
			ui.Text(c, f.err).FontSize(12).TextColor(theme.Error)
		}
	}, func() {
		ui.Spacer(c)
		if widget.GhostButton(c, "Cancel").Clicked() {
			f.open = false
		}
		if widget.PrimaryButton(c, nil, "Save", "").Disabled(!f.loaded).Clicked() {
			a.saveHooks()
		}
	})
}
