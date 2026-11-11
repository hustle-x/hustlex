package proto

// MethodUsageHistory reports the tokens the host's agents used by day, as
// their conversation files count them: UsageHistoryParams →
// UsageHistory. The first reading of a host takes a few seconds; until it
// is done the reply says Reading.
const MethodUsageHistory = "usage.history"

// UsageHistoryParams asks for the days since Days ago, today included.
type UsageHistoryParams struct {
	Days int `json:"days"`
}

// UsageHistory is the tokens of the host's agents' conversations.
type UsageHistory struct {
	Conversations []UsageConversation `json:"conversations"`
	Rows          []UsageRow          `json:"rows"`
	Reading       bool                `json:"reading,omitempty"` // still reading the first time
}

// UsageConversation is a conversation the rows count.
type UsageConversation struct {
	Agent   string `json:"agent"`             // agent preset ID: "claude", "codex"
	Account string `json:"account"`           // a CreditAccount's Ref: RefDefault for the agent's own
	Project string `json:"project,omitempty"` // the project it ran in, else its folder
	Session string `json:"session,omitempty"` // the hustleX session it was, "" for none
	Title   string `json:"title,omitempty"`   // its first prompt's first line
}

// UsageRow is a conversation's tokens of one model on one day.
type UsageRow struct {
	Day          string `json:"day"`          // the host's local date, 2006-01-02
	Conversation int    `json:"conversation"` // index in Conversations
	Model        string `json:"model"`
	Input        int64  `json:"input,omitempty"`
	CacheRead    int64  `json:"cacheRead,omitempty"`
	CacheCreate  int64  `json:"cacheCreate,omitempty"`
	Output       int64  `json:"output,omitempty"`
}
