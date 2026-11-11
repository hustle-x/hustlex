package daemon

import (
	"time"

	"github.com/hustlex/hustlex/core/agent"
	"github.com/hustlex/hustlex/core/session"
	"github.com/hustlex/hustlex/proto"
)

// A session can be queued until a time, when one of its account's limits
// resets (SessionCreateParams.StartAt): it waits in the scheduler's queue
// like one held for MaxAgents, and its record keeps it over restarts.

// startsWhen says when a queued session starts: the time today, else the
// day and time.
func startsWhen(at, now time.Time) string {
	at, now = at.Local(), now.Local()
	if y, m, d := at.Date(); y == now.Year() && m == now.Month() && d == now.Day() {
		return "at " + at.Format("15:04")
	}
	return at.Format("Mon 15:04")
}

// restoreHeld queues again a session record r held until a time: what
// it runs is worked out anew, with this daemon's hooks and guide.
func (s *Server) restoreHeld(r record) {
	p := proto.SessionCreateParams{Agent: r.Agent, Prompt: r.Prompt, SkipPermissions: r.Skip, Account: r.Account}
	meta := session.Meta{
		Name: r.Name, Prompt: r.Prompt, Project: r.Project, Agent: r.Agent, Branch: r.Branch, Worktree: r.Worktree,
		Budget: time.Duration(r.Budget) * time.Second, Group: r.Group, Account: r.Account, Features: r.Features, Creds: r.Creds,
		NoAutoHibernate: r.NoAutoHibernate, StartAt: r.StartAt, Named: r.Named, Earlier: r.Earlier,
	}
	launch, err := launchFor(p, s.chatOf("", "", s.guideFor(meta)))
	if err == nil {
		var acc []string
		if acc, err = s.accounts.Env(agent.Base(r.Agent), r.Account); err == nil {
			launch.env = append(launch.env, acc...)
		}
	}
	if err != nil {
		s.logger().Warn("could not restore a queued session", "id", r.ID, "err", err)
		s.persist.remove(r.ID)
		return
	}
	meta.Restart, meta.Relaunch, meta.SkipPermissions = launch.restart, launch.relaunch, launch.skip
	spec := session.Spec{
		ID: r.ID, Created: r.Created, Command: launch.command, Dir: r.Dir, Meta: meta,
		Env: append(append(launch.env, s.sessionEnv()...), s.vaultEnvFor(r.ID, r.Project, r.Creds)...),
	}
	if _, err := s.enqueue(spec, launch.command, launch.typed); err != nil {
		s.logger().Warn("could not restore a queued session", "id", r.ID, "err", err)
	}
}
