// Package screenmenu finds the menu an agent waits on in its terminal's
// text: a question with numbered choices, as Claude Code and Codex ask for
// a permission or an answer, so the app can offer the choices elsewhere
// and answer by typing the choice's key.
package screenmenu

import (
	"regexp"
	"strconv"
	"strings"
)

// Menu is a question and its choices.
type Menu struct {
	Question string
	Detail   []string // what it is about: the command, the file, a line or a few
	Choices  []Choice
}

// Choice is one answer and the key that picks it.
type Choice struct {
	Label string
	Key   string // what to type: the choice's number, its letter, or Escape
}

var (
	choiceLine = regexp.MustCompile(`^(\s*)([❯›>▶]\s*)?(\d{1,2})[.)]\s+(\S.*?)\s*$`)
	keyHint    = regexp.MustCompile(`\s*\((esc|[a-z])\)$`)
	borders    = "│┃║|╭╮╰╯─━═┌┐└┘"
)

// lookBack is how far above the screen's end a menu may start: one
// waiting for an answer is at the bottom.
const lookBack = 30

// Parse finds the last menu on screen; false when there is none.
func Parse(screen string) (Menu, bool) {
	lines := strings.Split(strings.ReplaceAll(screen, "\r", ""), "\n")
	for i := range lines {
		lines[i] = clean(lines[i])
	}
	last := -1
	for i := len(lines) - 1; i >= max(0, len(lines)-lookBack); i-- {
		if choiceLine.MatchString(lines[i]) {
			last = i
			break
		}
	}
	if last < 0 {
		return Menu{}, false
	}
	// Walk up over the choices and their descriptions to the first one.
	type found struct {
		n     int
		label string
	}
	var choices []found
	first, indent, cursor := last, -1, false
	for i := last; i >= 0; i-- {
		line := lines[i]
		if m := choiceLine.FindStringSubmatch(line); m != nil {
			n, _ := strconv.Atoi(m[3])
			if len(choices) > 0 && n != choices[0].n-1 {
				break
			}
			choices = append([]found{{n, m[4]}}, choices...)
			first, indent = i, len(m[1])
			cursor = cursor || m[2] != ""
			if n == 1 {
				break
			}
			continue
		}
		if strings.TrimSpace(line) == "" || leading(line) > indent+2 && indent >= 0 {
			continue // a blank line, or a choice's description
		}
		break
	}
	// A menu has a cursor on a choice; a numbered list in a reply doesn't.
	if len(choices) < 2 || choices[0].n != 1 || !cursor {
		return Menu{}, false
	}
	var m Menu
	for _, c := range choices {
		label, key := c.label, strconv.Itoa(c.n)
		if h := keyHint.FindStringSubmatch(label); h != nil {
			label = strings.TrimSpace(label[:len(label)-len(h[0])])
			if h[1] == "esc" {
				key = "\x1b"
			} else {
				key = h[1]
			}
		}
		if strings.HasPrefix(label, "Type something") || strings.HasPrefix(label, "Chat about this") {
			continue // answered in the session itself
		}
		m.Choices = append(m.Choices, Choice{Label: label, Key: key})
	}
	// The question is the line above the choices; what it is about, the
	// lines above it up to a gap or a rule.
	q := first - 1
	for q >= 0 && strings.TrimSpace(lines[q]) == "" {
		q--
	}
	if q < 0 {
		return Menu{}, false
	}
	m.Question = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(lines[q]), "☐☒✔ "))
	for i := q - 1; i >= 0 && len(m.Detail) < 4; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			if len(m.Detail) > 0 {
				break
			}
			continue
		}
		if rule(lines[i]) {
			break
		}
		m.Detail = append([]string{strings.TrimSpace(strings.TrimLeft(line, "☐☒✔ "))}, m.Detail...)
	}
	return m, len(m.Choices) > 0
}

// clean drops a line's box borders and trailing spaces.
func clean(line string) string {
	line = strings.TrimRight(line, " \t")
	t := strings.TrimSpace(line)
	if strings.ContainsRune("│┃║|", firstRune(t)) && strings.ContainsRune("│┃║|", lastRune(t)) && len([]rune(t)) > 1 {
		lead := len(line) - len(strings.TrimLeft(line, " "))
		inner := strings.TrimSpace(string([]rune(t)[1 : len([]rune(t))-1]))
		body := []rune(t)[1 : len([]rune(t))-1]
		pad := len(string(body)) - len(strings.TrimLeft(string(body), " "))
		return strings.Repeat(" ", lead+pad) + inner
	}
	return line
}

// rule reports whether line is only box drawing: a menu's edge.
func rule(line string) bool {
	t := strings.TrimSpace(line)
	return t != "" && strings.Trim(t, borders+" ") == ""
}

func leading(line string) int { return len(line) - len(strings.TrimLeft(line, " ")) }

func firstRune(s string) rune {
	for _, r := range s {
		return r
	}
	return 0
}

func lastRune(s string) rune {
	r := []rune(s)
	if len(r) == 0 {
		return 0
	}
	return r[len(r)-1]
}
