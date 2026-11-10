// Package creditview shows what the agent accounts signed in on the
// connected host have left of their plans: a chip for the title bar, its
// popover, and the Credits page.
package creditview

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/client"
	"github.com/hustlex/hustlex/proto"
	"github.com/hustlex/hustlex/ui/icon"
	"github.com/hustlex/hustlex/ui/textfmt"
	"github.com/hustlex/hustlex/ui/theme"
	"github.com/hustlex/hustlex/ui/widget"
)

// The poller asks the daemon often while it reads, so the reading shows
// as soon as it is done, and rarely once it has one: the daemon reads
// again by itself when its reading gets old.
const (
	whileChecking = 3 * time.Second
	settled       = time.Minute
)

// Tracker holds the latest reading. It belongs to the main thread; the
// poller started by Follow fills it.
type Tracker struct {
	report proto.CreditsReport
	err    string
	loaded bool
	// openKey is the chip whose popover shows: an account's, or addKey.
	openKey  string
	nameBuf  string        // the name being typed in an account's popover
	renaming bool          // the popover asks for the name
	removing bool          // the popover asks to confirm removing the account
	force    chan struct{} // asks the poller for a fresh reading
}

// Host is what the views need of the connected host.
type Host struct {
	Name string
	// AgentName names an agent preset ID.
	AgentName func(id string) string
	// SignIn, when set, offers to sign an account in again.
	SignIn func(agent, ref string)
	// OpenPage shows the Credits page.
	OpenPage func()
	// AccountName is the name the user gave an account's chip, "" for none.
	AccountName func(agent, ref string) string
	// Rename gives an account's chip a name; "" takes the name back.
	Rename func(agent, ref, name string)
	// Active is the account (a Ref) new sessions of agent start as.
	Active func(agent string) string
	// Switch makes ref the account new sessions of agent start as.
	Switch func(agent, ref, name string)
	// MoveName is the name of the session in view that could go on as
	// account ref of agent, "" for none; Move moves it: the agent restarts
	// on the same conversation. Nil for neither.
	MoveName func(agent, ref string) string
	Move     func(agent, ref string)
	// Remove removes one of the host's other accounts; nil hides it.
	Remove func(agent, ref string)
	// AddAgents are the agents that can have another account, and Add
	// starts adding one of agent.
	AddAgents []AddAgent
	Add       func(agent string)
}

// AddAgent is an agent that can have another account.
type AddAgent struct{ ID, Name string }

// addKey is the openKey of the add button's popover.
const addKey = "+"

// moveSuffix follows an account's key in openKey for its "continue the
// session here" popover.
const moveSuffix = " move"

// Follow reads cl's host while wanted (called on the main thread) reports
// true and cl is open. update runs a function on the main thread.
func (t *Tracker) Follow(cl *client.Client, update func(func()), wanted func() bool) {
	*t = Tracker{force: make(chan struct{}, 1)}
	force := t.force
	go func() {
		refresh := false
		for {
			r, err := cl.Credits(refresh)
			refresh = false
			stop := make(chan bool, 1)
			update(func() {
				if !wanted() {
					stop <- true
					return
				}
				stop <- false
				t.loaded = true
				t.err = ""
				if err != nil {
					t.err = describe(err)
					return
				}
				t.report = r
			})
			if <-stop {
				return
			}
			wait := settled
			if err == nil && (r.Checking || awaitingSignIn(r)) {
				wait = whileChecking
			}
			select {
			case <-cl.Done():
				return
			case <-force:
				refresh = true
			case <-time.After(wait):
			}
		}
	}()
}

// awaitingSignIn reports whether an account of the host has no login yet:
// the daemon looks again every half minute, to show it once it has one.
func awaitingSignIn(r proto.CreditsReport) bool {
	return slices.ContainsFunc(r.Accounts, func(a proto.CreditAccount) bool { return a.Ref != "" && len(a.Windows) == 0 && a.Error != "" })
}

// Refresh asks the daemon to read the accounts again.
func (t *Tracker) Refresh() {
	select {
	case t.force <- struct{}{}:
	default:
	}
}

func describe(err error) string {
	if strings.Contains(err.Error(), "unknown method") {
		return "This host runs an older hustleX. Reconnect to update it."
	}
	return err.Error()
}

// tightest is the account closest to a limit, with that limit.
func (t *Tracker) tightest() (proto.CreditAccount, proto.CreditWindow, bool) {
	var acc proto.CreditAccount
	var win proto.CreditWindow
	found := false
	for _, a := range t.report.Accounts {
		if w, ok := a.Tightest(); ok && (!found || w.Used > win.Used) {
			acc, win, found = a, w, true
		}
	}
	return acc, win, found
}

// level colors a used percentage.
func level(used float64) ui.Color { return widget.Level(used / 100) }

// levelText colors a percentage in the chip: plain while far from the
// limit, the level's color once close to it.
func levelText(used float64) ui.Color {
	if used < 70 {
		return theme.Text
	}
	return level(used)
}

// Summary is a line for the status bar: each chip's account and its
// limits, short.
func (t *Tracker) Summary(h Host) string {
	accs := t.managed()
	names := namesOf(h, accs)
	var parts []string
	for i, a := range accs {
		if len(a.Windows) == 0 {
			continue
		}
		windows := make([]string, len(a.Windows))
		for j, w := range a.Windows {
			windows[j] = fmt.Sprintf("%s %.0f%%", w.Short(), w.Used)
		}
		parts = append(parts, names[i]+" "+strings.Join(windows, " · "))
	}
	return strings.Join(parts, "  ·  ")
}

// managed are the accounts the chips show: the host's own and those
// hustleX keeps. Other logins found on the host only show in the Credits
// page.
func (t *Tracker) managed() []proto.CreditAccount {
	var out []proto.CreditAccount
	for _, a := range t.report.Accounts {
		if a.Ref != "" {
			out = append(out, a)
		}
	}
	return out
}

// Loaded reports whether the accounts have been read once.
func (t *Tracker) Loaded() bool { return t.loaded }

// Accounts are the accounts hustleX manages, as last read.
func (t *Tracker) Accounts() []proto.CreditAccount { return t.managed() }

// namesOf is what each account's chip reads: the user's name for it, else
// "Account 1", "Account 2"… when its agent has several, else the agent's.
func namesOf(h Host, accs []proto.CreditAccount) []string {
	count, seen := map[string]int{}, map[string]int{}
	for _, a := range accs {
		count[a.Agent]++
	}
	names := make([]string, len(accs))
	for i, a := range accs {
		seen[a.Agent]++
		switch {
		case h.AccountName != nil && h.AccountName(a.Agent, a.Ref) != "":
			names[i] = h.AccountName(a.Agent, a.Ref)
		case count[a.Agent] > 1:
			names[i] = fmt.Sprintf("Account %d", seen[a.Agent])
		default:
			names[i] = h.AgentName(a.Agent)
		}
	}
	return names
}

// chipWindows is how many of an account's limits its chip shows: "5h" and
// "Wk"; the popover has them all.
const chipWindows = 2

// Chips draws a chip for each account, the one new sessions start as
// marked when its agent has several, and a button to add another.
func (t *Tracker) Chips(c *ui.Context, h Host) {
	accs := t.managed()
	names := namesOf(h, accs)
	count := map[string]int{}
	for _, a := range accs {
		count[a.Agent]++
	}
	for i, a := range accs {
		t.accountChip(c.Key(a.Agent+" "+a.Ref), h, a, names[i], count[a.Agent])
	}
	t.addButton(c, h)
}

// accountChip is one account's chip. A click on one that isn't in use
// switches to it, a right-click opens its popover: limits, who it is
// signed in as, and a way to rename it.
func (t *Tracker) accountChip(c *ui.Context, h Host, a proto.CreditAccount, name string, of int) {
	key := a.Agent + " " + a.Ref
	inUse := of > 1 && h.Active != nil && h.Active(a.Agent) == a.Ref
	tip := name
	if a.Account != "" {
		tip += " · " + a.Account
	}
	if of > 1 && !inUse {
		tip += ": click to use it for new sessions"
	}
	tip += " (right-click for details)"
	b := ui.ButtonBase(c).Height(26).PaddingX(9).Gap(6).Radius(theme.RadiusControl).Shrink(0).
		Label(name).Tooltip(tip).Cursor(ui.CursorPointer).TextColor(theme.TextSoft)
	widget.Glass(b, theme.Sunken)
	border := theme.BorderStrong
	switch {
	case inUse:
		border = theme.Accent
	case b.Hovered() || t.openKey == key:
		border = theme.AccentLine
	}
	b.Border(1, widget.Tint(b, "border", border)).Children(func() {
		widget.Avatar(c, icon.Agent(a.Agent), 16, theme.Text)
		ui.Text(c, name).FontSize(11.5).TextColor(theme.Text).SingleLine()
		if len(a.Windows) == 0 {
			if a.Error != "" {
				label := "retrying"
				if strings.Contains(strings.ToLower(a.Error), "sign") {
					label = "sign in" // not signed in yet, or signed out
				}
				ui.Text(c, label).FontSize(11).TextColor(theme.Warning).SingleLine()
			}
			return
		}
		for i, w := range a.Windows[:min(len(a.Windows), chipWindows)] {
			if i > 0 {
				ui.Text(c, "·").FontSize(11).TextColor(theme.Muted)
			}
			ui.Row(c).Gap(3).Children(func() {
				ui.Text(c, w.Short()).FontSize(11).TextColor(theme.Muted).SingleLine()
				ui.Text(c, fmt.Sprintf("%.0f%%", w.Used)).FontSize(11.5).TextColor(levelText(w.Used)).SingleLine()
			})
		}
	})
	openPopover := func() { t.openKey, t.nameBuf, t.renaming, t.removing = key, name, false, false }
	if b.RightClicked() {
		openPopover()
	}
	if b.Clicked() {
		switch {
		case of > 1 && !inUse && a.Error == "" && h.Switch != nil:
			h.Switch(a.Agent, a.Ref, name)
			if h.MoveName != nil && h.MoveName(a.Agent, a.Ref) != "" {
				t.openKey = key + moveSuffix // offer to go on with the session in view
			}
		case inUse && moveName(h, a) != "":
			// Already the account in use, but the session in view isn't on it.
			t.openKey = key + moveSuffix
		case a.Error != "":
			// Signed out: the popover has the way to sign in.
			openPopover()
		}
	}
	if t.openKey == key+moveSuffix {
		t.movePopover(c, h, b, a, name)
		return
	}
	if t.openKey != key {
		return
	}
	open := true
	ui.PopoverBase(c, b, &open, func(panel ui.Element) {
		widget.Popup(panel.AttachTo(b, ui.AnchorBottomRight, ui.AnchorTopRight).Margin(6, 0, 0, 0).Width(340)).Padding(14).Gap(10)
		if t.renaming {
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				in := widget.Input(c, &t.nameBuf, name).Grow(1).AutoFocus()
				ok := strings.TrimSpace(t.nameBuf) != ""
				if widget.GhostButton(c, "Cancel").Clicked() {
					t.renaming = false
				}
				if (widget.PrimaryButton(c, nil, "Save", "").Disabled(!ok).Clicked() || in.Submitted()) && ok {
					h.Rename(a.Agent, a.Ref, strings.TrimSpace(t.nameBuf))
					open = false
				}
			})
		} else {
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				ui.Text(c, name).FontSize(12.5).Grow(1).SingleLine()
				ui.Text(c, t.checkedLabel()).FontSize(11).TextColor(theme.Muted).SingleLine()
				if h.Rename != nil && widget.GhostButton(c, "Rename").Clicked() {
					t.renaming = true
				}
				if widget.IconButton(c, icon.Sync, "Read again", false).Size(26, 26).Clicked() {
					t.Refresh()
				}
			})
		}
		accountRow(c, h, a)
		if session := moveName(h, a); session != "" && widget.GhostButton(c, "Continue “"+session+"” as "+name).FillWidth().Justify(ui.Center).Clicked() {
			h.Move(a.Agent, a.Ref)
			open = false
		}
		if h.Remove != nil && a.Ref != proto.RefDefault {
			if t.removing {
				ui.Text(c, "Remove "+name+"? Its sign-in is deleted from "+h.Name+"; the "+h.AgentName(a.Agent)+" account itself isn't touched.").
					FontSize(11.5).TextColor(theme.Muted)
				ui.Row(c).FillWidth().Gap(8).Children(func() {
					if widget.GhostButton(c, "Cancel").Grow(1).Justify(ui.Center).Clicked() {
						t.removing = false
					}
					if widget.DangerButton(c, "Remove").Grow(1).Justify(ui.Center).Clicked() {
						h.Remove(a.Agent, a.Ref)
						open = false
					}
				})
			} else if widget.GhostButton(c, "Remove account").FillWidth().Justify(ui.Center).Clicked() {
				t.removing = true
			}
		}
		if widget.GhostButton(c, "All accounts").FillWidth().Justify(ui.Center).Clicked() {
			open = false
			if h.OpenPage != nil {
				h.OpenPage()
			}
		}
	})
	if !open {
		t.openKey = ""
	}
}

// moveName is the session in view that could go on as account a, "" for
// none.
func moveName(h Host, a proto.CreditAccount) string {
	if h.MoveName == nil || h.Move == nil || a.Error != "" {
		return ""
	}
	return h.MoveName(a.Agent, a.Ref)
}

// movePopover follows a switch: it offers to go on with the session in
// view as the account just chosen, in its conversation and folder.
func (t *Tracker) movePopover(c *ui.Context, h Host, b ui.Element, a proto.CreditAccount, name string) {
	session := moveName(h, a)
	open := session != ""
	if open {
		ui.PopoverBase(c, b, &open, func(panel ui.Element) {
			widget.Popup(panel.AttachTo(b, ui.AnchorBottomRight, ui.AnchorTopRight).Margin(6, 0, 0, 0).Width(320)).Padding(14).Gap(10)
			ui.Text(c, "New sessions now use "+name+".").FontSize(12.5)
			ui.Text(c, "Go on with “"+session+"” as "+name+"? Its agent restarts on the same conversation, in the same folder.").
				FontSize(11.5).TextColor(theme.Muted)
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				if widget.GhostButton(c, "Not now").Grow(1).Justify(ui.Center).Clicked() {
					open = false
				}
				if widget.PrimaryButton(c, nil, "Continue here", "").Grow(1).Justify(ui.Center).Clicked() {
					h.Move(a.Agent, a.Ref)
					open = false
				}
			})
		})
	}
	if !open {
		t.openKey = ""
	}
}

// addButton is the "+" after the chips: it offers the agents that can
// have another account.
func (t *Tracker) addButton(c *ui.Context, h Host) {
	if h.Add == nil || len(h.AddAgents) == 0 {
		return
	}
	b := ui.ButtonBase(c.Key("add")).Width(26).Height(26).Radius(theme.RadiusControl).Shrink(0).
		Label("Add an account").Tooltip("Add an account").Cursor(ui.CursorPointer)
	widget.Glass(b, theme.Sunken)
	border := theme.BorderStrong
	if b.Hovered() || t.openKey == addKey {
		border = theme.AccentLine
	}
	b.Border(1, widget.Tint(b, "border", border)).Justify(ui.Center).Children(func() {
		ui.Icon(c, icon.Plus).FontSize(12).TextColor(theme.TextSoft)
	})
	if b.Clicked() {
		if t.openKey == addKey {
			t.openKey = ""
		} else {
			t.openKey = addKey
		}
	}
	if t.openKey != addKey {
		return
	}
	open := true
	ui.PopoverBase(c, b, &open, func(panel ui.Element) {
		widget.Popup(panel.AttachTo(b, ui.AnchorBottomRight, ui.AnchorTopRight).Margin(6, 0, 0, 0).Width(260).Padding(6).Gap(2))
		ui.Text(c, "Add an account to").FontSize(11).TextColor(theme.Muted).Padding(4, 6)
		for _, ag := range h.AddAgents {
			r := ui.ButtonBase(c.Key(ag.ID)).FillWidth().Padding(6).Gap(10).Radius(theme.RadiusControl).Cursor(ui.CursorPointer).Label(ag.Name)
			if r.Hovered() {
				r.Background(theme.Hover)
			}
			r.Children(func() {
				widget.Avatar(c, icon.Agent(ag.ID), 24, theme.Text)
				ui.Text(c, ag.Name).FontSize(12.5).SingleLine()
			})
			if r.Clicked() {
				open = false
				h.Add(ag.ID)
			}
		}
	})
	if !open {
		t.openKey = ""
	}
}

// checkedLabel says how fresh the reading is.
func (t *Tracker) checkedLabel() string {
	switch {
	case t.report.Checking && t.report.Checked.IsZero():
		return "Reading…"
	case t.report.Checking:
		return "Updating…"
	case t.report.Checked.IsZero():
		return ""
	}
	return textfmt.Age(t.report.Checked, time.Now()) + " ago"
}

// accounts lists the accounts, as rows in the popover or cards on the
// page, or says why there are none.
func (t *Tracker) accounts(c *ui.Context, h Host, cards bool) {
	switch {
	case t.err != "":
		ui.Text(c, t.err).FontSize(12).TextColor(theme.Error)
		return
	case !t.loaded || len(t.report.Accounts) == 0 && t.report.Checking:
		ui.Row(c).Gap(8).Children(func() {
			widget.Spinner(c, 14)
			ui.Text(c, "Reading the accounts on "+h.Name+"…").FontSize(12).TextColor(theme.Muted)
		})
		return
	case len(t.report.Accounts) == 0:
		ui.Text(c, "No agent is signed in on "+h.Name+". Claude Code, Codex and Kiro CLI show here once they are.").
			FontSize(12).TextColor(theme.Muted)
		return
	}
	if !cards {
		for i, a := range t.report.Accounts {
			accountRow(c.Key(i), h, a)
		}
		return
	}
	widget.FitGrid(c, 200, 2).Gap(10).Children(func() {
		for i, a := range t.report.Accounts {
			accountCard(c.Key(i), h, a)
		}
	})
}

// accountHead is an account's agent, plan and who it is signed in as.
func accountHead(c *ui.Context, h Host, a proto.CreditAccount, size float32) {
	ui.Row(c).FillWidth().Gap(9).Children(func() {
		widget.Avatar(c, icon.Agent(a.Agent), size, theme.Text)
		ui.Column(c).Gap(1).Grow(1).MinWidth(0).Children(func() {
			ui.Row(c).Gap(7).MinWidth(0).Children(func() {
				ui.Text(c, h.AgentName(a.Agent)).FontSize(12.5).SingleLine()
				if a.Plan != "" {
					widget.Tag(c, a.Plan, theme.TextSoft)
				}
			})
			if a.Account != "" {
				ui.Text(c, a.Account).FontSize(11).TextColor(theme.Muted).SingleLine().Shrink(1)
			}
		})
	})
}

// window is one limit: its name, a meter, how much is used and when it
// starts over.
func window(c *ui.Context, w proto.CreditWindow) {
	ui.Column(c).FillWidth().Gap(5).Children(func() {
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			ui.Text(c, w.Label).FontSize(11.5).TextColor(theme.TextSoft).SingleLine()
			ui.Spacer(c)
			ui.Text(c, fmt.Sprintf("%.0f%%", w.Used)).FontSize(11.5).TextColor(level(w.Used)).SingleLine()
			if !w.ResetsAt.IsZero() {
				ui.Text(c, "resets in "+textfmt.Until(w.ResetsAt, time.Now())).FontSize(11).TextColor(theme.Muted).SingleLine()
			}
		})
		widget.Meter(c, widget.Segment{Fraction: w.Used / 100, Color: level(w.Used)})
		if w.Detail != "" {
			ui.Text(c, w.Detail).FontSize(11).TextColor(theme.Muted).SingleLine()
		}
	})
}

// accountRow is an account in the popover.
func accountRow(c *ui.Context, h Host, a proto.CreditAccount) {
	ui.Column(c).FillWidth().Gap(8).Padding(9).Radius(theme.RadiusControl+2).Background(theme.Raised).
		Border(1, theme.Border).Children(func() {
		accountHead(c, h, a, 24)
		problem(c, h, a, false)
		for i, w := range a.Windows {
			window(c.Key(i), w)
		}
	})
}

// accountCard is an account on the Credits page.
func accountCard(c *ui.Context, h Host, a proto.CreditAccount) {
	widget.Card(c).Gap(12).Children(func() {
		accountHead(c, h, a, 30)
		problem(c, h, a, true)
		for i, w := range a.Windows {
			window(c.Key(i), w)
		}
		if w, ok := a.Tightest(); ok && w.Used >= 90 {
			ui.Text(c, "Almost used up. Start new sessions on this host with another agent until it resets.").
				FontSize(11.5).TextColor(theme.Error)
		}
		if a.Note != "" {
			ui.Text(c, a.Note).FontSize(11.5).TextColor(theme.Muted)
		}
	})
}

// problem says why an account has no reading, with a way to sign in again.
func problem(c *ui.Context, h Host, a proto.CreditAccount, offerSignIn bool) {
	if a.Error == "" {
		return
	}
	ui.Row(c).FillWidth().Gap(8).Children(func() {
		ui.Text(c, a.Error).FontSize(11.5).TextColor(theme.Warning).Grow(1).Shrink(1)
		if offerSignIn && h.SignIn != nil && strings.Contains(strings.ToLower(a.Error), "sign") && widget.GhostButton(c, "Sign in").Clicked() {
			h.SignIn(a.Agent, a.Ref)
		}
	})
}

// Page is the Credits page's body: a summary, then a card per account.
func (t *Tracker) Page(c *ui.Context, h Host) {
	ui.Column(c).FillWidth().Gap(14).Children(func() {
		widget.FitGrid(c, 220, 3).Gap(10).Children(func() {
			n := len(t.report.Accounts)
			summaryCard(c, "Accounts", fmt.Sprintf("%d on %s", n, h.Name), theme.Text)
			if acc, w, ok := t.tightest(); ok {
				summaryCard(c, "Closest to its limit", fmt.Sprintf("%s · %.0f%%", h.AgentName(acc.Agent), w.Used), level(w.Used))
			} else {
				summaryCard(c, "Closest to its limit", "–", theme.Muted)
			}
			summaryCard(c, "Next reset", t.nextReset(), theme.Text)
		})
		t.accounts(c, h, true)
		ui.Text(c, "hustleX reads these with each agent's own sign-in on "+h.Name+", from the agent's vendor only, every few minutes while connected.").
			FontSize(11.5).TextColor(theme.Muted)
	})
}

func summaryCard(c *ui.Context, label, value string, col ui.Color) {
	widget.Card(c).Gap(4).Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(theme.Muted).SingleLine()
		ui.Text(c, value).FontSize(15).TextColor(col).SingleLine()
	})
}

// nextReset is how soon the first limit starts over.
func (t *Tracker) nextReset() string {
	var next time.Time
	for _, a := range t.report.Accounts {
		for _, w := range a.Windows {
			if !w.ResetsAt.IsZero() && w.ResetsAt.After(time.Now()) && (next.IsZero() || w.ResetsAt.Before(next)) {
				next = w.ResetsAt
			}
		}
	}
	if next.IsZero() {
		return "–"
	}
	return "in " + textfmt.Until(next, time.Now())
}

// CheckedLabel says how fresh the reading is, for the page header.
func (t *Tracker) CheckedLabel() string { return t.checkedLabel() }
