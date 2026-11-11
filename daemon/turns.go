package daemon

import (
	"errors"
	"sync"

	"github.com/hustlex/hustlex/core/agent"
	"github.com/hustlex/hustlex/core/turnsnap"
	"github.com/hustlex/hustlex/proto"
)

// An agent's turn can be undone: when its hooks say it starts working,
// the session's folder is snapshotted (turnsnap); when they say it is
// done, again, and what changed between them is the turn's, to put back
// and forth. Snapshots of a session are taken one after another, off the
// update callback, in the order its states came.

type turnSnaps struct {
	mu    sync.Mutex
	snaps map[string]*turnSnap // session ID → its snapshots
}

// turnSnap is one session's: its queue of work, run by one goroutine at a
// time, and the trees it took (owned by that goroutine).
type turnSnap struct {
	mu      sync.Mutex
	queue   []func()
	running bool
	inTurn  bool   // its agent is in a turn, as its states said
	dir     string // its folder
	overlap bool   // another agent worked in its folder during the turn

	before, after, redo string // tree IDs; redo while undone
}

func (t *turnSnaps) of(id string) *turnSnap {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.snaps == nil {
		t.snaps = map[string]*turnSnap{}
	}
	ts := t.snaps[id]
	if ts == nil {
		ts = &turnSnap{}
		t.snaps[id] = ts
	}
	return ts
}

// markOverlap notes that an agent other than session id works in folder
// dir: the turns there now share their files with it, so none is undone.
func (t *turnSnaps) markOverlap(dir, id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for other, ts := range t.snaps {
		if other == id {
			continue
		}
		ts.mu.Lock()
		if ts.inTurn && ts.dir == dir {
			ts.overlap = true
		}
		ts.mu.Unlock()
	}
}

func (t *turnSnaps) forget(id string) {
	t.mu.Lock()
	delete(t.snaps, id)
	t.mu.Unlock()
}

// do runs job after the session's earlier ones.
func (ts *turnSnap) do(job func()) {
	ts.mu.Lock()
	ts.queue = append(ts.queue, job)
	if ts.running {
		ts.mu.Unlock()
		return
	}
	ts.running = true
	ts.mu.Unlock()
	go func() {
		for {
			ts.mu.Lock()
			if len(ts.queue) == 0 {
				ts.running = false
				ts.mu.Unlock()
				return
			}
			job := ts.queue[0]
			ts.queue = ts.queue[1:]
			ts.mu.Unlock()
			job()
		}
	}()
}

// observeTurn notes a session's state: a turn starting or ending
// snapshots its folder. It runs in the sessions' update callback.
func (s *Server) observeTurn(info proto.SessionInfo) {
	if _, ok := agent.Find(info.Agent); !ok || info.Parent != "" || info.NotesFor != "" || info.CanvasFor != "" || info.Queued {
		return
	}
	if info.State == proto.StateWorking {
		s.turns.markOverlap(info.Dir, info.ID)
	}
	others := info.State == proto.StateWorking && s.othersWorkIn(info.Dir, info.ID)
	ts := s.turns.of(info.ID)
	ts.mu.Lock()
	began := info.State == proto.StateWorking && !ts.inTurn
	ended := ts.inTurn && (info.State == proto.StateDone || info.Exited)
	shared := ts.overlap
	switch {
	case began:
		ts.inTurn, ts.dir, ts.overlap = true, info.Dir, others
	case ended:
		ts.inTurn = false
	}
	ts.mu.Unlock()
	id, dir := info.ID, info.Dir
	switch {
	case began:
		ts.do(func() {
			tree, err := turnsnap.Take(dir)
			if err != nil {
				tree = ""
			}
			ts.before, ts.after, ts.redo = tree, "", ""
			s.setTurn(id, nil)
		})
	case ended:
		ts.do(func() {
			if ts.before == "" {
				return
			}
			tree, err := turnsnap.Take(dir)
			if err != nil {
				return
			}
			ts.after = tree
			files, _ := turnsnap.Changed(dir, ts.before, tree)
			if shared {
				ts.before = "" // its changes mix with another agent's
				return
			}
			if len(files) > 0 {
				s.setTurn(id, &proto.TurnChange{Files: len(files)})
			}
		})
	}
}

func (s *Server) setTurn(id string, t *proto.TurnChange) {
	if sess, err := s.sessions.Get(id); err == nil {
		s.sessions.SetTurn(sess, t)
	}
}

// undoTurn puts session p.ID's files back as before its agent's last
// turn, or forth again with Redo.
func (s *Server) undoTurn(p proto.SessionUndoParams) (proto.SessionUndoResult, error) {
	sess, err := s.sessions.Get(p.ID)
	if err != nil {
		return proto.SessionUndoResult{}, err
	}
	info := sess.Info()
	if working(info) {
		return proto.SessionUndoResult{}, errors.New("the agent is still on its turn: undo once it's done")
	}
	if s.othersWorkIn(info.Dir, info.ID) {
		return proto.SessionUndoResult{}, errors.New("another agent is working in this folder: undo once it's done")
	}
	ts := s.turns.of(p.ID)
	type result struct {
		files []string
		err   error
	}
	done := make(chan result, 1)
	ts.do(func() {
		var r result
		switch {
		case !p.Redo && ts.before == "":
			r.err = errors.New("there is no turn to undo")
		case p.Redo && ts.redo == "":
			r.err = errors.New("there is nothing to redo")
		default:
			now, err := turnsnap.Take(info.Dir)
			if err != nil {
				r.err = err
				break
			}
			to := ts.before
			if p.Redo {
				to = ts.redo
			}
			if r.files, r.err = turnsnap.Restore(info.Dir, now, to); r.err != nil {
				break
			}
			if p.Redo {
				ts.redo = ""
			} else {
				ts.redo = now
			}
			s.setTurn(p.ID, &proto.TurnChange{Files: len(r.files), Undone: !p.Redo})
		}
		done <- r
	})
	r := <-done
	return proto.SessionUndoResult{Files: r.files}, r.err
}

// othersWorkIn reports whether an agent other than session id works in
// folder dir now.
func (s *Server) othersWorkIn(dir, id string) bool {
	for _, sess := range s.sessions.List() {
		if sess.ID() == id {
			continue
		}
		info := sess.Info()
		if info.Dir == dir && working(info) {
			if _, ok := agent.Find(info.Agent); ok {
				return true
			}
		}
	}
	return false
}
