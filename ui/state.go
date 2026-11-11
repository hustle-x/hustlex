package ui

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/ui/theme"
)

// appState is what the app remembers between launches, in StateFile: the
// host it showed, and per host the project and session that were open.
type appState struct {
	Host            string            `json:"host,omitempty"`
	Projects        map[string]string `json:"projects,omitempty"` // host ID → project path
	Sessions        map[string]string `json:"sessions,omitempty"` // host ID → session ID
	SkipPermissions *bool             `json:"skipPermissions,omitempty"`
	Chats           map[string]bool   `json:"chats,omitempty"`           // session IDs shown as a chat
	NotesAgents     map[string]string `json:"notesAgents,omitempty"`     // note ID → the agent its Notes page talks to
	CanvasChatWidth float32           `json:"canvasChatWidth,omitempty"` // the Canvas page's agent panel, as the user left it
	NotesView       map[string]string `json:"notesView,omitempty"`       // "<note ID> <what>" → how its page was left: tabs, toggles, outline
	CloneDirs       map[string]string `json:"cloneDirs,omitempty"`       // host ID → the folder cloned into last
	Theme           string            `json:"theme,omitempty"`           // palette ID
	Accent          string            `json:"accent,omitempty"`          // #RRGGBB replacing the palette's accent
	RailWide        bool              `json:"railWide,omitempty"`        // the rail shows labels
	Layouts         map[string]layout `json:"layouts,omitempty"`         // "<host ID> <project path>" → layout
	Panel           sidePanel         `json:"panel,omitempty"`
	Updates         updatePrefs       `json:"updates,omitempty"`
	// ProjectNames are the names the user gave projects in the list, in
	// place of their folders': "<host ID> <project path>" → name.
	ProjectNames map[string]string `json:"projectNames,omitempty"`
	// AccountNames are the names the user gave agent accounts' chips:
	// "<host ID> <agent> <ref>" (creditview's) → name.
	AccountNames map[string]string `json:"accountNames,omitempty"`
	// ActiveAccounts is the account new sessions start as: "<host ID>
	// <agent>" → a CreditAccount.Ref.
	ActiveAccounts map[string]string `json:"activeAccounts,omitempty"`
	// AutoSwitchAccounts makes new sessions move to another account of the
	// same agent once the active one nears its limit.
	// Templates are New session forms kept to use again.
	Templates          []sessionTemplate  `json:"templates,omitempty"`
	AutoSwitchAccounts bool               `json:"autoSwitchAccounts,omitempty"`
	Editor             string             `json:"editor,omitempty"`       // the editor last opened (editor.id)
	Previews           map[string]preview `json:"previews,omitempty"`     // session ID → where its browser was
	BrowserViews       map[string]bool    `json:"browserViews,omitempty"` // session IDs showing the Browser as their view
	TrafficViews       map[string]bool    `json:"trafficViews,omitempty"` // session IDs showing Traffic as their view
	AutoPreview        map[string][]int   `json:"autoPreview,omitempty"`  // "<host ID> <project>" → ports previewed when opened
	QuietPorts         bool               `json:"quietPorts,omitempty"`   // no card when a session opens a port
	CloseQuits         bool               `json:"closeQuits,omitempty"`   // closing the window quits, rather than keeping the app in the tray
	// What New session starts with (Settings): an agent, the project
	// folder rather than a new branch, a time budget in minutes.
	DefaultAgent string `json:"defaultAgent,omitempty"`
	// Layout is the window's layout: layoutCompact or layoutExpert; ""
	// before the user picked one (Expert for those from before layouts).
	Layout string `json:"layout,omitempty"`
	// TerminalViews are the sessions whose terminal the user picked over
	// the chat Compact shows first.
	TerminalViews map[string]bool `json:"terminalViews,omitempty"`
	// WelcomeDone hides the first-run steps for good.
	WelcomeDone   bool `json:"welcomeDone,omitempty"`
	ProjectFolder bool `json:"projectFolder,omitempty"`
	DefaultBudget int  `json:"defaultBudget,omitempty"`
	NotifyOff     bool `json:"notifyOff,omitempty"` // no desktop notifications
	// VaultRemember keeps the vault's key in the system's keychain.
	VaultRemember bool `json:"vaultRemember,omitempty"`
	// OAuthClients are the user's OAuth apps, by platform, for signing
	// in in the browser (not secret).
	OAuthClients map[string]string `json:"oauthClients,omitempty"`
	// NotionSyncs are imports from Notion made again on their own.
	NotionSyncs []notionSync `json:"notionSyncs,omitempty"`
	// StopAgentsOnQuit stops this device's agents when the app quits; they
	// keep working without it otherwise.
	StopAgentsOnQuit bool `json:"stopAgentsOnQuit,omitempty"`
	// FloatingControl keeps the radial panel's global shortcut bound, to
	// summon it over any app.
	FloatingControl bool `json:"floatingControl,omitempty"`
	// FloatingFoldOnBlur folds the panel away when the user goes
	// elsewhere; FloatingAtPointer shows it where the cursor is. Pointers
	// so that an app which never saved them keeps both on.
	FloatingFoldOnBlur *bool `json:"floatingFoldOnBlur,omitempty"`
	FloatingAtPointer  *bool `json:"floatingAtPointer,omitempty"`
	// Floating appearance stays independent of the app and terminal theme.
	FloatingAccent       string `json:"floatingAccent,omitempty"`
	FloatingTint         bool   `json:"floatingTint,omitempty"`
	FloatingCompact      bool   `json:"floatingCompact,omitempty"`
	FloatingReduceMotion bool   `json:"floatingReduceMotion,omitempty"`
	// HideSessions folds the Sessions panel away, giving the session the
	// room.
	HideSessions bool `json:"hideSessions,omitempty"`
	// SessionsRight puts the Sessions panel on the right, past the tools;
	// otherwise it is on the left. SessionsWidth is as wide as the user
	// dragged it, 0 for the default.
	// GlassOff paints what floats over the content (the title bar's chips,
	// menus, pop-overs) as plain surfaces, not Liquid Glass.
	GlassOff      bool `json:"glassOff,omitempty"`
	SessionsRight bool `json:"sessionsRight,omitempty"`
	// LastDark is the palette to come back to from the light mode.
	LastDark      string  `json:"lastDark,omitempty"`
	SessionsWidth float32 `json:"sessionsWidth,omitempty"`
}

// sessionsLeft reports whether the Sessions panel is on the left.
func (st appState) sessionsLeft() bool { return !st.SessionsRight }

// palette is the palette the user picked, with their accent.
func (st appState) palette() theme.Palette {
	var accent ui.Color
	if st.Accent != "" {
		accent = ui.Hex(st.Accent)
	}
	return theme.Lookup(st.Theme, accent)
}

func loadState(path string) appState {
	var st appState
	if path == "" {
		return st
	}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &st)
	}
	return st
}

// saveState writes the state when it changed. It runs on the main thread
// at the end of each frame, so every way of switching host, project or
// session is remembered without each one saying so.
func (a *App) saveState() {
	if a.opts.StateFile == "" || a.host.ID == "" {
		return
	}
	st := a.state
	st.Host = a.host.ID
	if st.Projects == nil {
		st.Projects, st.Sessions = map[string]string{}, map[string]string{}
	}
	if a.client != nil && a.restoredView {
		st.Projects[a.host.ID] = a.project
		st.Sessions[a.host.ID] = a.selected
	}
	skip := a.skipPermissions
	st.SkipPermissions = &skip
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil || string(data) == a.savedState {
		return
	}
	if err := os.MkdirAll(filepath.Dir(a.opts.StateFile), 0o700); err != nil {
		return
	}
	tmp := a.opts.StateFile + ".tmp"
	if os.WriteFile(tmp, data, 0o600) == nil && os.Rename(tmp, a.opts.StateFile) == nil {
		a.state, a.savedState = st, string(data)
	}
}

// restoreView reopens the project and session that were open on this host,
// once its state has loaded.
func (a *App) restoreView() {
	if a.restoredView {
		return
	}
	a.restoredView = true
	if p := a.state.Projects[a.host.ID]; p != "" {
		for _, proj := range a.projects {
			if proj.Path == p {
				a.project = p
			}
		}
	}
	if id := a.state.Sessions[a.host.ID]; id != "" {
		for _, s := range a.sessions {
			if s.ID == id {
				a.selectSession(s)
			}
		}
	}
}
