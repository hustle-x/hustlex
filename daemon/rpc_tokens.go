package daemon

import (
	"os"

	"github.com/hustlex/hustlex/core/transcript"
	"github.com/hustlex/hustlex/proto"
)

var tokenHandlers = map[string]handler{
	proto.MethodUsageTokens: with(func(c *conn, _ struct{}) (any, error) { return c.srv.tokenReport() }),
	proto.MethodUsageHistory: with(func(c *conn, p proto.UsageHistoryParams) (any, error) {
		return c.srv.usageHistoryReport(p.Days)
	}),
}

// tokenReport adds up the tokens of every session whose conversation can be
// read: the files its agent wrote since the session began.
func (s *Server) tokenReport() (proto.TokenReport, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return proto.TokenReport{}, err
	}
	var out proto.TokenReport
	for _, sess := range s.sessions.List() {
		info := sess.Info()
		if info.Agent != "claude" && info.Agent != "claude-agent-teams" {
			continue
		}
		var total transcript.Tokens
		models := map[string]int64{}
		for _, found := range transcript.Find(info.Agent, home, info.Dir) {
			if found.Changed.Before(info.Created) {
				break
			}
			byModel, err := s.tokens.Count(found.Path)
			if err != nil {
				continue
			}
			for m, t := range byModel {
				total = total.Add(t)
				models[m] += t.Total()
			}
		}
		if total.Total() == 0 {
			continue
		}
		top, most := "", int64(-1)
		for m, n := range models {
			if n > most {
				top, most = m, n
			}
		}
		out.Sessions = append(out.Sessions, proto.SessionTokens{ID: info.ID, Model: top,
			Input: total.Input, CacheRead: total.CacheRead, CacheCreate: total.CacheCreate, Output: total.Output})
	}
	return out, nil
}
