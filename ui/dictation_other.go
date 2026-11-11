//go:build !darwin && !windows

package ui

func startDictation() bool { return false }

const dictationHint = "Use your desktop's dictation: it types into the message"
