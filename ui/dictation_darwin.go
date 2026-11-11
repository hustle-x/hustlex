package ui

import "github.com/ebitengine/purego/objc"

// startDictation asks macOS to start its dictation, which types into the
// text input with the focus; false when nothing took the request.
func startDictation() bool {
	app := objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication"))
	if app == 0 {
		return false
	}
	return objc.Send[bool](app, objc.RegisterName("sendAction:to:from:"), objc.RegisterName("startDictation:"), objc.ID(0), objc.ID(0))
}

// dictationHint is said when dictation couldn't start.
const dictationHint = "Turn on Dictation in System Settings → Keyboard, then try again"
