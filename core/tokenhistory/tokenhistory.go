// Package tokenhistory adds up the tokens agents' conversations used, by
// day, model and conversation, from the files the agents keep: Claude Code's
// in ~/.claude/projects, Codex's in ~/.codex/sessions. Each file is read
// once, then only what was appended; what was read is kept in a cache file,
// so a restart doesn't read gigabytes again.
package tokenhistory

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Tokens are the tokens of answers, by kind.
type Tokens struct {
	Input       int64 `json:"i,omitempty"`
	CacheRead   int64 `json:"r,omitempty"`
	CacheCreate int64 `json:"w,omitempty"`
	Output      int64 `json:"o,omitempty"`
}

func (t Tokens) add(u Tokens) Tokens {
	return Tokens{t.Input + u.Input, t.CacheRead + u.CacheRead, t.CacheCreate + u.CacheCreate, t.Output + u.Output}
}

// Total is every kind together.
func (t Tokens) Total() int64 { return t.Input + t.CacheRead + t.CacheCreate + t.Output }

// Root is a folder of an agent's conversation files.
type Root struct {
	Agent string // "claude" or "codex"
	Dir   string
}

// Roots are where the agents of the user whose home is home keep their
// conversations. Other accounts' homes link these folders, so they are
// read once.
func Roots(home string) []Root {
	return []Root{
		{Agent: "claude", Dir: filepath.Join(home, ".claude", "projects")},
		{Agent: "codex", Dir: filepath.Join(home, ".codex", "sessions")},
	}
}

// Row is a conversation's tokens of one model on one day.
type Row struct {
	Day          string // local date, 2006-01-02
	Agent        string
	Conversation string // the ID the agent resumes it by
	Dir          string // the folder it ran in
	Title        string // its first prompt's first line
	Model        string
	Tokens
}

// History holds what was read. Safe for concurrent use; Scan runs one at a
// time.
type History struct {
	cache string // where what was read is kept; "" for nowhere

	scan  sync.Mutex // one scan at a time
	mu    sync.Mutex
	files map[string]*file
}

// file is how far a conversation file was read and what it came to.
type file struct {
	Agent string `json:"agent"`
	Read  int64  `json:"read"` // bytes counted, up to the end of the last whole line
	Conv  string `json:"conv,omitempty"`
	Dir   string `json:"dir,omitempty"`
	Title string `json:"title,omitempty"`
	// Model is the model a Codex conversation is on; Totals what its
	// token counts last came to, as Codex counts them as running totals.
	Model  string `json:"model,omitempty"`
	Totals Tokens `json:"totals,omitzero"`
	// Last are the IDs of Claude Code's last answers: one answer streams as
	// several records with one ID, and counts once.
	Last []string                     `json:"last,omitempty"`
	Days map[string]map[string]Tokens `json:"days,omitempty"` // day → model → tokens
}

// New is a History keeping what it reads in the file cache ("" for
// nowhere), starting from what it kept there.
func New(cache string) *History {
	h := &History{cache: cache, files: map[string]*file{}}
	if cache != "" {
		if data, err := os.ReadFile(cache); err == nil {
			_ = json.Unmarshal(data, &h.files)
		}
	}
	return h
}

// Scan reads what roots' files have that wasn't read yet: files changed
// since since, the others as they are. Then it keeps the reading in the
// cache, forgetting files gone or older than since.
func (h *History) Scan(roots []Root, since time.Time) error {
	h.scan.Lock()
	defer h.scan.Unlock()
	seen := map[string]bool{}
	for _, root := range roots {
		err := filepath.WalkDir(root.Dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if path == root.Dir {
					return filepath.SkipDir // the agent isn't used here
				}
				return nil
			}
			if d.IsDir() || filepath.Ext(path) != ".jsonl" {
				return nil
			}
			info, err := d.Info()
			if err != nil || info.ModTime().Before(since) {
				return nil
			}
			seen[path] = true
			h.read(root.Agent, path, info.Size())
			return nil
		})
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("tokenhistory: %s: %w", root.Dir, err)
		}
	}
	h.mu.Lock()
	for path := range h.files {
		if !seen[path] {
			delete(h.files, path)
		}
	}
	h.mu.Unlock()
	return h.save()
}

// read counts what file path has past what was read of it.
func (h *History) read(agent, path string, size int64) {
	h.mu.Lock()
	f := h.files[path]
	if f != nil && f.Read == size {
		h.mu.Unlock()
		return
	}
	var work file
	if f != nil && f.Agent == agent && f.Read < size {
		work = f.clone()
	} else {
		work = file{Agent: agent} // new, or rewritten: from the start
	}
	h.mu.Unlock()

	if work.Conv == "" {
		work.Conv = conversationOf(agent, path)
	}
	fh, err := os.Open(path)
	if err != nil {
		return
	}
	defer fh.Close()
	if _, err := fh.Seek(work.Read, io.SeekStart); err != nil {
		return
	}
	br := bufio.NewReaderSize(fh, 256<<10)
	for {
		line, err := br.ReadBytes('\n')
		if len(line) == 0 || line[len(line)-1] != '\n' {
			break // the rest is being written: read it next time
		}
		work.Read += int64(len(line))
		switch agent {
		case "claude":
			work.claudeLine(line)
		case "codex":
			work.codexLine(line)
		}
		if err != nil {
			break
		}
	}
	h.mu.Lock()
	h.files[path] = &work
	h.mu.Unlock()
}

func (f *file) clone() file {
	c := *f
	c.Last = append([]string(nil), f.Last...)
	c.Days = make(map[string]map[string]Tokens, len(f.Days))
	for d, models := range f.Days {
		m := make(map[string]Tokens, len(models))
		for k, v := range models {
			m[k] = v
		}
		c.Days[d] = m
	}
	return c
}

func (f *file) count(at time.Time, model string, t Tokens) {
	if at.IsZero() || t.Total() == 0 {
		return
	}
	if model == "" {
		model = "unknown"
	}
	day := at.Local().Format(time.DateOnly)
	if f.Days == nil {
		f.Days = map[string]map[string]Tokens{}
	}
	if f.Days[day] == nil {
		f.Days[day] = map[string]Tokens{}
	}
	f.Days[day][model] = f.Days[day][model].add(t)
}

// conversationOf is the ID of the conversation file path belongs to:
// Claude Code keeps a subagent's next to its conversation, in
// <conversation>/subagents.
func conversationOf(agent, path string) string {
	name := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	if agent == "claude" && filepath.Base(filepath.Dir(path)) == "subagents" {
		return filepath.Base(filepath.Dir(filepath.Dir(path)))
	}
	return name
}

type claudeLine struct {
	Type        string    `json:"type"`
	UUID        string    `json:"uuid"`
	Timestamp   time.Time `json:"timestamp"`
	Cwd         string    `json:"cwd"`
	IsMeta      bool      `json:"isMeta"`
	IsSidechain bool      `json:"isSidechain"`
	Message     struct {
		ID      string          `json:"id"`
		Model   string          `json:"model"`
		Content json.RawMessage `json:"content"`
		Usage   *struct {
			Input       int64 `json:"input_tokens"`
			CacheRead   int64 `json:"cache_read_input_tokens"`
			CacheCreate int64 `json:"cache_creation_input_tokens"`
			Output      int64 `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

// lastIDs is how many answer IDs a Claude Code file remembers: an answer's
// records come one after another.
const lastIDs = 16

func (f *file) claudeLine(line []byte) {
	answer := bytes.Contains(line, []byte(`"usage"`))
	prompt := f.Title == "" && bytes.Contains(line, []byte(`"type":"user"`))
	if !answer && !prompt && (f.Dir != "" || !bytes.Contains(line, []byte(`"cwd"`))) {
		return
	}
	var rec claudeLine
	if json.Unmarshal(line, &rec) != nil {
		return
	}
	if f.Dir == "" && rec.Cwd != "" {
		f.Dir = rec.Cwd
	}
	switch {
	case rec.Type == "assistant" && rec.Message.Usage != nil:
		if rec.Message.Model == "<synthetic>" {
			return // written by Claude Code, not a model
		}
		id := rec.Message.ID
		if id == "" {
			id = rec.UUID
		}
		for _, l := range f.Last {
			if l == id {
				return
			}
		}
		f.Last = append(f.Last, id)
		if len(f.Last) > lastIDs {
			f.Last = f.Last[len(f.Last)-lastIDs:]
		}
		u := rec.Message.Usage
		f.count(rec.Timestamp, rec.Message.Model, Tokens{u.Input, u.CacheRead, u.CacheCreate, u.Output})
	case rec.Type == "user" && f.Title == "" && !rec.IsMeta && !rec.IsSidechain:
		var text string
		if json.Unmarshal(rec.Message.Content, &text) == nil {
			f.Title = titleOf(text)
		}
	}
}

type codexLine struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Payload   struct {
		Type    string `json:"type"`
		ID      string `json:"id"`
		Cwd     string `json:"cwd"`
		Model   string `json:"model"`
		Message string `json:"message"`
		Info    *struct {
			Total struct {
				Input  int64 `json:"input_tokens"`
				Cached int64 `json:"cached_input_tokens"`
				Output int64 `json:"output_tokens"`
			} `json:"total_token_usage"`
		} `json:"info"`
	} `json:"payload"`
}

func (f *file) codexLine(line []byte) {
	var rec codexLine
	if json.Unmarshal(line, &rec) != nil {
		return
	}
	p := rec.Payload
	switch {
	case rec.Type == "session_meta":
		if p.ID != "" {
			f.Conv = p.ID
		}
		if p.Cwd != "" {
			f.Dir = p.Cwd
		}
	case rec.Type == "turn_context" && p.Model != "":
		f.Model = p.Model
	case rec.Type == "event_msg" && p.Type == "user_message" && f.Title == "":
		f.Title = titleOf(p.Message)
	case rec.Type == "event_msg" && p.Type == "token_count" && p.Info != nil:
		// Codex counts running totals, input with what was cached.
		t := p.Info.Total
		total := Tokens{Input: t.Input - t.Cached, CacheRead: t.Cached, Output: t.Output}
		d := Tokens{total.Input - f.Totals.Input, total.CacheRead - f.Totals.CacheRead, 0, total.Output - f.Totals.Output}
		if d.Input < 0 || d.CacheRead < 0 || d.Output < 0 {
			d = total // started over
		}
		f.Totals = total
		f.count(rec.Timestamp, f.Model, d)
	}
}

// titleOf is a prompt's first line, or "" for what Claude Code wrote
// itself (commands, caveats).
func titleOf(text string) string {
	text = strings.TrimSpace(text)
	if text == "" || strings.HasPrefix(text, "<") || strings.HasPrefix(text, "Caveat:") {
		return ""
	}
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		text = text[:i]
	}
	if r := []rune(text); len(r) > 80 {
		text = string(r[:79]) + "…"
	}
	return text
}

// Rows are the tokens of days from since on, a row per day, conversation
// and model.
func (h *History) Rows(since time.Time) []Row {
	from := since.Local().Format(time.DateOnly)
	type key struct{ day, agent, conv, model string }
	sum := map[key]Tokens{}
	about := map[[2]string]*file{}
	h.mu.Lock()
	for _, f := range h.files {
		ck := [2]string{f.Agent, f.Conv}
		if a := about[ck]; a == nil || a.Title == "" && f.Title != "" {
			about[ck] = f
		}
		for day, models := range f.Days {
			if day < from {
				continue
			}
			for model, t := range models {
				k := key{day, f.Agent, f.Conv, model}
				sum[k] = sum[k].add(t)
			}
		}
	}
	h.mu.Unlock()
	out := make([]Row, 0, len(sum))
	for k, t := range sum {
		a := about[[2]string{k.agent, k.conv}]
		out = append(out, Row{Day: k.day, Agent: k.agent, Conversation: k.conv, Dir: a.Dir, Title: a.Title, Model: k.model, Tokens: t})
	}
	return out
}

func (h *History) save() error {
	if h.cache == "" {
		return nil
	}
	h.mu.Lock()
	data, err := json.Marshal(h.files)
	h.mu.Unlock()
	if err != nil {
		return err
	}
	tmp := h.cache + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("tokenhistory: %w", err)
	}
	return os.Rename(tmp, h.cache)
}
