package agent

import "strings"

// limitLook is how many of a screen's last non-blank lines LimitReached
// reads: an agent says its limit ran out where its reply ends.
const limitLook = 15

// LimitReached reports whether an agent's screen ends saying its account
// ran out of a usage limit, as Claude Code ("5-hour limit reached ∙
// resets 6pm", "Claude usage limit reached") and Codex ("You've hit your
// usage limit") do. Words about limits in a reply can match too, so
// callers confirm it with the account's reading.
func LimitReached(screen string) bool {
	lines := strings.Split(screen, "\n")
	seen := 0
	for i := len(lines) - 1; i >= 0 && seen < limitLook; i-- {
		line := strings.ToLower(strings.TrimSpace(lines[i]))
		if line == "" {
			continue
		}
		seen++
		switch {
		case strings.Contains(line, "usage limit reached"),
			strings.Contains(line, "hit your usage limit"),
			strings.Contains(line, "hit your limit"),
			strings.Contains(line, "limit reached") && strings.Contains(line, "reset"):
			return true
		}
	}
	return false
}
