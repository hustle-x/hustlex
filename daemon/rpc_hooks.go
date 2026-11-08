package daemon

import (
	"time"

	"github.com/hustlex/hustlex/core/hooks"
	"github.com/hustlex/hustlex/proto"
)

var hookHandlers = map[string]handler{
	proto.MethodHooksGet: with(func(c *conn, p proto.HooksParams) (any, error) {
		list, err := hooks.Load(p.Project)
		if err != nil {
			return nil, err
		}
		return proto.HooksState{Project: p.Project, Hooks: list, Allowed: c.srv.hooksAllowed.Is(p.Project)}, nil
	}),
	proto.MethodHooksSet: with(func(c *conn, p proto.HooksState) (any, error) {
		if err := hooks.Save(p.Project, p.Hooks); err != nil {
			return nil, err
		}
		if err := c.srv.hooksAllowed.Set(p.Project, p.Allowed); err != nil {
			return nil, err
		}
		return p, nil
	}),
}

// fireHooks runs the hooks of the session's project for the event its
// update means, if the project may run hooks here.
func (s *Server) fireHooks(info proto.SessionInfo) {
	if info.Project == "" {
		return
	}
	ev := s.hookEvents.Event(info, time.Now())
	if ev == "" || !s.hooksAllowed.Is(info.Project) {
		return
	}
	list, err := hooks.Load(info.Project)
	if err != nil {
		s.logger().Warn("hooks not read", "project", info.Project, "err", err)
		return
	}
	hooks.Fire(list, ev, info, func(msg string, args ...any) { s.logger().Warn(msg, args...) })
}
