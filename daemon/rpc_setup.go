package daemon

import (
	"github.com/hustlex/hustlex/core/setup"
	"github.com/hustlex/hustlex/proto"
)

var setupHandlers = map[string]handler{
	proto.MethodSetupScan: bare(func(c *conn) (any, error) { return setup.Scan(), nil }),
}
