package screenmenu

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name   string
		screen string
		want   Menu
		ok     bool
	}{
		{"claude permission in a box", `
● I'll install the SDK.

╭────────────────────────────────────────────────────────────╮
│ Bash command                                               │
│                                                            │
│   npm install firebase-admin                               │
│   Install the Firebase Admin SDK                           │
│                                                            │
│ Do you want to proceed?                                    │
│ ❯ 1. Yes                                                   │
│   2. Yes, and don't ask again for npm install commands     │
│   3. No, and tell Claude what to do differently (esc)      │
╰────────────────────────────────────────────────────────────╯
`, Menu{Question: "Do you want to proceed?", Detail: []string{"npm install firebase-admin", "Install the Firebase Admin SDK"},
			Choices: []Choice{{"Yes", "1"}, {"Yes, and don't ask again for npm install commands", "2"}, {"No, and tell Claude what to do differently", "\x1b"}}}, true},
		{"claude question with descriptions", `
────────────────────────────────────────
 ☐ Settings route

Keep the old settings route as a redirect?

❯ 1. Yes, keep a redirect
     Old links keep working.
  2. No, remove it
     Links to it break.
  3. Type something.

Enter to select · ↑/↓ to navigate · Esc to cancel
`, Menu{Question: "Keep the old settings route as a redirect?", Detail: []string{"Settings route"},
			Choices: []Choice{{"Yes, keep a redirect", "1"}, {"No, remove it", "2"}}}, true},
		{"codex approval", `
  Would you like to run the following command?

  $ npm install firebase-admin

› 1. Yes, proceed (y)
  2. Yes, and don't ask again for this command (a)
  3. No, and tell Codex what to do differently (esc)

  Press enter to confirm or esc to cancel
`, Menu{Question: "$ npm install firebase-admin", Detail: []string{"Would you like to run the following command?"},
			Choices: []Choice{{"Yes, proceed", "y"}, {"Yes, and don't ask again for this command", "a"}, {"No, and tell Codex what to do differently", "\x1b"}}}, true},
		{"a numbered list in a reply", `
● Done. I changed:
  1. the redirect
  2. its test

> 
`, Menu{}, false},
	}
	for _, c := range cases {
		got, ok := Parse(c.screen)
		if ok != c.ok || ok && !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %v %#v", c.name, ok, got)
		}
	}
}
