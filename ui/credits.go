package ui

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/client"
	"github.com/hustlex/hustlex/core/agent"
	"github.com/hustlex/hustlex/proto"
	"github.com/hustlex/hustlex/ui/creditview"
	"github.com/hustlex/hustlex/ui/icon"
	"github.com/hustlex/hustlex/ui/theme"
	"github.com/hustlex/hustlex/ui/widget"
)

// pollCredits reads the host's agent accounts while connection gen lasts.
func (a *App) pollCredits(cl *client.Client, gen int) {
	a.credits.Follow(cl, a.onMain, func() bool { return gen == a.gen })
}

// creditsHost describes the connected host to the credit views.
func (a *App) creditsHost() creditview.Host {
	h := creditview.Host{
		Name:        a.host.Name,
		AgentName:   a.agentName,
		OpenPage:    func() { a.page = pageCredits },
		AccountName: a.accountChipName,
		Rename:      a.renameAccountChip,
		Active:      a.activeRef,
		Switch:      a.switchAccount,
		Add:         a.addNextAccount,
		Remove:      a.removeAccountChip,
		MoveName:    a.movableSessionName,
		Move:        a.moveSession,
	}
	for _, ag := range a.accountAgents() {
		h.AddAgents = append(h.AddAgents, creditview.AddAgent{ID: ag.ID, Name: ag.Name})
	}
	if p, ok := a.currentProject(); ok {
		h.SignIn = func(agentID, ref string) { a.signInFromCredits(p, agentID, ref) }
	}
	return h
}

// accountKey keys an account's chip in the app's state.
func (a *App) accountKey(agentID, ref string) string { return a.host.ID + " " + agentID + " " + ref }

// accountChipName is the name of an account's chip: the user's, else the
// one the account was added under; "" for the agent's own, unnamed.
func (a *App) accountChipName(agentID, ref string) string {
	if name := a.state.AccountNames[a.accountKey(agentID, ref)]; name != "" {
		return name
	}
	if ref != proto.RefDefault {
		for _, acc := range a.accountsOf().list {
			if acc.ID == ref {
				return acc.Name
			}
		}
	}
	return ""
}

func (a *App) renameAccountChip(agentID, ref, name string) {
	if a.state.AccountNames == nil {
		a.state.AccountNames = map[string]string{}
	}
	a.state.AccountNames[a.accountKey(agentID, ref)] = name
	a.saveState()
}

// activeRef is the account new sessions of agentID start as.
func (a *App) activeRef(agentID string) string {
	if ref := a.state.ActiveAccounts[a.host.ID+" "+agentID]; ref != "" {
		return ref
	}
	return proto.RefDefault
}

// activeAccountID is that account as a new session names it: "" for the
// agent's own, and for one that has since been removed.
func (a *App) activeAccountID(agentID string) string {
	ref := a.activeRef(agent.Base(agentID))
	if ref == proto.RefDefault || !slices.ContainsFunc(a.accountsOf().list, func(acc proto.Account) bool { return acc.ID == ref }) {
		return ""
	}
	return ref
}

func (a *App) switchAccount(agentID, ref, name string) {
	if a.state.ActiveAccounts == nil {
		a.state.ActiveAccounts = map[string]string{}
	}
	a.state.ActiveAccounts[a.host.ID+" "+agentID] = ref
	a.saveState()
	a.notify("New " + a.agentName(agentID) + " sessions use " + name)
}

// accountOfRef is a chip's Ref as a session names its account: "" for the
// agent's own.
func accountOfRef(ref string) string {
	if ref == proto.RefDefault {
		return ""
	}
	return ref
}

// movableSession is the session in view if it is of agentID, runs as
// another account than ref, and can go on as ref: a session of its own,
// not a tab or a terminal.
func (a *App) movableSession(agentID, ref string) (proto.SessionInfo, bool) {
	for _, s := range a.sessions {
		if s.ID == a.selected && s.Parent == "" && s.Project != "" && agent.Base(s.Agent) == agentID && s.Account != accountOfRef(ref) {
			return s, true
		}
	}
	return proto.SessionInfo{}, false
}

func (a *App) movableSessionName(agentID, ref string) string {
	if s, ok := a.movableSession(agentID, ref); ok {
		return a.sessionName(s)
	}
	return ""
}

// moveSession goes on with the session in view as account ref of agentID:
// a session of the same agent, in the same folder, on the same
// conversation, which then replaces it. The folders of an agent's accounts
// share its conversations, so the new account finds it.
func (a *App) moveSession(agentID, ref string) {
	s, ok := a.movableSession(agentID, ref)
	cl := a.client
	if !ok || cl == nil {
		return
	}
	p := proto.SessionCreateParams{
		Project: s.Project, Name: s.Name, Agent: s.Agent, Account: accountOfRef(ref),
		SkipPermissions: s.SkipPermissions, BranchMode: proto.BranchNone, Cols: 120, Rows: 36,
	}
	if s.Worktree && s.Branch != "" {
		p.BranchMode, p.Branch = proto.BranchExisting, s.Branch
	}
	go func() {
		var info proto.SessionInfo
		tr, err := cl.TranscriptPage(proto.TranscriptParams{ID: s.ID, Tail: 1})
		if err == nil && tr.Conversation == "" {
			err = errors.New("its conversation isn't recorded yet, so there is nothing to go on with")
		}
		if err == nil {
			p.Resume = tr.Conversation
			info, err = cl.CreateSession(p)
		}
		if err == nil {
			err = cl.KillSession(s.ID)
		}
		a.onMain(func() {
			if info.ID != "" {
				a.upsert(info)
				a.selectSession(info)
			}
			switch {
			case err != nil && info.ID == "":
				a.status = "Couldn't move the session: " + client.Explain(err)
			case err != nil:
				a.status = "Moved, but the old session is still there: " + client.Explain(err)
			}
		})
	}()
}

// removeAccountChip removes the account a chip is of, and what the app
// kept for it.
func (a *App) removeAccountChip(agentID, ref string) {
	st := a.accountsOf()
	i := slices.IndexFunc(st.list, func(acc proto.Account) bool { return acc.ID == ref })
	if i < 0 {
		return
	}
	delete(a.state.AccountNames, a.accountKey(agentID, ref))
	if a.state.ActiveAccounts[a.host.ID+" "+agentID] == ref {
		delete(a.state.ActiveAccounts, a.host.ID+" "+agentID)
	}
	a.saveState()
	a.removeAccount(st, st.list[i])
	a.credits.Refresh()
}

// addNextAccount adds another account of agentID, named "Account 2", "Account
// 3"…, and starts signing it in; its chip can be renamed after.
func (a *App) addNextAccount(agentID string) {
	st := a.accountsOf()
	a.addAccount(st, agentID, fmt.Sprintf("Account %d", len(st.accountsOfAgent(agentID))+2))
}

// creditChip is the title bar's credits button, once connected.
func (a *App) creditChip(c *ui.Context) {
	if a.client == nil {
		return
	}
	a.credits.Chips(c, a.creditsHost())
}

// creditsPage is the Credits page of the connected host.
func (a *App) creditsPage(c *ui.Context) {
	a.viewHeader(c, func() {
		ui.Row(c).Grow(1).MinWidth(0).Children(func() {
			widget.Breadcrumb(c, a.host.Name, "Credits")
		})
		ui.Text(c, a.credits.CheckedLabel()).FontSize(11).TextColor(theme.Muted).SingleLine()
		if widget.GhostButton(c, "Read again").Clicked() {
			a.credits.Refresh()
		}
		if widget.IconButton(c, icon.Close, "Back to sessions", false).Clicked() {
			a.page = ""
		}
	})
	ui.Scroll(c).Grow(1).FillWidth().Children(func() {
		ui.Column(c).FillWidth().Padding(16).Children(func() {
			a.credits.Page(c, a.creditsHost())
		})
	})
}

// signInFromCredits starts a session that signs agent in again, in
// project p, and shows it.
func (a *App) signInFromCredits(p proto.ProjectInfo, agentID, ref string) {
	cl := a.client
	if cl == nil {
		return
	}
	account := ref
	if ref == proto.RefDefault {
		account = ""
	}
	go func() {
		info, err := cl.CreateSession(proto.SessionCreateParams{Project: p.Path, Agent: agentID, SignIn: true, Account: account})
		a.onMain(func() {
			if err != nil {
				a.status = strings.TrimPrefix(err.Error(), proto.MethodSessionCreate+": ")
				return
			}
			a.upsert(info)
			a.selectSession(info)
		})
	}()
}
