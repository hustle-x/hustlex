package turnsnap

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/hustlex/hustlex/core/git"
)

func write(t *testing.T, dir, name, text string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(dir, name string) string {
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return "<gone>"
	}
	return string(b)
}

// A turn's edits, additions and removals go back, and forth again; the
// user's index and ignored files stay as they are.
func TestUndoAndRedoATurn(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "t@example.com"}, {"config", "user.name", "t"}} {
		if _, err := git.Run(dir, args...); err != nil {
			t.Fatal(err)
		}
	}
	write(t, dir, ".gitignore", "secret.env\n")
	write(t, dir, "app.go", "v1")
	write(t, dir, "old.go", "old")
	if _, err := git.Run(dir, "add", "-A"); err != nil {
		t.Fatal(err)
	}
	if _, err := git.Run(dir, "commit", "-qm", "first"); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "draft.txt", "the user's, not committed")
	if _, err := git.Run(dir, "add", "draft.txt"); err != nil {
		t.Fatal(err)
	}
	before, err := Take(dir)
	if err != nil {
		t.Fatal(err)
	}

	// The turn.
	write(t, dir, "app.go", "v2")
	write(t, dir, "pkg/new.go", "new")
	os.Remove(filepath.Join(dir, "old.go"))
	write(t, dir, "secret.env", "ignored")
	after, err := Take(dir)
	if err != nil {
		t.Fatal(err)
	}
	changed, _ := Changed(dir, before, after)
	if want := []string{"app.go", "old.go", "pkg/new.go"}; !slices.Equal(changed, want) {
		t.Fatalf("changed %v, want %v", changed, want)
	}

	files, err := Restore(dir, after, before)
	if err != nil || len(files) != 3 {
		t.Fatalf("restored %v: %v", files, err)
	}
	if read(dir, "app.go") != "v1" || read(dir, "old.go") != "old" || read(dir, "pkg/new.go") != "<gone>" {
		t.Fatalf("after undo: app %q old %q new %q", read(dir, "app.go"), read(dir, "old.go"), read(dir, "pkg/new.go"))
	}
	if _, err := os.Stat(filepath.Join(dir, "pkg")); !os.IsNotExist(err) {
		t.Fatal("the folder the turn made stays")
	}
	if read(dir, "secret.env") != "ignored" || read(dir, "draft.txt") != "the user's, not committed" {
		t.Fatal("undo touched what the turn didn't")
	}
	if staged, _ := git.Run(dir, "diff", "--cached", "--name-only"); staged != "draft.txt" {
		t.Fatalf("staged %q", staged)
	}

	if _, err := Restore(dir, before, after); err != nil {
		t.Fatal(err)
	}
	if read(dir, "app.go") != "v2" || read(dir, "old.go") != "<gone>" || read(dir, "pkg/new.go") != "new" {
		t.Fatal("redo didn't put the turn back")
	}
}

func TestNotGit(t *testing.T) {
	if _, err := Take(t.TempDir()); err != ErrNotGit {
		t.Fatalf("Take = %v", err)
	}
}

func TestTooBigUntracked(t *testing.T) {
	dir := t.TempDir()
	if _, err := git.Run(dir, "init", "-q"); err != nil {
		t.Fatal(err)
	}
	f, _ := os.Create(filepath.Join(dir, "dump.bin"))
	f.Truncate(maxFile + 1)
	f.Close()
	if _, err := Take(dir); err != ErrTooBig {
		t.Fatalf("Take = %v", err)
	}
	write(t, dir, ".gitignore", "dump.bin\n")
	if _, err := Take(dir); err != nil {
		t.Fatalf("ignored, it still counts: %v", err)
	}
}
