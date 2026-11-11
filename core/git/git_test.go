package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// A repository reached through a symlink keeps the path it was reached
// by, as macOS's /var is /private/var.
func TestTopLevelThroughASymlink(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	real := t.TempDir()
	repo := filepath.Join(real, "repo")
	if err := os.MkdirAll(filepath.Join(repo, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", repo, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %s", out)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skip("no symlinks here")
	}
	for dir, want := range map[string]string{
		filepath.Join(link, "repo"):        filepath.Join(link, "repo"),
		filepath.Join(link, "repo", "sub"): filepath.Join(link, "repo"),
	} {
		got, err := TopLevel(dir)
		if err != nil || got != want {
			t.Errorf("TopLevel(%s) = %s, %v; want %s", dir, got, err, want)
		}
	}
}

func TestNestedRepo(t *testing.T) {
	dir := t.TempDir()
	if got := NestedRepo(dir); got != "" {
		t.Fatalf("empty folder: %q", got)
	}
	for _, d := range []string{"docs", "app/.git", ".hidden/.git", "site/.git"} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	old, recent := time.Now().Add(-time.Hour), time.Now()
	for d, at := range map[string]time.Time{"app": recent, "site": old} {
		index := filepath.Join(dir, d, ".git", "index")
		if err := os.WriteFile(index, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(index, at, at); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := NestedRepo(dir), filepath.Join(dir, "app"); got != want {
		t.Fatalf("worked in last: %q, want %q", got, want)
	}
}
