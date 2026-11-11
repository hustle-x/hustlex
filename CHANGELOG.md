# Changelog

Each version's section is also what the app's update dialog shows.

## 1.1.0

- **Black and white:** the dark mode's accent is white by default; coral stays one of the accents in Appearance.
- **Claude's own logo:** Claude Code shows Claude's spark, in its color, instead of Anthropic's mark.
- **Needs you:** sessions whose agent waits for an answer rise to the top of the Sessions panel.
- **The session's account:** the title bar outlines the account the session in view runs as, and a bar under the session says which one, what is left of it, with Switch account.
- **Account chips by email:** chips read as the email an account is signed in as (or the name you gave it), not Account 2.
- **Search & commands:** a box in the nav bar opens the command palette (⌘K).

## 1.0.7

- **Welcome:** until the first session, hustleX walks a new user through installing an agent, signing in and opening a folder, ticking each step once done.
- **New session without an agent:** it offers to install Claude Code instead of starting a plain shell.
- **Quieter form:** a folder that isn't a git repository gets a one-line tip instead of a warning, and saving the form as a template shows from the second session on.

## 1.0.6

- **Rename a project:** its menu's Rename… gives it a name in hustleX; the folder on disk keeps its own, so sessions and conversations stay as they are. An empty name goes back to the folder's.
- **Updates are found sooner:** hustleX looks every hour, and when you come back to its window after a while.

## 1.0.5

- **Accounts by name:** a new session's account choices read as their chips do: the name you gave the account, else the email it is signed in as, instead of "Default" and "Account 2".

## 1.0.4

- **Sessions keep their own chat across updates:** after an update or a restart, each agent session resumes its own conversation, even when several sessions share a folder (before, they all went on with the folder's latest one).
- **macOS and Windows only:** the Linux desktop app is gone. Linux machines stay supported as servers hustleX connects to.

## 1.0.3

- **Updates itself:** hustleX looks for a new version when it starts and every few hours, shows it in the corner and in the title bar, and installs it with one click. From this version on, no need to download it again.

## 1.0.2

- **No project needed to start:** Install Claude Code and Sign in to Claude Code in the title bar work before any folder is opened; they run in the home folder and show right away.

## 1.0.1

- **Windows:** the window's buttons are −, □ and × on the right, as on Windows (macOS keeps its three dots).
- **Windows:** agents install from the app (Claude Code with its PowerShell installer, others with npm), and are found as soon as they are installed.
- **Sign in to Claude Code:** while no account shows, the title bar offers to install Claude Code or sign it in.
- **Install from PowerShell:** the install command works again, closes a running hustleX first and checks the version it installed.
- **Tabs side by side:** a small icon in a session's tab strip shows two of its tabs at once.

## 1.0.0

The first hustleX release, from here on its own version line.

- **Two modes:** Dark and Light, in plain neutral grays and white (no warm tint). The Light mode's panels sit on a darker ground, as the Dark's do.
- **New dark look:** flat near-black panels on a darker ground with a coral accent. The page, session or tool in view is a raised card lit from its left side, with a coral bar at the list's edge (or under the page in the bar); buttons are flat and glow a little in their color.
- **Color picker:** the accent is picked with hue, intensity and brightness sliders, not a hex code. Text on buttons follows the accent's lightness so it always reads.
- **Sessions in their own windows:** drag a session out of the Sessions list, or use "Open in new window" in its menu. The window shows its terminal; closing it brings the session back to the main window and ends nothing.
- **Switch accounts at the limit:** command palette "Switch accounts at the limit". When the active account of an agent is nearly full (95%), new sessions use the account with the most room left (under 80%). Running sessions are left alone.
- **Tokens:** the Resource usage page lists the tokens each Claude Code session used (input, cache, output) with the model, and how much of what the agents read came from the cache. Counted from the agent's own conversation files, read once and then only what is appended.
- **Hooks:** a project's `.hustlex/hooks.json` can run a command or call a webhook when a session starts, finishes, needs you or fails (palette: "Hooks of this project…"). Commands run in the session's folder with `HUSTLEX_EVENT`, `HUSTLEX_SESSION` and `HUSTLEX_PROJECT` set. A project's hooks run only after you allow them on that host, so opening a repository never runs its commands.
- **Motion:** pages slide in from the side of their tab, sessions from above or below as the list has them, and the old view fades under the new one. The marker of the picked tab slides to the next one in the nav bar, segmented controls and the tools panel.
- **Session templates:** save a New session form (agent, account, branching, first prompt) under a name and start from it later; the saved ones show as chips at the top of the form.
- **Command palette:** open the selected session in its own window, or bring it back.
- **Sessions list:** while an agent works, its own note (what it is at) shows in its row in place of the branch.
- **New** works before a project exists: it opens the folder picker instead of staying greyed out.

## 0.3.0

- **Canvas:** design mockups, wireframes, diagrams and boards with agents on an endless board (scroll to move, pinch to zoom), watch them built live, compare and star variations, click a component to ask about it, change or redesign it, lay designs on a grid, share a read-only link on your tailnet, and turn a design into code. Agents start diagrams on a sheet of their own and keep designs within their frame.
- **Notes:** structured pages (tabs, toggles, columns, callouts, cards, an outline), version history, SVG icons in any color and size, an agent beside each page (Claude Code, Codex…) whose changes you accept, notes attached to projects on any host, and import from Notion (also on a schedule), Obsidian, Evernote, Keep and HTML, export to Markdown, HTML, PDF or a backup.
- **Credential vault:** your logins kept encrypted on this device and lent to sessions on any host when you allow it. Fill it from what a host's tools already keep, a token copied on the platform's page, or signing in to GitHub in the browser; git, gh, docker and ssh on servers use it.
- **Your data on every host:** notes, canvases, settings and the vault sync between this device and your servers, merged when both changed, conflicts kept to pick.
- **Projects across hosts:** clone from Git (also onto another host), keep a project's copies in sync on their own with conflicts announced, and auto push to a branch of its own.
- **Sounds:** a sound when an agent finishes, needs you or fails, your own sounds too, quiet hours and a count on the app's icon.
- **Features:** turn what agents are told about on or off, everywhere or per host, project or session.
- **Traffic:** pause, stop and resume recording. **Hibernate** a session yourself from its menu.
- **Floating control:** an optional radial menu over other apps (Cmd/Ctrl+Shift+Space): this PC or a VPS, a project, its sessions.
- Menus say a few words, the rest in a tooltip; Settings fills the window, and its sections no longer flicker between tabs and a menu.
- Fixed: the app's tests no longer stop a running daemon, Claude Code keeps its transcript when hustleX was opened from an agent's terminal, a second daemon no longer takes the server's systemd service, the first auto push of a clean project works, and Windows passes agents their arguments whole in PowerShell 5 and 7.

## 0.2.2

- **Notes:** a Notes page in the rail with pages of notes on each host, nested like a notebook's. Write a block at a time: Markdown's shortcuts (`# `, `- `, `[] `, `> `) and `/` make headings, lists, to-dos, quotes, code, tables and dividers; drag blocks and pages to move them. Pages save themselves, and agents read and write them too (`hustlex-app notes`).
- **Groups and order of projects:** drag projects and groups in the session list, or put projects in a group from their menu. The host keeps them, so every device shows the same list.
- **Pick your shell:** Settings → Terminal picks the shell terminals start with (zsh, bash, fish, PowerShell, Command Prompt, Git Bash, WSL…), and a session's **+** opens a tab of any of them.
- **Running as root:** a server's form now says whom sessions run as when you sign in as root, a regular user or root, and the choice sticks: connecting again no longer goes back to the regular user.
- Session rows stay two lines high; an agent's note shows on hovering its name.
- **Windows:** sessions start again (0.2.1's background daemon missed a library).
- Android: a device on the server connects through the server's adb, and the SDK is found by its adb.

## 0.2.1

- **The tray:** closing the window keeps hustleX in the menu bar (macOS) or the notification area (Windows, Linux). Its icon counts the sessions that need you; a click lists those needing input, working or done, each opening the window on it. Settings → App can make closing quit again.
- **Android TV:** TVs are recognized; their screen fills the panel with a remote under it (D-pad, OK, Menu), Enter is OK, and agents press the remote's keys. **+ Device** connects a TV over the network, or makes a virtual one. Mi TVs' screenshots work.
- **Windows:** installing a new version while hustleX runs no longer fails with "Error opening file for writing": the background daemon runs from a copy of the app. Updating from 0.2.0 may still need hustleX ended once in Task Manager.

## 0.2.0

- **Traffic:** see the HTTP what a session runs sends and receives, HTTPS, WebSocket, SSE and gRPC included; mock, forward, change, fail or pause requests with rules; catch a terminal, a browser, an Electron app, Docker containers, the Browser tool or an Android device; export HAR. Its forms open as dialogs.
- **Android:** run the session's app on an emulator or a phone, of this machine or the server, read its log, and use its screen right in the side panel. Catching a device's traffic trusts the CA by itself on emulators and rooted phones.
- **Model APIs** (Settings): point Claude Code, Codex, OpenCode, Gemini CLI and 12 more agents at an API of yours (a gateway, OpenRouter, a local server), with several models and, where the agent takes them, vision, context window, output limit and reasoning effort. Each agent's settings are backed up first and restored in a click.
- **Every agent knows hustleX's tools:** not only Claude Code and Codex. And agents can now do everything the side panel does: look at and tap an Android device, change traffic, run the project's commands, list its ports, set model APIs, from the command line or over MCP.
- The port cards can be turned off, in the Ports tool.
- Windows: no console window flashes for the programs hustleX runs in the background.

## 0.1.1

- Windows: setting up a server no longer flashes a burst of console windows while this device's tools are scanned.
- The New session form fits small windows: its body scrolls, and Cancel and Start session stay in view.

Thanks to @Tiyouw for both fixes.

## 0.1.0

The first release.

- Run Claude Code, Codex and 43 other coding agents in parallel, each on its own branch in its own git worktree; fan out one prompt to several and keep the best; pipelines and automations.
- Watch them as terminals, or Claude Code and Codex as a chat; review each session's changes, comment lines back, commit, push, open a pull request and merge.
- Your own server over SSH or Tailscale: the daemon runs there as a service, sessions keep running with the laptop closed, one-click setup brings the server your tools, agents and skills. Desktop apps built there open on your screen.
- Copy a project and its sessions to another host, and sync them back both ways.
- Built-in browser for dev servers, ports forwarded, a files panel, memory and CPU of every session, and the `hustlex-app` command line.
