package ui

import (
	"testing"

	"github.com/hustlex/hustlex/proto"
)

// A project renamed keeps its name when the list comes again from the
// daemon, and an empty name gives it its folder's back.
func TestRenameProject(t *testing.T) {
	a := New(Options{})
	a.state.ProjectNames = nil
	a.setProjects([]proto.ProjectInfo{{Path: "/code/Hustle Project", Name: "Hustle Project"}})
	a.renameProject("/code/Hustle Project", " Hustle Hub ")
	if got := a.projects[0].Name; got != "Hustle Hub" {
		t.Fatalf("renamed to %q", got)
	}
	a.setProjects([]proto.ProjectInfo{{Path: "/code/Hustle Project", Name: "Hustle Project"}})
	if got := a.projects[0].Name; got != "Hustle Hub" {
		t.Fatalf("after a reload %q", got)
	}
	a.renameProject("/code/Hustle Project", "")
	a.setProjects([]proto.ProjectInfo{{Path: "/code/Hustle Project", Name: "Hustle Project"}})
	if got := a.projects[0].Name; got != "Hustle Project" {
		t.Fatalf("back to %q", got)
	}
}
