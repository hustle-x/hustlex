<p align="center">
  <img src="cmd/hustlex/resources/icon.png" width="88" height="88" alt="">
</p>

<h1 align="center">hustleX</h1>

<p align="center">
  Run Claude Code, Codex and other coding agents side by side, each on its own git branch,<br>
  on your laptop or on your own server. No cloud, no account, no telemetry.
</p>

<p align="center">
  <a href="https://github.com/hustle-x/hustlex/releases/latest"><img src="https://img.shields.io/github/v/release/hustle-x/hustlex?style=for-the-badge&color=555&label=release" alt="Latest release"></a>
  <img src="https://img.shields.io/badge/macOS_·_Windows-555?style=for-the-badge" alt="macOS, Windows">
  <img src="https://img.shields.io/badge/written_in-Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Written in Go">
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-555?style=for-the-badge" alt="License: MIT"></a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/no_cloud-111?style=flat-square" alt="No cloud">
  <img src="https://img.shields.io/badge/no_account-111?style=flat-square" alt="No account">
  <img src="https://img.shields.io/badge/no_telemetry-111?style=flat-square" alt="No telemetry">
  <a href="https://github.com/hustle-x/hustlex/stargazers"><img src="https://img.shields.io/github/stars/hustle-x/hustlex?style=flat-square&logo=github" alt="GitHub stars"></a>
</p>

<p align="center">
  <a href="https://hustle-x.github.io/hustlex/">Website</a> ·
  <a href="https://github.com/hustle-x/hustlex/releases/latest">Download</a> ·
  <a href="docs/install.md">Install</a> ·
  <a href="docs/agents.md">Agents</a> ·
  <a href="docs/cli.md">Command line</a> ·
  <a href="#license">License</a>
</p>

<p align="center">
  <img src="docs/assets/hero.gif" alt="hustleX with five sessions: a prompt typed into Claude Code, which reads, edits, writes a test and runs it" width="880">
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

Works with Claude Code, Codex, OpenCode, Gemini CLI, Aider, Goose, Cursor and many more, plus any program you can run in a terminal. The list and how each one is started is in [docs/agents.md](docs/agents.md).

## See it work

### Compare

One prompt, three agents. Each works on a branch of its own, and Compare shows what every one of them changed, side by side, so you keep the best.

<p align="center">
  <img src="docs/assets/compare.gif" alt="A new session sent to Claude Code, Codex and Gemini CLI at once, then Compare showing the files each one changed" width="880">
</p>

### Review and ship

When an agent is done, the Git panel lists what it changed. Read the diff file by file, then commit from the same place.

<p align="center">
  <img src="docs/assets/review.gif" alt="Claude Code finishing a change in the terminal, the Git panel listing four changed files, the diff of notes.js, and the Commit dialog" width="880">
</p>

### Terminal or chat

Read a Claude Code or Codex session as a chat: its tool calls fold into one line, each edit opens to its diff, and the message box types into the agent.

<p align="center">
  <img src="docs/assets/chat.gif" alt="A Claude Code session shown as a chat: the work folded, an edit opened to its diff, then a second message sent from the message box" width="880">
</p>

### Traffic

See the HTTP that what a session runs sends and gets back, live. Mock a request from what it answered, and run it again.

<p align="center">
  <img src="docs/assets/traffic.gif" alt="A caught terminal running a smoke test against a local API, the requests listed with their status, a 500 turned into a mock, and the same call answered 200 by it" width="880">
</p>

### Notes with an agent

Ask an agent about a page. It proposes its changes above the page, as a diff, and nothing changes until you accept.

<p align="center">
  <img src="docs/assets/notes.gif" alt="A rough note tidied by Claude Code: its proposal shown as a diff, accepted, and a to-do ticked" width="880">
</p>

### Accounts

Each account's limits sit in the title bar. When one runs low, pick another, and the session goes on there with the same conversation.

<p align="center">
  <img src="docs/assets/accounts.gif" alt="An account at 96% of its five-hour limit, a switch to a second account, and the session continuing there on the same conversation" width="880">
</p>


## Install

Download the installer for [macOS (Apple silicon) or Windows](https://github.com/hustle-x/hustlex/releases/latest). The builds are not signed yet, so your system will warn you the first time; [docs/install.md](docs/install.md) shows how to open them.

Or install from a terminal.

**macOS** (Apple silicon)

```sh
curl -fsSL https://hustle-x.github.io/hustlex/get-mac.sh | sh
```

**Windows** (PowerShell)

```powershell
irm https://hustle-x.github.io/hustlex/get.ps1 | iex
```

<details>
<summary><b>Build from source</b></summary>

You need Go 1.27+ and git:

```sh
make build
./bin/hustlex                          # this machine
./bin/hustlex --remote user@host       # a server; hustleX installs its daemon there
```
</details>

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

[MIT](LICENSE). Third-party parts keep their own licenses, next to them in the source. Security issues: [SECURITY.md](SECURITY.md).
