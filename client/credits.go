package client

import "github.com/hustlex/hustlex/proto"

// Credits returns the daemon's latest reading of the agent accounts on its
// host; refresh asks it to read them again.
func (c *Client) Credits(refresh bool) (proto.CreditsReport, error) {
	var out proto.CreditsReport
	err := c.Call(proto.MethodCreditsGet, proto.CreditsParams{Refresh: refresh}, &out)
	return out, err
}

// Tokens returns the tokens each readable session's agent has used.
func (c *Client) Tokens() (proto.TokenReport, error) {
	var out proto.TokenReport
	err := c.Call(proto.MethodUsageTokens, struct{}{}, &out)
	return out, err
}

// Hooks reads a project's hooks and whether the host runs them.
func (c *Client) Hooks(project string) (proto.HooksState, error) {
	var out proto.HooksState
	err := c.Call(proto.MethodHooksGet, proto.HooksParams{Project: project}, &out)
	return out, err
}

// SetHooks saves a project's hooks and whether the host runs them.
func (c *Client) SetHooks(st proto.HooksState) (proto.HooksState, error) {
	var out proto.HooksState
	err := c.Call(proto.MethodHooksSet, st, &out)
	return out, err
}

// UsageHistory returns the tokens the host's agents used in the last days,
// by day, conversation and model.
func (c *Client) UsageHistory(days int) (proto.UsageHistory, error) {
	var out proto.UsageHistory
	err := c.Call(proto.MethodUsageHistory, proto.UsageHistoryParams{Days: days}, &out)
	return out, err
}
