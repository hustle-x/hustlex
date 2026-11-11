# Agent accounts

An agent can have other accounts on a host besides the one it signs in with on its own: a work and a personal Claude, say. Each is signed in once, then picked per session, with no new sign-in to switch.

Claude Code and Codex can have them; they keep their sign-in in a folder a variable names (`CLAUDE_CONFIG_DIR`, `CODEX_HOME`). Claude Agent Teams uses Claude Code's.

- **Settings → Agent accounts**: Add account names one and opens a session signing it in. Remove forgets it, with its sign-in.
- **New session** picks the account when the agent has others: Default is its own.
- **Command line**: `hustlex-app spawn --account Work …`.

Each account's home is `~/.hustlex-app/accounts/<id>` on the host (`core/accounts`). It links to what is in the agent's own home (`~/.claude`, `~/.codex`): settings, skills, subagents, commands, plugins and conversations are the same whichever account a session uses, so the chat view, history and Settings see them all. Only the sign-in is the account's own (for Claude Code also `.claude.json`, which holds who is signed in, so MCP servers added with `claude mcp add --scope user` are per account). A session keeps its account when continued, after hibernation or a restart.

## At a limit

When a session's account runs out of a usage limit mid-task, the daemon moves it to another of the agent's accounts with room (each of its limits under 80%, the one with the most room): the agent starts again in the same session, on the same conversation, as that account, and is told to go on (`daemon/accountmove.go`). It watches the screens of idle agents for the agent saying its limit ran out (`agent.LimitReached`) and confirms it with a fresh reading of the accounts, so words about limits in a reply don't move anything. The session's terminal, the app and its summary on coming back say what moved where. **Settings → New sessions → Continue on another account** turns it off per host (`DaemonSettings.StayOnAccount`); the session then waits for its reset.

A new session can also wait for its account: **New session → Start → When a limit resets** queues it until the account's fullest limit resets (`SessionCreateParams.StartAt`). The daemon keeps it in the Sessions panel's Queued group, across restarts, and starts it then.
