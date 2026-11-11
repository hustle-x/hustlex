package daemon

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/hustlex/hustlex/core/agent"
	"github.com/hustlex/hustlex/core/screenmenu"
	"github.com/hustlex/hustlex/proto"
)

// phoneAsk is what a session waits on, as the phone shows it.
type phoneAsk struct {
	Session  string        `json:"session"`
	Name     string        `json:"name"`
	Agent    string        `json:"agent"`
	Question string        `json:"question"`
	Detail   []string      `json:"detail,omitempty"`
	Choices  []phoneChoice `json:"choices,omitempty"`
	Project  string        `json:"project,omitempty"`
}

type phoneChoice struct {
	Label string `json:"label"`
	Key   string `json:"key"`
}

// phoneSessionRow is a session as the phone lists it, the way the app's
// session list shows it.
type phoneSessionRow struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Agent   string `json:"agent"`
	Project string `json:"project"`
	Path    string `json:"path,omitempty"` // its project's folder, which groups it
	Status  string `json:"status"`
	// Label is the row's second line: its state and where it runs, or what
	// its agent says it is at; Hollow draws its dot as a ring.
	Label   string `json:"label"`
	Hollow  bool   `json:"hollow,omitempty"`
	Note    string `json:"note,omitempty"`
	Account string `json:"account,omitempty"`
	Ref     string `json:"ref,omitempty"`
	RSS     uint64 `json:"rss,omitempty"` // memory of everything it runs
}

// phoneProject is a project as the phone groups sessions by it, in the
// user's order.
type phoneProject struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

type phoneStateView struct {
	Host     string                `json:"host"`
	Inbox    []phoneAsk            `json:"inbox"`
	Projects []phoneProject        `json:"projects"`
	Sessions []phoneSessionRow     `json:"sessions"`
	Accounts []proto.CreditAccount `json:"accounts"`
}

// phoneSessions are the sessions the user started agents in.
func (s *Server) phoneSessions() []proto.SessionInfo {
	var out []proto.SessionInfo
	for _, sess := range s.sessions.List() {
		info := sess.Info()
		if _, ok := agent.Find(info.Agent); !ok || info.Parent != "" || info.NotesFor != "" || info.CanvasFor != "" {
			continue
		}
		out = append(out, info)
	}
	return out
}

func (s *Server) phoneState() phoneStateView {
	host, _ := os.Hostname()
	host = strings.TrimSuffix(host, ".local")
	v := phoneStateView{Host: host, Inbox: []phoneAsk{}, Projects: []phoneProject{}, Sessions: []phoneSessionRow{}, Accounts: s.credits.Report(false).Accounts}
	emails := map[string]string{}
	for _, a := range v.Accounts {
		emails[a.Agent+" "+a.Ref] = a.Account
	}
	names := map[string]string{}
	for _, p := range s.projects.List() {
		v.Projects = append(v.Projects, phoneProject{Path: p.Path, Name: p.Name})
		names[p.Path] = p.Name
	}
	rss := map[string]uint64{}
	for _, u := range s.stats().Sessions {
		rss[u.ID] = u.RSS
	}
	for _, info := range s.phoneSessions() {
		name := phoneName(info)
		ref := info.Account
		if ref == "" {
			ref = proto.RefDefault
		}
		path := info.Project
		if path == "" {
			path = info.Dir
		}
		project := names[path]
		if project == "" {
			project = projectName(info.Project)
		}
		label, hollow := phoneLabel(info, name)
		v.Sessions = append(v.Sessions, phoneSessionRow{ID: info.ID, Name: name, Agent: agent.Base(info.Agent), Project: project, Path: path,
			Status: phoneStatusOf(info), Label: label, Hollow: hollow, Note: info.Note, Account: emails[agent.Base(info.Agent)+" "+ref], Ref: ref, RSS: rss[info.ID]})
		if info.State != proto.StateWaiting || info.Exited {
			continue
		}
		ask := phoneAsk{Session: info.ID, Name: name, Agent: agent.Base(info.Agent), Question: "Needs your answer", Project: project}
		if sess, err := s.sessions.Get(info.ID); err == nil {
			if m, ok := screenmenu.Parse(sess.Terminal().Text()); ok {
				ask.Question, ask.Detail = m.Question, m.Detail
				for _, c := range m.Choices {
					ask.Choices = append(ask.Choices, phoneChoice{c.Label, c.Key})
				}
			}
		}
		v.Inbox = append(v.Inbox, ask)
	}
	return v
}

// projectName is a project folder's name, "" for none.
func projectName(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Base(path)
}

// phoneName is what the phone calls a session, as the app does: the
// user's name for it, else what its agent titles it, else its first prompt.
func phoneName(info proto.SessionInfo) string {
	if info.Name != "" {
		return info.Name
	}
	if t := cleanTitle(info.Title); t != "" && !genericTitle(info, t) {
		return t
	}
	switch {
	case strings.TrimSpace(info.Prompt) != "":
		line, _, _ := strings.Cut(strings.TrimSpace(info.Prompt), "\n")
		if r := []rune(line); len(r) > 60 {
			line = string(r[:60]) + "…"
		}
		return line
	case info.Branch != "" && info.Worktree:
		return info.Branch
	}
	if p, ok := agent.Find(info.Agent); ok {
		return p.Name
	}
	return info.Agent
}

// cleanTitle drops the spinners and marks agents put before their title.
func cleanTitle(t string) string {
	return strings.TrimSpace(strings.TrimLeftFunc(t, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}))
}

// genericTitle reports titles that say nothing about the task: the agent's
// own name, or a shell's "user@host: dir".
func genericTitle(info proto.SessionInfo, t string) bool {
	lower := strings.ToLower(t)
	if p, ok := agent.Find(info.Agent); ok && (lower == strings.ToLower(p.Name) || lower == strings.ToLower(p.Command)) {
		return true
	}
	for _, generic := range []string{info.Agent, "claude code", "claude", "codex", "openai codex", "opencode", "gemini", "gemini cli", "kiro", "kiro cli", "kiro-cli"} {
		if lower == generic {
			return true
		}
	}
	if at, colon := strings.Index(t, "@"), strings.Index(t, ":"); at > 0 && colon > at {
		return true
	}
	return strings.HasPrefix(t, "~") || strings.HasPrefix(t, "/")
}

// phoneLabel is a session row's second line as the app's list words it,
// and whether its dot is a ring.
func phoneLabel(info proto.SessionInfo, name string) (string, bool) {
	label, hollow := "idle", true
	switch {
	case info.Hibernated:
		label = "hibernated"
		if h := info.HibernatedHow; h != nil {
			if h.Manual {
				label = "hibernated by you"
			}
			since := h.Since.Local()
			if time.Since(since) > 20*time.Hour {
				label += " · since " + since.Format("2 Jan")
			} else {
				label += " · since " + since.Format("15:04")
			}
		}
		return label, true
	case info.Exited && info.Restored && info.ExitCode == -1:
		label, hollow = "stopped", false
	case info.Exited && info.ExitCode == 0:
		label, hollow = "done", false
	case info.Exited:
		label, hollow = "exited", false
	case info.Queued:
		label = "queued"
	case info.State == proto.StateWaiting:
		label, hollow = "needs input", false
	case working(info):
		label, hollow = "working", false
	case info.State == proto.StateDone:
		label = "done"
	}
	switch {
	case info.Branch != "" && info.Branch != name:
		label += " · " + info.Branch
	case !info.Worktree && info.Project != "":
		label += " · project folder"
	}
	if info.Note != "" && !info.Exited {
		line, _, _ := strings.Cut(info.Note, "\n")
		if r := []rune(line); len(r) > 60 {
			line = string(r[:60]) + "…"
		}
		label = line
	}
	return label, hollow
}

func phoneStatusOf(info proto.SessionInfo) string {
	switch {
	case info.Hibernated:
		return "hibernated"
	case info.Exited:
		return "ended"
	case info.Queued:
		return "queued"
	case info.State == proto.StateWaiting:
		return "needs you"
	case working(info):
		return "working"
	case info.State == proto.StateDone:
		return "done"
	}
	return "idle"
}

// phoneSessionView is a session as the phone opens it: its last messages,
// and the account it runs as with that account's limits.
type phoneSessionView struct {
	phoneSessionRow
	Plan     string               `json:"plan,omitempty"`
	Windows  []proto.CreditWindow `json:"windows,omitempty"`
	Messages []phoneMessage       `json:"messages"`
}

type phoneMessage struct {
	Kind   string `json:"kind"` // user, assistant, tool
	Text   string `json:"text"`
	Detail string `json:"detail,omitempty"`
	Failed bool   `json:"failed,omitempty"`
}

func (s *Server) phoneSession(id string) (phoneSessionView, error) {
	var v phoneSessionView
	state := s.phoneState()
	found := false
	for _, row := range state.Sessions {
		if row.ID == id {
			v.phoneSessionRow, found = row, true
		}
	}
	if !found {
		return v, errors.New("no such session")
	}
	for _, a := range state.Accounts {
		if a.Agent == v.Agent && a.Ref == v.Ref {
			v.Plan, v.Windows = a.Plan, a.Windows
		}
	}
	v.Messages = []phoneMessage{}
	if t, err := s.transcript(proto.TranscriptParams{ID: id, Tail: 80}); err == nil {
		for _, e := range t.Entries {
			switch e.Kind {
			case proto.TranscriptUser, proto.TranscriptAssistant:
				if text := strings.TrimSpace(e.Text); text != "" {
					v.Messages = append(v.Messages, phoneMessage{Kind: e.Kind, Text: text})
				}
			case proto.TranscriptTool:
				v.Messages = append(v.Messages, phoneMessage{Kind: e.Kind, Text: e.Text, Detail: e.Detail, Failed: e.Failed})
			}
		}
	}
	return v, nil
}
