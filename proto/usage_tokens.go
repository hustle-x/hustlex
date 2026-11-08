package proto

// MethodUsageTokens reports the tokens each session's agent has used, as
// the agent's own conversation file counts them: no params →
// TokenReport. Only agents whose conversations hustleX reads (Claude Code
// today) are in it; reading is incremental, so it may be asked often.
const MethodUsageTokens = "usage.tokens"

// TokenReport is the tokens of the host's sessions.
type TokenReport struct {
	Sessions []SessionTokens `json:"sessions"`
}

// SessionTokens is one session's tokens, with the model that used most.
type SessionTokens struct {
	ID          string `json:"id"`
	Model       string `json:"model,omitempty"`
	Input       int64  `json:"input"`
	CacheRead   int64  `json:"cacheRead"`
	CacheCreate int64  `json:"cacheCreate"`
	Output      int64  `json:"output"`
}
