package ui

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/client"
	"github.com/hustlex/hustlex/core/screenmenu"
	"github.com/hustlex/hustlex/proto"
	"github.com/hustlex/hustlex/ui/icon"
	"github.com/hustlex/hustlex/ui/textfmt"
	"github.com/hustlex/hustlex/ui/theme"
	"github.com/hustlex/hustlex/ui/widget"
)

// The Inbox lists what agents wait on: each waiting session's question,
// read off its terminal, with its choices to answer from here.

// inboxEvery is how often the Inbox reads the waiting sessions' screens.
const inboxEvery = 2 * time.Second

// inboxState belongs to the main thread; the poller fills menus.
type inboxState struct {
	menus    map[string]inboxMenu // session ID → its menu, as last read
	since    map[string]time.Time // session ID → when it was first seen waiting
	answered map[string]time.Time // session ID → when it was answered here
	polling  *int                 // the poller's token while one runs
}

type inboxMenu struct {
	menu screenmenu.Menu
	ok   bool
}

// waiting are the sessions whose agent waits for the user, oldest wait
// first, but those just answered here.
func (a *App) waiting() []proto.SessionInfo {
	if a.inbox.since == nil {
		a.inbox.since = map[string]time.Time{}
	}
	var out []proto.SessionInfo
	seen := map[string]bool{}
	for _, s := range a.sessions {
		if !waitsForYou(s) {
			delete(a.inbox.answered, s.ID)
			continue
		}
		seen[s.ID] = true
		if _, ok := a.inbox.since[s.ID]; !ok {
			a.inbox.since[s.ID] = time.Now()
		}
		if at, ok := a.inbox.answered[s.ID]; ok && time.Since(at) < 10*time.Second {
			continue // its agent hasn't said it goes on yet
		}
		out = append(out, s)
	}
	for id := range a.inbox.since {
		if !seen[id] {
			delete(a.inbox.since, id)
		}
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && a.inbox.since[out[j].ID].Before(a.inbox.since[out[j-1].ID]); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// followInbox reads the waiting sessions' screens while the Inbox shows.
func (a *App) followInbox(cl *client.Client) {
	if a.inbox.polling != nil {
		return
	}
	tok, gen := new(int), a.gen
	a.inbox.polling = tok
	go func() {
		for {
			ids := make(chan []string, 1)
			a.onMain(func() {
				if a.page != pageInbox || gen != a.gen || a.inbox.polling != tok {
					if a.inbox.polling == tok {
						a.inbox.polling = nil
					}
					ids <- nil
					return
				}
				var list []string
				for _, s := range a.waiting() {
					list = append(list, s.ID)
				}
				ids <- append(list, "") // "" marks a live poller with nothing to read
			})
			list := <-ids
			if list == nil {
				return
			}
			read := map[string]inboxMenu{}
			for _, id := range list[:len(list)-1] {
				if text, err := cl.Screen(id); err == nil {
					m, ok := screenmenu.Parse(text)
					read[id] = inboxMenu{m, ok}
				}
			}
			a.onMain(func() {
				if a.inbox.menus == nil {
					a.inbox.menus = map[string]inboxMenu{}
				}
				for id, m := range read {
					a.inbox.menus[id] = m
				}
			})
			select {
			case <-cl.Done():
				return
			case <-time.After(inboxEvery):
			}
		}
	}()
}

// inboxPage is the Inbox: a card per waiting session.
func (a *App) inboxPage(c *ui.Context) {
	if a.client != nil {
		a.followInbox(a.client)
	}
	a.viewHeader(c, func() {
		ui.Row(c).Grow(1).MinWidth(0).Children(func() {
			widget.Breadcrumb(c, a.host.Name, "Inbox")
		})
		if widget.IconButton(c, icon.Close, "Back to sessions", false).Clicked() {
			a.page = ""
		}
	})
	waiting := a.waiting()
	ui.Scroll(c).Grow(1).FillWidth().Children(func() {
		ui.Column(c).FillWidth().Padding(22, 24).AlignItems(ui.Center).Children(func() {
			ui.Column(c).FillWidth().MaxWidth(720).Gap(10).Children(func() {
				ui.Text(c, "Inbox").FontSize(17)
				ui.Text(c, "What your agents are waiting on. Answer here or in the session.").FontSize(12.5).TextColor(theme.Muted).Margin(0, 0, 4, 0)
				if len(waiting) == 0 {
					ui.Row(c).FillWidth().Padding(40, 0).Justify(ui.Center).Gap(8).Children(func() {
						ui.Icon(c, icon.Check).FontSize(14).TextColor(theme.Done)
						ui.Text(c, "Nothing waiting on you.").FontSize(12.5).TextColor(theme.Muted)
					})
					return
				}
				for _, s := range waiting {
					a.inboxCard(c.Key(s.ID), s)
				}
			})
		})
	})
}

// inboxCard is one waiting session: what it asks and how to answer.
func (a *App) inboxCard(c *ui.Context, s proto.SessionInfo) {
	m, read := a.inbox.menus[s.ID]
	widget.Card(c).FillWidth().Gap(8).Children(func() {
		ui.Row(c).FillWidth().Gap(9).AlignItems(ui.Center).Children(func() {
			widget.Avatar(c, icon.Agent(s.Agent), 24, icon.AgentTint(s.Agent, theme.Text))
			ui.Text(c, a.sessionName(s)).FontSize(13).SingleLine().Shrink(0).MaxWidth(260)
			question := "needs your answer"
			switch {
			case read && m.ok:
				question = m.menu.Question
			case !read:
				question = "Reading its question…"
			}
			ui.Text(c, question).FontSize(12.5).TextColor(theme.Muted).SingleLine().Shrink(1)
			ui.Spacer(c)
			if at, ok := a.inbox.since[s.ID]; ok {
				ui.Text(c, textfmt.Age(at, time.Now())+" ago").FontSize(11.5).TextColor(theme.Muted).SingleLine().Shrink(0)
			}
		})
		if read && m.ok && len(m.menu.Detail) > 0 {
			ui.Text(c, strings.Join(m.menu.Detail, "\n")).Font(theme.MonoFont).FontSize(12).TextColor(theme.TextSoft).
				FillWidth().Padding(8, 10).Radius(8).Background(theme.Sunken).Border(1, theme.Border)
		}
		ui.Row(c).FillWidth().Gap(8).Wrap().AlignItems(ui.Center).Children(func() {
			if read && m.ok {
				for i, ch := range m.menu.Choices {
					kind := widget.Secondary
					if i == 0 {
						kind = widget.Primary
					}
					if widget.Button(c.Key(i), kind, nil, ch.Label, "").Clicked() {
						a.answerInbox(s, ch)
					}
				}
			}
			ui.Spacer(c)
			if widget.GhostButton(c.Key("open"), "Open session").Clicked() {
				a.page = ""
				a.selectSession(s)
			}
		})
	})
}

// answerInbox types choice ch's key into session s.
func (a *App) answerInbox(s proto.SessionInfo, ch screenmenu.Choice) {
	cl := a.client
	if cl == nil {
		return
	}
	if a.inbox.answered == nil {
		a.inbox.answered = map[string]time.Time{}
	}
	a.inbox.answered[s.ID] = time.Now()
	delete(a.inbox.menus, s.ID)
	name := a.sessionName(s)
	go func() {
		err := cl.Input(s.ID, ch.Key, false, false)
		a.onMain(func() {
			if err != nil {
				delete(a.inbox.answered, s.ID)
				a.status = err.Error()
				return
			}
			a.notify(name + ": " + ch.Label)
		})
	}()
}
