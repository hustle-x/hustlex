package ui

import (
	"slices"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/proto"
	"github.com/hustlex/hustlex/ui/widget"
)

// navOrder is the order of the pages in the nav bar, the sessions first:
// a page further along slides in from the right, one before it from the
// left, as tabs do.
var navOrder = []string{"", pageNotes, pageCanvas, pageVault, pageCredits, pageUsage, pageSync, pageAppearance, pageSettings, pageHosts}

// viewMotion is how the view the workspace shows now comes in: from the
// side of its nav item for another page, from above or below for another
// session, as the list has them. The view going away fades where it is,
// under the new one.
func (a *App) viewMotion(view string) ui.ElementTransition {
	if view != a.motion.view {
		a.motion.x, a.motion.y = 0, 0
		page, last := slices.Index(navOrder, a.page), slices.Index(navOrder, a.motion.page)
		switch {
		case a.page != a.motion.page && page >= 0 && last >= 0:
			a.motion.x = 28
			if page < last {
				a.motion.x = -28
			}
		case a.page == "" && a.motion.page == "":
			now, before := a.sessionRank(a.selected), a.sessionRank(a.motion.session)
			if now >= 0 && before >= 0 {
				a.motion.y = 16
				if now < before {
					a.motion.y = -16
				}
			}
		}
		a.motion.view, a.motion.page, a.motion.session = view, a.page, a.selected
	}
	return ui.ElementTransition{
		Animation: &widget.Spring,
		Enter:     &ui.Motion{X: a.motion.x, Y: a.motion.y, Opacity: 0},
		Exit:      &ui.Motion{Opacity: 0, Scale: 0.985},
	}
}

// viewMotionState is what viewMotion keeps between frames.
type viewMotionState struct {
	view, page, session string
	x, y                float32
}

// sessionRank is where session id is in the Sessions list, -1 if it isn't.
func (a *App) sessionRank(id string) int {
	return slices.IndexFunc(a.sessions, func(s proto.SessionInfo) bool { return s.ID == id })
}
