package ui

// startDictation can't start Windows' voice typing for the app: the user
// presses Win+H, which dictationHint says.
func startDictation() bool { return false }

const dictationHint = "Press Win+H: Windows types what you say into the message"
