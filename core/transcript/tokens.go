package transcript

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sync"
)

// Tokens are the tokens an agent's answers used, as it records them.
type Tokens struct {
	Input, CacheRead, CacheCreate, Output int64
}

// Add is t and u together.
func (t Tokens) Add(u Tokens) Tokens {
	return Tokens{t.Input + u.Input, t.CacheRead + u.CacheRead, t.CacheCreate + u.CacheCreate, t.Output + u.Output}
}

// Total is every kind together.
func (t Tokens) Total() int64 { return t.Input + t.CacheRead + t.CacheCreate + t.Output }

// TokenCounter adds up the tokens of Claude Code conversations. It reads
// each file once and then only what was appended, so asking again every few
// seconds is cheap. Safe for concurrent use.
type TokenCounter struct {
	mu    sync.Mutex
	files map[string]*countedFile
}

// countedFile is how far a file was read and what it came to. Streaming
// writes one answer as several records with one message ID: it counts once.
type countedFile struct {
	offset  int64
	seen    map[string]bool
	byModel map[string]Tokens
	rest    []byte
}

type tokenRecord struct {
	Type    string `json:"type"`
	UUID    string `json:"uuid"`
	Message struct {
		ID    string `json:"id"`
		Model string `json:"model"`
		Usage *struct {
			Input       int64 `json:"input_tokens"`
			CacheRead   int64 `json:"cache_read_input_tokens"`
			CacheCreate int64 `json:"cache_creation_input_tokens"`
			Output      int64 `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

// Count is the tokens of the conversation at path, by model.
func (c *TokenCounter) Count(path string) (map[string]Tokens, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.files == nil {
		c.files = map[string]*countedFile{}
	}
	f := c.files[path]
	if f == nil {
		f = &countedFile{seen: map[string]bool{}, byModel: map[string]Tokens{}}
		c.files[path] = f
	}
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	info, err := fh.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() < f.offset { // rewritten: start over
		*f = countedFile{seen: map[string]bool{}, byModel: map[string]Tokens{}}
	}
	if _, err := fh.Seek(f.offset, io.SeekStart); err != nil {
		return nil, err
	}
	br := bufio.NewReaderSize(fh, 64<<10)
	for {
		line, err := br.ReadBytes('\n')
		f.offset += int64(len(line))
		if errors.Is(err, io.EOF) {
			f.rest = append(f.rest, line...)
			break
		}
		if err != nil {
			return nil, err
		}
		if len(f.rest) > 0 {
			line = append(f.rest, line...)
			f.rest = nil
		}
		f.add(bytes.TrimSpace(line))
	}
	out := make(map[string]Tokens, len(f.byModel))
	for m, t := range f.byModel {
		out[m] = t
	}
	return out, nil
}

func (f *countedFile) add(line []byte) {
	if len(line) == 0 {
		return
	}
	var rec tokenRecord
	if json.Unmarshal(line, &rec) != nil || rec.Type != "assistant" || rec.Message.Usage == nil {
		return
	}
	id := rec.Message.ID
	if id == "" {
		id = rec.UUID
	}
	if id != "" {
		if f.seen[id] {
			// The last record of an answer has its final output count.
			return
		}
		f.seen[id] = true
	}
	u := rec.Message.Usage
	model := rec.Message.Model
	if model == "" {
		model = "unknown"
	}
	f.byModel[model] = f.byModel[model].Add(Tokens{u.Input, u.CacheRead, u.CacheCreate, u.Output})
}
