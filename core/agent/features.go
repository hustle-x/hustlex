package agent

import (
	"slices"
	"strings"
)

// The Guide is built of features the user turns on or off (Settings →
// Features, core/features): a feature turned off is left out of what
// agents are told, and its MCP tools aren't served, so it costs no
// tokens. A lazy feature, when the user asks for it, is a line pointing
// at "hustlex-app help ID", which prints the whole of it.

// Feature is something hustleX gives agents, which the user can turn off.
type Feature struct {
	ID, Name string
	Group    string // where Settings lists it
	About    string // what it is, for the user
	Default  bool   // on unless the user turned it off
	Core     bool   // always on
	Lazy     bool   // a line (Summary) in a lazy guide, the rest on demand
	Requires []string
	Summary  string // one line for a lazy guide
	Guide    string // its part of the guide, "" for none
	// Tools are its MCP tools: names, or prefixes ending in "_".
	Tools []string
}

// Feature groups, in the order Settings lists them.
const (
	GroupCore    = "Core"
	GroupPreview = "Preview & debug"
	GroupNotes   = "Notes"
	GroupProject = "Project"
	GroupDesign  = "Design"
	GroupSecure  = "Security"
)

// FeatureGroups are the groups, in order.
var FeatureGroups = []string{GroupCore, GroupPreview, GroupNotes, GroupProject, GroupDesign, GroupSecure}

const guideIntro = `You are running inside a hustleX session, possibly on a remote Linux server. The user watches this session from the hustleX app on their own computer. hustleX's tools are commands of "$HUSTLEX_BIN" (set in every session's environment): "$HUSTLEX_BIN" help lists them all, "$HUSTLEX_BIN" guide prints this again. Use them rather than asking the user to do it by hand.`

const guideMCP = `The same tools are MCP tools too, from "$HUSTLEX_BIN" mcp (stdio), screenshots as images, and its "hustlex" tool runs any command above, for when MCP suits you better than the shell.`

const previewHead = "Previewing what you build:\n"

// Features are every feature, in the guide's order.
var Features = []Feature{
	{ID: "browser", Name: "Browser", Group: GroupPreview, Default: true,
		About:   "Dev servers the session starts show in the app's Browser, which agents can drive and screenshot.",
		Summary: "Browser: dev servers show in the user's Browser; \"$HUSTLEX_BIN\" browser open|snapshot|click|fill|screenshot drives it.",
		Guide: `- Web apps: start the dev server as usual (bind to localhost). hustleX notices ports the session listens on, forwards them to the user's computer, and shows them in its Browser tool, with hot reload. Don't ask the user to open ports or use the server's IP.
` + `- Run long-running dev servers and apps in the background or so they keep running, and say which port or window to look at.
` + `- To check a web app yourself, drive the same Browser the user sees (it needs the hustleX app open):
    "$HUSTLEX_BIN" browser open localhost:5173   # or a path of the dev server, or any URL
    "$HUSTLEX_BIN" browser snapshot              # the page as text, its controls numbered
    "$HUSTLEX_BIN" browser click 3               # a control's number, or a CSS selector
    "$HUSTLEX_BIN" browser fill 2 "ada@example.com"
    "$HUSTLEX_BIN" browser screenshot /tmp/page.png
    "$HUSTLEX_BIN" browser eval "document.title"   # any JavaScript in the page: DOM, storage, state
    "$HUSTLEX_BIN" browser back | forward | reload
`,
		Tools: []string{"browser_"}},
	{ID: "ports", Name: "Ports & Run menu", Group: GroupPreview, Default: true,
		About:   "The ports a session listens on, and its project's commands.",
		Summary: "Ports & Run: \"$HUSTLEX_BIN\" ports, run.",
		Guide: `- The session's ports, and its project's commands (the Run menu), run in a tab of the session:
    "$HUSTLEX_BIN" ports                          # what it listens on; "ports preview 5173" shows one in the Browser
    "$HUSTLEX_BIN" run                            # the commands; "run 2" or "run dev" runs one, "read TAB" shows it` + "\n",
		Tools: []string{"ports_", "run_"}},
	{ID: "display", Name: "Display (desktop apps on Linux)", Group: GroupPreview,
		About:   "A virtual display for desktop apps on a Linux server, shown on the user's screen.",
		Summary: "Desktop apps on Linux: eval \"$(\"$HUSTLEX_BIN\" display)\" first, then run the app; it opens on the user's screen.",
		Guide: `- Desktop apps (MyGo, Tauri, Electron, Wails, GTK, Qt, Flutter desktop, ...) on Linux: the session may have no screen. Before running one, get a virtual display with
    eval "$("$HUSTLEX_BIN" display)"
  then run the app (for example the framework's dev command) in that same shell. hustleX opens the app on the user's own screen, in a window of its own, as if it ran there: they see and use it live. Prefer this over Xvfb and screenshots, and tell the user the app's window is open. When the user closes that window, the app and everything started with that DISPLAY stop, dev server included: run the eval line and the app again if they ask for it.
- If the command fails because Xpra is missing, install it with the system package manager (xpra), then run it again; install the libraries the app needs (GTK, WebKitGTK, ...) the same way. When sudo asks for a password you don't have, give the user the exact install command to run in a terminal tab of this session (the + next to the session's tabs), and go on once they say it is done.
`},
	{ID: "traffic", Name: "Traffic (HTTP inspector)", Group: GroupPreview, Lazy: true,
		About:   "Catches the HTTP of what sessions run, to read, mock and change it.",
		Summary: "HTTP traffic: hustleX can catch, mock and change what this session's programs send (\"$HUSTLEX_BIN\" help traffic).",
		Guide: `HTTP traffic: hustleX catches the HTTP(S) what this session runs sends, once attached, to read, mock and change it, rather than adding logging to the code:
    eval "$("$HUSTLEX_BIN" traffic env)"         # this shell's programs go through it (curl, Node, Python, Go, Java...)
    "$HUSTLEX_BIN" traffic list --filter "domain:api.example.com status:>=400"
    "$HUSTLEX_BIN" traffic show 12               # headers and bodies, decoded, secrets masked
    "$HUSTLEX_BIN" traffic watch --until-match --filter "path:/api/login"
    "$HUSTLEX_BIN" traffic mock --method POST --path /api/pay --status 500   # saved in .hustlex/traffic.json
    "$HUSTLEX_BIN" traffic fail --path /api/pay --how timeout                 # or forward --to, transform --res-set…
    "$HUSTLEX_BIN" traffic show 12 --as curl     # the request as code that sends it again
    "$HUSTLEX_BIN" traffic status                # recording, paused or stopped; "traffic resume" or "start" when nothing new shows
    "$HUSTLEX_BIN" traffic help                  # breakpoints, rules, send, HAR, settings, the CA, attaching browsers, Electron, docker, Android`,
		Tools: []string{"traffic_"}},
	{ID: "android", Name: "Android", Group: GroupPreview, Lazy: true,
		About:   "Emulators and phones: run the project's app, look at it, tap and type.",
		Summary: "Android: run and drive the app on an emulator or a phone (\"$HUSTLEX_BIN\" help android).",
		Guide: `Android: run and debug the app on an emulator or a phone, of this host or another the user connected:
    "$HUSTLEX_BIN" android status                # devices and virtual phones; "android setup" brings the SDK when missing
    "$HUSTLEX_BIN" android start Pixel_8         # a virtual phone, by its name in status
    "$HUSTLEX_BIN" android run emulator-5554     # the project's app (Flutter, Expo, React Native, Capacitor, NativePHP, Gradle) in a new tab
    "$HUSTLEX_BIN" android logcat emulator-5554 --package com.example.app --level W
  and use it, to check what you built yourself (-s SERIAL when more than one is online):
    "$HUSTLEX_BIN" android screenshot /tmp/screen.png   # look at it (read the PNG)
    "$HUSTLEX_BIN" android controls              # what is on screen, controls numbered
    "$HUSTLEX_BIN" android tap 3                 # a control by its number, or "tap X Y"; also longpress, swipe X1 Y1 X2 Y2
    "$HUSTLEX_BIN" android type --into 2 "ada@example.com"
    "$HUSTLEX_BIN" android key back              # home, recents, enter, delete…; on an Android TV, up, down, left, right, ok
    "$HUSTLEX_BIN" android catch                 # its HTTP shows in traffic list, its CA trusted`,
		Tools: []string{"android_"}},
	{ID: "agents", Name: "Multi-agent", Group: GroupCore, Default: true,
		About:   "Agents start, follow and talk to other agents in sessions of their own.",
		Summary: "Other agents: \"$HUSTLEX_BIN\" spawn, wait, read, send.",
		Guide: `Other agents: when work splits into parts that can go on at once, or the user asks for a second opinion, you can start other coding agents in sessions of their own, which the user sees in hustleX too, and follow them:
    "$HUSTLEX_BIN" spawn --agent codex --branch fix/parser "Fix the parser's handling of ..."   # prints the session's ID
    "$HUSTLEX_BIN" wait ID          # until it is done or asks something
    "$HUSTLEX_BIN" read ID          # what its screen shows (also a tab's, as android run's)
    "$HUSTLEX_BIN" send ID "..."    # a message to it
    "$HUSTLEX_BIN" ls               # every session; "$HUSTLEX_BIN" agents lists the agents installed
--branch gives it a branch in a worktree of its own, so it doesn't touch your files; without it, it works in this project's folder. Tell the user what you started.
Done with sessions you spawned? "$HUSTLEX_BIN" hibernate ID... frees the host's memory and CPU (the user confirms in the app; "wake ID" brings one back on its conversation; --dry-run says what it would stop).`,
		Tools: []string{"session_spawn", "session_read", "session_send", "session_wait", "sessions_list"}},
	{ID: "modelapi", Name: "Model APIs", Group: GroupCore,
		About:   "Point an agent at another model API (a gateway, OpenRouter, a local server).",
		Summary: "Model APIs: \"$HUSTLEX_BIN\" api set AGENT --url URL --key-env VAR --model M, api restore AGENT.",
		Guide:   `Model APIs: when the user asks to point an agent at another model API (a gateway, OpenRouter, a local server), "$HUSTLEX_BIN" api set AGENT --url URL --key-env VAR --model M writes it into that agent's settings, backed up first; "$HUSTLEX_BIN" api restore AGENT puts them back.`,
		Tools:   []string{"agent_api_"}},
	{ID: "clone", Name: "Clone from Git", Group: GroupProject, Default: true,
		About:   "Clone a repository as a new project, on any host.",
		Summary: "Projects: \"$HUSTLEX_BIN\" project clone URL --dir D, project add PATH, project list.",
		Guide:   `Projects: "$HUSTLEX_BIN" project clone URL --dir ~/code clones a repository on this host (with its git credentials) as a new hustleX project, showing git's progress; "project add PATH" makes a folder one, "project list" lists them.`,
		Tools:   []string{"project_add", "project_clone", "project_clone_status", "projects_list"}},
	{ID: "autosync", Name: "Project copies & auto push", Group: GroupProject, Default: true, Lazy: true,
		About:   "Copies of a project on other hosts kept in sync, and work pushed to a branch of its own.",
		Summary: "Project copies on other hosts and auto push to its own branch (\"$HUSTLEX_BIN\" help autosync).",
		Guide:   ProjectLinksGuide,
		Tools:   []string{"project_autopush", "project_conflicts", "project_copy", "project_history", "project_links", "project_origin", "project_sync"}},
	{ID: "attach", Name: "Notes attached to projects", Group: GroupProject, Default: true, Requires: []string{"notes"},
		About:   "Plans and TODOs attached to projects on any host, and their context read-only.",
		Summary: "A project's plans are notes attached to it: \"$HUSTLEX_BIN\" notes --project . (\"$HUSTLEX_BIN\" help attach).",
		Guide:   AttachGuide,
		Tools:   []string{"notes_attach", "notes_detach", "notes_projects", "notes_for_project", "project_context", "project_read"}},
	{ID: "notes", Name: "Notes", Group: GroupNotes, Default: true, Core: true,
		About: "The user's notes, which agents read and write.",
		Guide: `The user's notes: their Notes page in hustleX, pages of Markdown on this host. Read or write them when the user asks (a plan to keep, a checklist, what you found):
    "$HUSTLEX_BIN" notes                         # the notes, as a tree
    "$HUSTLEX_BIN" notes show "Release plan"     # one's Markdown, by title or ID
    "$HUSTLEX_BIN" notes new "Release plan" --body "- [ ] tag"   # also --parent NOTE, --group G, --body - for stdin
    "$HUSTLEX_BIN" notes write "Release plan" --append "- [ ] publish"   # or --body to replace it`,
		Tools: []string{"notes_list", "notes_read", "notes_create", "notes_write", "notes_delete", "notes_history", "notes_restore"}},
	{ID: "sync", Name: "Sync between hosts", Group: GroupCore, Default: true, Core: true,
		About: "Notes, canvases, settings and the vault the same on every host.",
		Guide: `Notes and the user's other hustleX data are the same on every host (the app syncs them): "$HUSTLEX_BIN" sync status, sync now, sync conflicts, sync resolve ID --keep HOST|both, sync trash, sync restore ID.`,
		Tools: []string{"sync_"}},
	{ID: "notes-agent", Name: "Agent chat in Notes", Group: GroupNotes, Default: true,
		About: "Ask an agent about a note, beside it in the Notes page.",
		Guide: "A note's own agent in the Notes page (notes only, its changes proposed to the user): \"$HUSTLEX_BIN\" notes ask NOTE \"…\" [--wait] asks it, notes chat NOTE shows what was said, notes agent NOTE --use codex picks it; notes proposals lists what agents proposed, notes accept|reject ID settles one when the user asks.",
		Tools: []string{"notes_ask", "notes_chat", "notes_agent", "notes_proposals", "notes_accept", "notes_reject"}},
	{ID: "groups", Name: "Note groups", Group: GroupNotes, Default: true,
		About: "Notes in groups, like folders.",
		Guide: "Notes can be in groups, like folders: \"$HUSTLEX_BIN\" notes groups, notes new T --group G, notes move NOTE --group G."},
	{ID: "blocks", Name: "Structured notes", Group: GroupNotes, Default: true, Requires: []string{"notes"},
		About:   "Tabs, toggles, columns, cards in notes, and reading or writing a section at a time.",
		Summary: "Notes have layout blocks and sections (\"$HUSTLEX_BIN\" help blocks).",
		Guide:   BlocksGuide,
		Tools:   []string{"notes_outline", "notes_check"}},
	{ID: "icons", Name: "SVG icons", Group: GroupNotes, Default: true, Requires: []string{"notes"},
		About: "Notes' and groups' icons are SVG from icon libraries.",
		Guide: IconsGuide,
		Tools: []string{"icons_search"}},
	{ID: "notesio", Name: "Import & export notes", Group: GroupNotes, Requires: []string{"notes"},
		About:   "Notes from Notion, Obsidian, Evernote, Keep, HTML, and out as Markdown, HTML, PDF, a backup.",
		Summary: "Notes import/export: \"$HUSTLEX_BIN\" notes import FILE --dry-run, notes export --all --format zip.",
		Guide:   "Notes from and to other apps:\n" + NotesIOGuide,
		Tools:   []string{"notes_import", "notes_import_status", "notes_export"}},
	{ID: "canvas", Name: "Canvas", Group: GroupDesign, Requires: []string{"browser"},
		About: "Mockups, wireframes and diagrams agents design live.",
		Guide: `Canvas: when the user asks for a mockup, a wireframe or a diagram, design it on a canvas they watch build up live in hustleX: "$HUSTLEX_BIN" canvas guide tells how (read it first); "$HUSTLEX_BIN" canvas list lists them.`,
		Tools: []string{"canvas_"}},
	{ID: "creds", Name: "Credential vault", Group: GroupSecure,
		About:   "Logins kept encrypted on the user's device, used on any host by name.",
		Summary: "Credentials: use the vault's logins by name, never their values (\"$HUSTLEX_BIN\" help creds).",
		Guide:   CredsGuide,
		Tools:   []string{"creds_"}},
	{ID: "notify", Name: "Notification sounds", Group: GroupCore, Default: true,
		About:   "Sounds in the app when agents finish, ask or fail; agents can play one.",
		Summary: "Sounds: \"$HUSTLEX_BIN\" notify sound done|input|error plays one in the user's app.",
		Guide:   NotifyGuide,
		Tools:   []string{"notify_"}},
	{ID: "status", Name: "Session status line", Group: GroupCore, Default: true,
		About:   "Agents say where they are in a line under their session.",
		Summary: "Say where you are at milestones: \"$HUSTLEX_BIN\" note self \"…\".",
		Guide: `Keeping the user in the loop: at milestones (a hypothesis confirmed, a fix in, tests running, blocked on something), say where you are in a line, which hustleX shows under this session:
    "$HUSTLEX_BIN" note self "fix in for the login redirect; running the integration tests"`,
		Tools: []string{"session_note"}},
	{ID: "titles", Name: "Session titles", Group: GroupCore, Default: true,
		About:   "Agents name their session for what it does.",
		Summary: "Name your session for what it does: \"$HUSTLEX_BIN\" title self \"…\".",
		Guide: `Session titles: once you know what this session is for, name it in a few words, and again when its work changes: the gist of the work ("Layout cards in Appearance"), never the user's message copied; a name the user gave it stays theirs:
    "$HUSTLEX_BIN" title self "Fix the login redirect"`,
		Tools: []string{"session_title"}},
}

// FindFeature is feature id.
func FindFeature(id string) (Feature, bool) {
	i := slices.IndexFunc(Features, func(f Feature) bool { return f.ID == id })
	if i < 0 {
		return Feature{}, false
	}
	return Features[i], true
}

// DefaultFeatures are the features on for a user who changed none.
func DefaultFeatures() map[string]bool {
	out := map[string]bool{}
	for _, f := range Features {
		out[f.ID] = f.Default || f.Core
	}
	return out
}

// AllFeatures are every feature on.
func AllFeatures() map[string]bool {
	out := map[string]bool{}
	for _, f := range Features {
		out[f.ID] = true
	}
	return out
}

// GuideFor is the guide of the features on: each in full, or with lazy,
// the long ones as a line pointing at their help.
func GuideFor(on map[string]bool, lazy bool) string {
	var b strings.Builder
	b.WriteString(guideIntro)
	part := func(f Feature) string {
		if lazy && f.Lazy && f.Summary != "" {
			return f.Summary
		}
		return strings.TrimRight(f.Guide, "\n")
	}
	preview := []string{"browser", "ports", "display"}
	var bullets []string
	for _, id := range preview {
		if f, _ := FindFeature(id); on[id] && f.Guide != "" {
			bullets = append(bullets, part(f))
		}
	}
	if len(bullets) > 0 {
		b.WriteString("\n\n" + previewHead + strings.Join(bullets, "\n"))
	}
	for _, f := range Features {
		if slices.Contains(preview, f.ID) || !on[f.ID] && !f.Core || f.Guide == "" {
			continue
		}
		b.WriteString("\n\n" + part(f))
	}
	b.WriteString("\n\n" + guideMCP)
	return b.String()
}

// FeatureOfTool is the feature MCP tool name belongs to, "" for one of
// none (always served).
func FeatureOfTool(name string) string {
	for _, f := range Features {
		for _, t := range f.Tools {
			if t == name || strings.HasSuffix(t, "_") && strings.HasPrefix(name, t) {
				return f.ID
			}
		}
	}
	return ""
}

// GuideTokens is about how many tokens text costs an agent.
func GuideTokens(text string) int { return (len(text) + 3) / 4 }
