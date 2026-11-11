package daemon

import "github.com/hustlex/hustlex/proto"

// phoneHandlers run the phone web app (phone.go).
var phoneHandlers = map[string]handler{
	proto.MethodPhoneStatus: bare(func(c *conn) (any, error) { return c.srv.phoneStatus(), nil }),
	proto.MethodPhoneSet:    with(func(c *conn, p proto.PhoneSetParams) (any, error) { return c.srv.setPhone(p) }),
	proto.MethodPhonePair:   bare(func(c *conn) (any, error) { return c.srv.pairPhone() }),
	proto.MethodPhoneForget: with(func(c *conn, p proto.PhoneDeviceRef) (any, error) { return ok, c.srv.phone.store.Forget(p.ID) }),
}
