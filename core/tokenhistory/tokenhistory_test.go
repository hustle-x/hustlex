package tokenhistory

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func write(t *testing.T, path string, lines ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, l := range lines {
		if _, err := f.WriteString(l + "\n"); err != nil {
			t.Fatal(err)
		}
	}
}

func byConv(rows []Row) map[string]Row {
	out := map[string]Row{}
	for _, r := range rows {
		k := r.Day + " " + r.Conversation + " " + r.Model
		o := out[k]
		o.Day, o.Conversation, o.Model, o.Dir, o.Title, o.Agent = r.Day, r.Conversation, r.Model, r.Dir, r.Title, r.Agent
		o.Tokens = o.add(r.Tokens)
		out[k] = o
	}
	return out
}

// A streamed answer counts once, a subagent's tokens count for its
// conversation, Claude Code's own records don't count, and what is
// appended later is read on the next scan, from the cache too.
func TestClaudeConversations(t *testing.T) {
	home, cache := t.TempDir(), filepath.Join(t.TempDir(), "cache.json")
	day := time.Now().Format(time.DateOnly)
	at := time.Now().UTC().Format(time.RFC3339)
	conv := filepath.Join(home, ".claude", "projects", "-code-site", "c1.jsonl")
	write(t, conv,
		`{"type":"user","message":{"content":"<command-name>/clear</command-name>"},"cwd":"/code/site","timestamp":"`+at+`"}`,
		`{"type":"user","message":{"content":"Fix the login redirect\nit loops"},"cwd":"/code/site","timestamp":"`+at+`"}`,
		`{"type":"assistant","timestamp":"`+at+`","message":{"id":"m1","model":"claude-opus-5-5","usage":{"input_tokens":10,"cache_read_input_tokens":100,"cache_creation_input_tokens":20,"output_tokens":1}}}`,
		`{"type":"assistant","timestamp":"`+at+`","message":{"id":"m1","model":"claude-opus-5-5","usage":{"input_tokens":10,"cache_read_input_tokens":100,"cache_creation_input_tokens":20,"output_tokens":5}}}`,
		`{"type":"assistant","timestamp":"`+at+`","message":{"id":"x","model":"<synthetic>","usage":{"input_tokens":0,"output_tokens":7}}}`,
	)
	write(t, filepath.Join(home, ".claude", "projects", "-code-site", "c1", "subagents", "agent-a.jsonl"),
		`{"type":"user","isSidechain":true,"message":{"content":"Look around"},"cwd":"/code/site","timestamp":"`+at+`"}`,
		`{"type":"assistant","isSidechain":true,"timestamp":"`+at+`","message":{"id":"s1","model":"claude-haiku-5-5","usage":{"input_tokens":3,"output_tokens":2}}}`,
	)
	h := New(cache)
	if err := h.Scan(Roots(home), time.Now().AddDate(0, 0, -1)); err != nil {
		t.Fatal(err)
	}
	rows := byConv(h.Rows(time.Now().AddDate(0, 0, -1)))
	opus := rows[day+" c1 claude-opus-5-5"]
	if opus.Total() != 131 || opus.Title != "Fix the login redirect" || opus.Dir != "/code/site" || opus.Agent != "claude" {
		t.Fatalf("%+v", opus)
	}
	if rows[day+" c1 claude-haiku-5-5"].Total() != 5 || len(rows) != 2 {
		t.Fatalf("%+v", rows)
	}

	write(t, conv, `{"type":"assistant","timestamp":"`+at+`","message":{"id":"m2","model":"claude-opus-5-5","usage":{"input_tokens":1,"output_tokens":1}}}`)
	h = New(cache) // a restart: from what was kept
	if err := h.Scan(Roots(home), time.Now().AddDate(0, 0, -1)); err != nil {
		t.Fatal(err)
	}
	if got := byConv(h.Rows(time.Now().AddDate(0, 0, -1)))[day+" c1 claude-opus-5-5"].Total(); got != 133 {
		t.Fatalf("after appending: %d", got)
	}
}

// Codex counts running totals; each count adds what is new.
func TestCodexConversations(t *testing.T) {
	home := t.TempDir()
	day := time.Now().Format(time.DateOnly)
	at := time.Now().UTC().Format(time.RFC3339)
	write(t, filepath.Join(home, ".codex", "sessions", "2026", "11", "11", "rollout-1.jsonl"),
		`{"timestamp":"`+at+`","type":"session_meta","payload":{"id":"cx1","cwd":"/code/app"}}`,
		`{"timestamp":"`+at+`","type":"turn_context","payload":{"model":"gpt-5.5-codex"}}`,
		`{"timestamp":"`+at+`","type":"event_msg","payload":{"type":"user_message","message":"Add push notifications"}}`,
		`{"timestamp":"`+at+`","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":60,"output_tokens":10}}}}`,
		`{"timestamp":"`+at+`","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":60,"output_tokens":10}}}}`,
		`{"timestamp":"`+at+`","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":150,"cached_input_tokens":90,"output_tokens":15}}}}`,
	)
	h := New("")
	if err := h.Scan(Roots(home), time.Now().AddDate(0, 0, -1)); err != nil {
		t.Fatal(err)
	}
	r := byConv(h.Rows(time.Now().AddDate(0, 0, -1)))[day+" cx1 gpt-5.5-codex"]
	if r.Input != 60 || r.CacheRead != 90 || r.Output != 15 || r.Title != "Add push notifications" || r.Dir != "/code/app" {
		t.Fatalf("%+v", r)
	}
}
