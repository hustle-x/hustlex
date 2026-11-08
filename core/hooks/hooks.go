// Package hooks runs what a project asks for when its sessions' agents
// start, finish, need the user or fail: a command or a webhook, listed in
// the project's .hustlex/hooks.json. A project's hooks run only once the
// user allowed them on this host: a cloned repository must not run
// commands by being opened.
package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/hustlex/hustlex/internal/quietcmd"
	"github.com/hustlex/hustlex/proto"
)

// Events a hook can ask for.
const (
	EventStart = "start"
	EventDone  = "done"
	EventInput = "needs-input"
	EventError = "error"
)

// Events lists them, for checking a file.
var Events = []string{EventStart, EventDone, EventInput, EventError}

// ConfigPath is where a project keeps its hooks.
func ConfigPath(project string) string { return filepath.Join(project, ".hustlex", "hooks.json") }

type file struct {
	Hooks []proto.ProjectHook `json:"hooks"`
}

// Load reads the hooks of a project; none when it has no file.
func Load(project string) ([]proto.ProjectHook, error) {
	data, err := os.ReadFile(ConfigPath(project))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("hooks: %s: %w", ConfigPath(project), err)
	}
	return f.Hooks, Check(f.Hooks)
}

// Check says what is wrong with a list of hooks, or nil.
func Check(list []proto.ProjectHook) error {
	for i, h := range list {
		switch {
		case !slices.Contains(Events, h.Event):
			return fmt.Errorf("hook %d: event %q is not one of %v", i+1, h.Event, Events)
		case h.Run == "" && h.Webhook == "":
			return fmt.Errorf("hook %d: give it a command (run) or a webhook", i+1)
		case h.Webhook != "":
			if u, err := url.Parse(h.Webhook); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
				return fmt.Errorf("hook %d: webhook %q is not an http(s) address", i+1, h.Webhook)
			}
		}
	}
	return nil
}

// Save writes a project's hooks.
func Save(project string, list []proto.ProjectHook) error {
	if err := Check(list); err != nil {
		return err
	}
	data, err := json.MarshalIndent(file{Hooks: list}, "", "  ")
	if err != nil {
		return err
	}
	path := ConfigPath(project)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// Allowed keeps which projects may run hooks on this host.
type Allowed struct {
	mu   sync.Mutex
	path string
}

// OpenAllowed keeps the list in dir.
func OpenAllowed(dir string) *Allowed {
	return &Allowed{path: filepath.Join(dir, "hooks-allowed.json")}
}

func (a *Allowed) read() []string {
	var list []string
	if data, err := os.ReadFile(a.path); err == nil {
		_ = json.Unmarshal(data, &list)
	}
	return list
}

// Is reports whether project may run hooks.
func (a *Allowed) Is(project string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Contains(a.read(), project)
}

// Set allows or forbids project.
func (a *Allowed) Set(project string, allowed bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	list := slices.DeleteFunc(a.read(), func(p string) bool { return p == project })
	if allowed {
		list = append(list, project)
	}
	data, _ := json.Marshal(list)
	if err := os.MkdirAll(filepath.Dir(a.path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(a.path, data, 0o600)
}

// Tracker turns the session updates it sees into events: one when a
// session's situation changes, not each time it is told again.
type Tracker struct {
	mu   sync.Mutex
	last map[string]string // session ID → the event last fired for it
}

// Event is the event info's session just had, or "". A session that was
// already running when the host started is not a start.
func (t *Tracker) Event(info proto.SessionInfo, now time.Time) string {
	ev := ""
	switch {
	case info.Exited && info.ExitCode != 0:
		ev = EventError
	case info.Exited, info.State == proto.StateDone:
		ev = EventDone
	case info.State == proto.StateWaiting:
		ev = EventInput
	case now.Sub(info.Created) < 10*time.Second:
		ev = EventStart
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.last == nil {
		t.last = map[string]string{}
	}
	if ev == "" || t.last[info.ID] == ev {
		return ""
	}
	t.last[info.ID] = ev
	return ev
}

// Forget drops what the tracker kept of a session that is gone.
func (t *Tracker) Forget(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.last, id)
}

// timeout is how long a hook gets.
const timeout = 30 * time.Second

// Fire runs the hooks of list that ask for event, each on a goroutine of
// its own; one that fails is logged and nothing else: a hook never holds
// an agent up. log gets each outcome.
func Fire(list []proto.ProjectHook, event string, info proto.SessionInfo, log func(msg string, args ...any)) {
	for _, h := range list {
		if h.Event != event {
			continue
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			var err error
			if h.Run != "" {
				err = run(ctx, h.Run, event, info)
			}
			if err == nil && h.Webhook != "" {
				err = post(ctx, h.Webhook, event, info)
			}
			if err != nil {
				log("hook failed", "event", event, "session", info.ID, "err", err)
			}
		}()
	}
}

func run(ctx context.Context, command, event string, info proto.SessionInfo) error {
	name, args := shell(command)
	cmd := quietcmd.CommandContext(ctx, name, args...)
	cmd.Dir = info.Dir
	cmd.Env = append(os.Environ(),
		"HUSTLEX_EVENT="+event, "HUSTLEX_SESSION="+info.ID, "HUSTLEX_SESSION_NAME="+info.Name,
		"HUSTLEX_PROJECT="+info.Project, "HUSTLEX_AGENT="+info.Agent, "HUSTLEX_DIR="+info.Dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w: %s", command, err, bytes.TrimSpace(out))
	}
	return nil
}

func post(ctx context.Context, address, event string, info proto.SessionInfo) error {
	body, _ := json.Marshal(map[string]any{
		"event": event, "session": info.ID, "name": info.Name, "agent": info.Agent,
		"project": info.Project, "dir": info.Dir, "exitCode": info.ExitCode,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, address, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook answered %s", resp.Status)
	}
	return nil
}
