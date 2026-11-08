<p align="center">
  <img src="cmd/hustlex/resources/icon.png" width="88" height="88" alt="">
</p>

<h1 align="center">hustleX</h1>

<p align="center">
  Run Claude Code, Codex and other coding agents side by side, each on its own git branch,<br>
  on your laptop or on your own server. No cloud, no account, no telemetry.
</p>

<p align="center">
  <a href="https://hustle-x.github.io/hustlex/">Website</a> ·
  <a href="https://github.com/hustle-x/hustlex/releases/latest">Download</a> ·
  <a href="docs/install.md">Install</a> ·
  <a href="docs/agents.md">Agents</a> ·
  <a href="docs/cli.md">Command line</a>
</p>

<p align="center">
  <img src="docs/assets/app.png" alt="hustleX with a Claude Code session that has just finished a change" width="860">
</p>

<p align="center"><sub>Every image on this page uses sample data.</sub></p>

## What it does

- **Parallel sessions.** Each session is an agent in a real terminal, on its own branch and its own git worktree. Agents never overwrite each other or your checkout.
- **Compare.** Send one prompt to several agents, look at what each one changed, keep the best and drop the rest.
- **Review and ship.** See a session's diff, send comments back to the agent, then commit, push, open a pull request or merge. A warning shows when two sessions touch the same lines.
- **Terminal or chat.** Claude Code and Codex can also be read as a chat, with tool calls and diffs folded in.
- **Your own servers.** Connect over SSH or Tailscale. Sessions keep running when the laptop is closed, and a blank server can be set up in one step.
- **Accounts and usage.** Use several accounts of the same agent and keep an eye on each one's limits.
- **Browser, traffic, Android.** Preview a session's dev server, inspect the HTTP it sends, run mobile apps on an emulator.
- **Notes and canvas.** Notes and design boards that you and the agents edit together.
- **Command line.** `hustlex-app` lets scripts, and agents, start and manage sessions.
- **Light on memory.** The app is written in Go with [MyGo](https://mygo.egoist.dev/), not a bundled browser, and idle agents hibernate.

<p align="center">
  <img src="docs/assets/views-1.png" alt="Compare three agents on one prompt, and review a session's diff before merging">
  <img src="docs/assets/views-2.png" alt="The chat view of a Claude Code session, and the Traffic tool">
  <img src="docs/assets/views-3.png" alt="A note with a checklist and an agent's proposed change, and moving a session between accounts">
</p>

Works with Claude Code, Codex, OpenCode, Gemini CLI, Aider, Goose, Cursor and many more, plus any program you can run in a terminal. The list and how each one is started is in [docs/agents.md](docs/agents.md).

## Install

Download the installer for [macOS (Apple silicon), Windows or Linux](https://github.com/hustle-x/hustlex/releases/latest). The builds are not signed yet, so your system will warn you the first time; [docs/install.md](docs/install.md) shows how to open them.

Or install from a terminal.

**macOS** (Apple silicon)

```sh
curl -fsSL https://hustle-x.github.io/hustlex/get-mac.sh | sh
```

**Linux** (x64, arm64)

```sh
curl -fsSL https://hustle-x.github.io/hustlex/get.sh | sh
```

**Windows** (PowerShell)

```powershell
irm https://hustle-x.github.io/hustlex/get.ps1 | iex
```

To build it yourself you need Go 1.27+ and git:

```sh
make build
./bin/hustlex                          # this machine
./bin/hustlex --remote user@host       # a server; hustleX installs its daemon there
```

State lives in `~/.hustlex-app`, and the command line is linked as `hustlex-app` (Settings → App).

## Develop

```sh
make dev       # hot reload for the app and its daemon
make test
make lint
make package   # installers for every platform, into dist/<version>
```

`make dev` runs apart from an installed hustleX, with its state in `~/.hustlex-dev`. See [AGENTS.md](AGENTS.md) for the layout of the code.

## Documentation

Everything is in [docs/](docs): [getting started](docs/install.md), [agents](docs/agents.md), [accounts](docs/accounts.md), [fan-out](docs/fan-out.md), [review](docs/review.md), [servers](docs/vps.md), [command line](docs/cli.md), [notes](docs/notes.md), [canvas](docs/canvas.md), [traffic](docs/traffic.md), [Android](docs/android.md) and [settings](docs/settings.md).

## License

[MIT](LICENSE). Security issues: [SECURITY.md](SECURITY.md).
