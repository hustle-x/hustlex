package proto

// MethodHooksGet reads a project's hooks and whether this host lets them
// run: HooksParams → HooksState.
const MethodHooksGet = "hooks.get"

// MethodHooksSet saves a project's hooks and whether they may run:
// HooksState → HooksState. Hooks are in the project's .hustlex/hooks.json.
const MethodHooksSet = "hooks.set"

// HooksParams names the project.
type HooksParams struct {
	Project string `json:"project"`
}

// HooksState is a project's hooks and whether this host runs them.
type HooksState struct {
	Project string        `json:"project"`
	Hooks   []ProjectHook `json:"hooks"`
	Allowed bool          `json:"allowed"`
}

// ProjectHook does something when an agent's session has an event: Event
// is "start", "done", "needs-input" or "error"; Run is a command run in the
// session's folder (with HUSTLEX_EVENT, HUSTLEX_SESSION and more in its
// environment), Webhook an address that gets the event as JSON.
type ProjectHook struct {
	Event   string `json:"event"`
	Run     string `json:"run,omitempty"`
	Webhook string `json:"webhook,omitempty"`
}
