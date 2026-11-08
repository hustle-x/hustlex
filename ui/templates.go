package ui

import (
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/ui/icon"
	"github.com/hustlex/hustlex/ui/widget"
)

// sessionTemplate is a New session form kept to use again: the agent, its
// account, how it branches and its first prompt.
type sessionTemplate struct {
	Name       string `json:"name"`
	Agent      string `json:"agent"`
	Account    string `json:"account,omitempty"`
	BranchMode string `json:"branchMode,omitempty"`
	Prompt     string `json:"prompt,omitempty"`
}

// templatesField shows the saved templates as chips above the form, and a
// way to save the form as one.
func (a *App) templatesField(c *ui.Context) {
	if len(a.state.Templates) > 0 {
		widget.Field(c, "Templates", func() {
			ui.Row(c).FillWidth().Gap(6).Wrap().AlignItems(ui.Center).Children(func() {
				for i, t := range a.state.Templates {
					row := c.Key(t.Name)
					ui.Row(row).Gap(0).AlignItems(ui.Center).Children(func() {
						b := widget.Button(row.Key("use"), widget.Secondary, nil, t.Name, "").Height(28).PaddingX(10).
							Tooltip(a.agentName(t.Agent) + templateDetail(t))
						if b.Clicked() {
							a.applyTemplate(t)
						}
						if widget.IconButton(row.Key("del"), icon.Close, "Delete template "+t.Name, false).Clicked() {
							a.state.Templates = slices.Delete(slices.Clone(a.state.Templates), i, i+1)
							a.saveState()
						}
					})
				}
			})
		})
	}
}

func templateDetail(t sessionTemplate) string {
	if t.Prompt == "" {
		return ""
	}
	return " · " + strings.SplitN(strings.TrimSpace(t.Prompt), "\n", 2)[0]
}

// applyTemplate fills the form from t, keeping the project and its branch
// name: a template is how to work, not where.
func (a *App) applyTemplate(t sessionTemplate) {
	f := &a.newSession
	f.agent, f.account, f.accountFor, f.prompt = t.Agent, t.Account, t.Agent, t.Prompt
	if f.project.IsGit && (t.BranchMode == "new" || t.BranchMode == "existing") {
		f.branchMode = t.BranchMode
	}
}

// saveTemplateRow is the name field and button that keep the form as a
// template.
func (a *App) saveTemplateRow(c *ui.Context) {
	f := &a.newSession
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
		widget.Input(c, &f.templateName, "Save this form as a template…").Grow(1)
		name := strings.TrimSpace(f.templateName)
		if b := widget.GhostButton(c, "Save template"); b.Clicked() && name != "" {
			t := sessionTemplate{Name: name, Agent: f.agent, Account: f.account, BranchMode: f.branchMode, Prompt: f.prompt}
			list := slices.DeleteFunc(slices.Clone(a.state.Templates), func(o sessionTemplate) bool { return o.Name == name })
			a.state.Templates = append(list, t)
			a.saveState()
			f.templateName = ""
			a.notify("Saved template " + name)
		}
	})
}
