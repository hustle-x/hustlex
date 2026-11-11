package ui

import (
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/ui/icon"
	"github.com/hustlex/hustlex/ui/widget"
)

// A project's name in the list is its folder's, unless the user gave it
// another: kept by this app, per host, the folder on disk left alone so
// sessions, conversations and paths stay as they are.

// projectRenameForm is the Rename dialog of a project.
type projectRenameForm struct {
	open       bool
	path, name string
}

// renameProject names project path in the list; "" goes back to its
// folder's name.
func (a *App) renameProject(path, name string) {
	key := a.host.ID + " " + path
	name = strings.TrimSpace(name)
	if name == filepath.Base(path) {
		name = ""
	}
	if a.state.ProjectNames == nil {
		a.state.ProjectNames = map[string]string{}
	}
	if name == "" {
		delete(a.state.ProjectNames, key)
	} else {
		a.state.ProjectNames[key] = name
	}
	a.saveState()
	for i := range a.projects {
		if a.projects[i].Path == path {
			a.projects[i].Name = name
			if name == "" {
				a.projects[i].Name = filepath.Base(path)
			}
		}
	}
}

// projectRenameDialog asks for a project's new name.
func (a *App) projectRenameDialog(c *ui.Context) {
	f := a.projectRename
	if f == nil {
		return
	}
	if !f.open {
		a.projectRename = nil
		return
	}
	save := false
	widget.FormSheet(c, &f.open, 420, "Rename project", "Its name in hustleX. The folder "+filepath.Base(f.path)+" keeps its own; leave it empty to use that again.", func() {
		widget.Field(c, "Name", func() {
			if widget.Input(c, &f.name, filepath.Base(f.path)).FillWidth().Submitted() {
				save = true
			}
		})
	}, func() {
		if widget.PrimaryButton(c, icon.Check, "Rename", "").Clicked() {
			save = true
		}
	})
	if save {
		a.renameProject(f.path, f.name)
		a.projectRename = nil
	}
}
