// Package turnsnap keeps the files of a session's folder as they were
// before its agent's turn, to put them back (undo) and forth (redo). A
// snapshot is a git tree of the folder's files, the ignored ones left
// out, written with an index of its own: the user's branch, index and
// files stay as they are.
package turnsnap

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hustlex/hustlex/core/git"
)

// ErrNotGit says the folder isn't in a git repository, so it has no
// snapshots.
var ErrNotGit = errors.New("turnsnap: the folder isn't in a git repository")

// ErrTooBig says the folder's files git doesn't track yet are too many
// or too large to copy into its repository at every turn.
var ErrTooBig = errors.New("turnsnap: too many or too large untracked files to snapshot")

// What untracked files a snapshot copies into the repository, at most.
const (
	maxFile  = 50 << 20
	maxTotal = 300 << 20
	maxFiles = 20000
)

// Take writes dir's files as a git tree and returns its ID.
func Take(dir string) (string, error) {
	top, err := git.TopLevel(dir)
	if err != nil {
		return "", ErrNotGit
	}
	if err := checkUntracked(top); err != nil {
		return "", err
	}
	idx, err := os.CreateTemp("", "hustlex-turn-*.index")
	if err != nil {
		return "", fmt.Errorf("turnsnap: %w", err)
	}
	idx.Close()
	os.Remove(idx.Name()) // git makes it
	defer os.Remove(idx.Name())
	env := []string{"GIT_INDEX_FILE=" + idx.Name()}
	if head, _ := runEnv(top, env, "rev-parse", "--verify", "--quiet", "HEAD^{commit}"); head != "" {
		// Starting from HEAD, only what changed is hashed.
		if _, err := runEnv(top, env, "read-tree", head); err != nil {
			return "", fmt.Errorf("turnsnap: %w", err)
		}
	}
	if _, err := runEnv(top, env, "add", "-A", "--", "."); err != nil {
		return "", fmt.Errorf("turnsnap: %w", err)
	}
	tree, err := runEnv(top, env, "write-tree")
	if err != nil {
		return "", fmt.Errorf("turnsnap: %w", err)
	}
	return tree, nil
}

// checkUntracked refuses a snapshot whose untracked files are too many
// or too large: each would go into the repository's objects.
func checkUntracked(top string) error {
	out, err := git.Run(top, "ls-files", "-z", "--others", "--exclude-standard")
	if err != nil {
		return fmt.Errorf("turnsnap: %w", err)
	}
	var total int64
	n := 0
	for _, f := range strings.Split(out, "\x00") {
		if f == "" {
			continue
		}
		if n++; n > maxFiles {
			return ErrTooBig
		}
		info, err := os.Lstat(filepath.Join(top, filepath.FromSlash(f)))
		if err != nil {
			continue
		}
		if info.Size() > maxFile {
			return ErrTooBig
		}
		if total += info.Size(); total > maxTotal {
			return ErrTooBig
		}
	}
	return nil
}

// Changed lists the files that differ between trees a and b, relative
// to the repository's root.
func Changed(dir, a, b string) ([]string, error) {
	if a == b {
		return nil, nil
	}
	top, err := git.TopLevel(dir)
	if err != nil {
		return nil, ErrNotGit
	}
	out, err := git.Run(top, "diff-tree", "-r", "-z", "--name-only", "--no-renames", a, b)
	if err != nil {
		return nil, fmt.Errorf("turnsnap: %w", err)
	}
	var files []string
	for _, f := range strings.Split(out, "\x00") {
		if f != "" {
			files = append(files, f)
		}
	}
	return files, nil
}

// Restore puts dir's files that differ between tree now (what they are)
// and tree to back as to has them: changed ones rewritten, ones to lacks
// removed. It returns the files it put back.
func Restore(dir, now, to string) ([]string, error) {
	files, err := Changed(dir, now, to)
	if err != nil || len(files) == 0 {
		return nil, err
	}
	top, _ := git.TopLevel(dir)
	inTo := map[string]bool{}
	out, err := git.Run(top, append([]string{"ls-tree", "-z", "--name-only", to, "--"}, files...)...)
	if err != nil {
		return nil, fmt.Errorf("turnsnap: %w", err)
	}
	for _, f := range strings.Split(out, "\x00") {
		if f != "" {
			inTo[f] = true
		}
	}
	var keep []string
	for _, f := range files {
		if inTo[f] {
			keep = append(keep, f)
			continue
		}
		if err := os.Remove(filepath.Join(top, filepath.FromSlash(f))); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("turnsnap: %w", err)
		}
		removeEmptyParents(top, filepath.Dir(filepath.Join(top, filepath.FromSlash(f))))
	}
	if len(keep) > 0 {
		// The work tree alone: the index, and so what is staged, stays.
		if _, err := git.Run(top, append([]string{"restore", "--source=" + to, "--worktree", "--"}, keep...)...); err != nil {
			return nil, fmt.Errorf("turnsnap: %w", err)
		}
	}
	return files, nil
}

// removeEmptyParents removes dir and its parents up to top while empty:
// the folders a turn made for files it added.
func removeEmptyParents(top, dir string) {
	for dir != top && strings.HasPrefix(dir, top) {
		if os.Remove(dir) != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}

func runEnv(dir string, env []string, args ...string) (string, error) {
	cmd := git.Command(dir, args...)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return "", fmt.Errorf("git %s: %s", args[0], strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return strings.TrimSpace(string(out)), nil
}
