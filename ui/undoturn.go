package ui

import (
	"github.com/hustlex/hustlex/client"
	"github.com/hustlex/hustlex/proto"
	"github.com/hustlex/hustlex/ui/textfmt"
)

// undoTurn puts the files session s's agent changed in its last turn back
// as they were (redo: forth again).
func (a *App) undoTurn(s proto.SessionInfo, redo bool) {
	cl := a.client
	if cl == nil {
		return
	}
	go func() {
		files, err := cl.UndoTurn(s.ID, redo)
		a.onMain(func() {
			if err != nil {
				a.status = "Couldn't undo the turn: " + client.Explain(err)
				return
			}
			n := count(len(files), "file", "files")
			if redo {
				a.notify(n + " " + textfmt.Plural(len(files), "has", "have") + " the turn's changes again")
			} else {
				a.notify(n + " restored to before this turn")
			}
		})
	}()
}

// dictate starts the system's dictation into the chat's composer, or
// says how to.
func (a *App) dictate() {
	if !startDictation() {
		a.notify(dictationHint)
	}
}
