package daemon

import "github.com/hustlex/hustlex/proto"

var accountHandlers = map[string]handler{
	proto.MethodAccountList: bare(func(c *conn) (any, error) { return c.srv.accounts.List(), nil }),
	proto.MethodAccountAdd: with(func(c *conn, p proto.AccountAddParams) (any, error) {
		acc, err := c.srv.accounts.Add(p.Agent, p.Name)
		c.srv.credits.Invalidate() // the credits list the new account
		return acc, err
	}),
	proto.MethodAccountRemove: with(func(c *conn, p proto.AccountRef) (any, error) {
		err := c.srv.accounts.Remove(p.ID)
		c.srv.credits.Invalidate()
		return ok, err
	}),
}
