package daemon

import (
	"errors"
	"sync"
	"time"

	"github.com/hustlex/hustlex/core/agent"
	"github.com/hustlex/hustlex/core/session"
	"github.com/hustlex/hustlex/proto"
)

// A session whose account runs out of a usage limit mid-task goes on with
// the same conversation as another account of its agent with room, unless
// the user turned that off (DaemonSettings.StayOnAccount). The scheduler
// looks at idle agents' screens; one that says its limit ran out is
// confirmed with a fresh reading of the accounts, then started again in
// place, as the other account, on its conversation, told to go on.

const (
	// moveFullAt is how much of a limit counts as run out, in percent;
	// moveRoomUnder how little of each an account must have used to take over.
	moveFullAt    = 99.0
	moveRoomUnder = 80.0
	// moveRetry is how long a session isn't looked at again after a
	// look found no account to go on with, or after it moved: its old
	// screen may still show the limit for a while.
	moveRetry = 10 * time.Minute
	moveQuiet = 30 * time.Minute
)

// moveGo is what a moved agent is told, to pick its task up again.
const moveGo = "Your usage limit ran out mid-task, so this conversation now goes on as another account. Go on where you left off."

// limitMoves belongs to the scheduler's goroutine but for checking, which
// its look goroutines share under mu.
type limitMoves struct {
	mu       sync.Mutex
	checking map[string]bool      // sessions a look is confirming
	tried    map[string]time.Time // sessions looked at and left, when
	seen     map[string]time.Time // sessions' screens as last read (ScreenChanged)
}

// moveAtLimit looks for agents stopped by their account's limit and moves
// each to another account with room.
func (s *Server) moveAtLimit() {
	if s.sched.get().StayOnAccount {
		return
	}
	m := &s.moves
	now := time.Now()
	for _, sess := range s.sessions.List() {
		info, meta := sess.Info(), sess.Meta()
		preset, ok := agent.Find(meta.Agent)
		if !ok || len(preset.Continue) == 0 || info.Exited || meta.Queued || meta.Parent != "" ||
			meta.NotesFor != "" || meta.CanvasFor != "" || meta.Display > 0 || working(info) {
			continue
		}
		if meta.Moved != nil && now.Sub(meta.Moved.At) < moveQuiet {
			continue
		}
		changed := sess.ScreenChanged()
		m.mu.Lock()
		if m.checking == nil {
			m.checking, m.tried, m.seen = map[string]bool{}, map[string]time.Time{}, map[string]time.Time{}
		}
		skip := m.checking[info.ID] || now.Sub(m.tried[info.ID]) < moveRetry || !changed.After(m.seen[info.ID])
		if !skip {
			m.seen[info.ID] = changed
		}
		m.mu.Unlock()
		if skip || !agent.LimitReached(sess.Terminal().Text()) {
			continue
		}
		m.mu.Lock()
		m.checking[info.ID] = true
		m.mu.Unlock()
		go func() {
			err := s.confirmMove(sess)
			m.mu.Lock()
			delete(m.checking, info.ID)
			m.tried[info.ID] = time.Now()
			m.mu.Unlock()
			if err != nil {
				s.logger().Info("a session at its limit stays on its account", "id", info.ID, "why", err)
			}
		}()
	}
}

// errNoRoom says no other account of the agent has room to go on with.
var errNoRoom = errors.New("no other account has room")

// confirmMove reads the accounts afresh and, when session sess's own ran
// out of a limit and another has room, moves sess to it.
func (s *Server) confirmMove(sess *session.Session) error {
	meta := sess.Meta()
	base := agent.Base(meta.Agent)
	ref := meta.Account
	if ref == "" {
		ref = proto.RefDefault
	}
	report := s.freshCredits()
	from, to, limit, ok := pickMove(report.Accounts, base, ref)
	if !ok {
		if limit == "" {
			return errors.New("its account hasn't run out of a limit")
		}
		return errNoRoom
	}
	if sess.Meta().Account != meta.Account || sess.Info().Exited {
		return errors.New("it changed meanwhile")
	}
	to.Ref = accountOfRef(to.Ref)
	return s.moveAccount(sess, to, from, limit)
}

// freshCredits asks for a new reading of the accounts and waits for it,
// up to a while: a reading starts in the background.
func (s *Server) freshCredits() proto.CreditsReport {
	r := s.credits.Report(true)
	for deadline := time.Now().Add(moveReadWait); r.Checking && time.Now().Before(deadline); {
		time.Sleep(500 * time.Millisecond)
		r = s.credits.Report(false)
	}
	return r
}

// moveReadWait bounds the wait for a fresh reading of the accounts.
const moveReadWait = 45 * time.Second

// pickMove finds, among accounts, agent base's account ref and the one to
// go on as: ok when ref ran out of a limit (named limit) and another of
// base's accounts hustleX manages has room, the most of them.
func pickMove(accounts []proto.CreditAccount, base, ref string) (from, to proto.CreditAccount, limit string, ok bool) {
	found := false
	for _, a := range accounts {
		if a.Agent != base || a.Ref != ref {
			continue
		}
		for _, w := range a.Windows {
			if w.Used >= moveFullAt {
				from, limit, found = a, w.LimitName(), true
				break
			}
		}
	}
	if !found {
		return from, to, "", false
	}
	best := moveRoomUnder
	for _, a := range accounts {
		if a.Agent != base || a.Ref == ref || a.Ref == "" || a.Error != "" || len(a.Windows) == 0 {
			continue
		}
		used := 0.0
		for _, w := range a.Windows {
			used = max(used, w.Used)
		}
		if used < best {
			to, best, ok = a, used, true
		}
	}
	return from, to, limit, ok
}

// accountOfRef is a credits Ref as a session names its account: "" for
// the agent's own.
func accountOfRef(ref string) string {
	if ref == proto.RefDefault {
		return ""
	}
	return ref
}

// moveAccount starts session sess's agent again in place as account
// to.Ref (a session's Account), on its conversation, told to go on; from
// is the account it ran out on, limit the limit.
func (s *Server) moveAccount(sess *session.Session, to, from proto.CreditAccount, limit string) error {
	info, meta := sess.Info(), sess.Meta()
	preset, ok := agent.Find(meta.Agent)
	if !ok || len(preset.Continue) == 0 {
		return errors.New("its agent can't go on with a conversation")
	}
	s.noteConversation(sess)
	conv := s.persist.conversation(info.ID)
	if conv == "" {
		return errors.New("its conversation isn't recorded yet")
	}
	l := agentLaunch(preset, meta.SkipPermissions, "", true, s.chatOf(meta.NotesFor, meta.CanvasFor, s.guideFor(meta)))
	l.Resume = conv
	command := thenShell(preset.Argv(l))
	meta.Moved = &proto.AccountMove{From: meta.Account, To: to.Ref, Limit: limit, At: time.Now().UTC()}
	meta.Account, meta.Restart, meta.Restored = to.Ref, command, false
	env, err := s.agentEnv(meta)
	if err != nil {
		return err
	}
	spec := session.Spec{
		ID: info.ID, Dir: info.Dir, Created: info.Created, History: sess.History(), Command: command, Meta: meta,
		Env:    append(append(append(s.sessionEnv(), env...), s.trafficEnvFor(meta.Parent, info.ID)...), s.vaultEnvFor(info.ID, meta.Project, meta.Creds)...),
		Notice: "⟳ " + accountWho(from) + " hit its " + limit + ". Continued on " + accountWho(to) + ", same conversation.",
	}
	next, err := s.sessions.Replace(spec)
	if err != nil {
		return err
	}
	s.began(next, command, moveGo)
	s.logger().Info("moved a session at its limit to another account", "id", info.ID, "limit", limit)
	return nil
}

// accountWho is what a notice calls an account: its email, else its plan.
func accountWho(a proto.CreditAccount) string {
	switch {
	case a.Account != "":
		return a.Account
	case a.Plan != "":
		return "the " + a.Plan + " account"
	}
	return "its other account"
}
