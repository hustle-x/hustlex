package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hustlex/hustlex/core/agent"
	"github.com/hustlex/hustlex/core/tokenhistory"
	"github.com/hustlex/hustlex/proto"
)

// historyEvery is how often the token history reads what the agents
// appended, at most: the Credits page asks every few seconds.
const historyEvery = 20 * time.Second

// usageHistory reads the agents' conversations in the background, one scan
// at a time; mu guards the rest.
type usageHistory struct {
	mu       sync.Mutex
	h        *tokenhistory.History
	scanned  time.Time // when the last scan finished; zero before the first
	scanning bool
}

// usageHistoryReport is the tokens of the last days, as last read, with a
// scan started when that reading is old.
func (s *Server) usageHistoryReport(days int) (proto.UsageHistory, error) {
	if days <= 0 || days > 366 {
		days = 91
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return proto.UsageHistory{}, err
	}
	u := &s.usageHistory
	u.mu.Lock()
	if u.h == nil {
		u.h = tokenhistory.New(filepath.Join(s.dataDir, "token-history.json"))
	}
	h, reading := u.h, u.scanned.IsZero()
	if !u.scanning && time.Since(u.scanned) > historyEvery {
		u.scanning = true
		go func() {
			// Older files than the longest history asked for stay unread.
			_ = h.Scan(tokenhistory.Roots(home), time.Now().AddDate(0, 0, -366))
			u.mu.Lock()
			u.scanning, u.scanned = false, time.Now()
			u.mu.Unlock()
		}()
	}
	u.mu.Unlock()

	since := time.Now().AddDate(0, 0, -(days - 1))
	out := proto.UsageHistory{Reading: reading, Conversations: []proto.UsageConversation{}, Rows: []proto.UsageRow{}}
	type key struct{ agent, conv string }
	sessions := map[key]record{}
	for _, r := range s.persist.saved() {
		if r.Conversation != "" {
			sessions[key{agent.Base(r.Agent), r.Conversation}] = r
		}
	}
	projects := s.projects.List()
	index := map[key]int{}
	for _, row := range h.Rows(since) {
		k := key{row.Agent, row.Conversation}
		i, ok := index[k]
		if !ok {
			c := proto.UsageConversation{Agent: row.Agent, Account: proto.RefDefault, Project: row.Dir, Title: row.Title}
			if r, ok := sessions[k]; ok {
				c.Session, c.Project = r.ID, r.Project
				if r.Account != "" {
					c.Account = r.Account
				}
			} else {
				// The project the folder is in, the deepest one.
				best := ""
				for _, p := range projects {
					if (row.Dir == p.Path || strings.HasPrefix(row.Dir, p.Path+string(filepath.Separator))) && len(p.Path) > len(best) {
						best = p.Path
					}
				}
				if best != "" {
					c.Project = best
				}
			}
			i = len(out.Conversations)
			index[k] = i
			out.Conversations = append(out.Conversations, c)
		}
		out.Rows = append(out.Rows, proto.UsageRow{Day: row.Day, Conversation: i, Model: row.Model,
			Input: row.Input, CacheRead: row.CacheRead, CacheCreate: row.CacheCreate, Output: row.Output})
	}
	return out, nil
}
