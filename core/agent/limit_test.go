package agent

import (
	"strings"
	"testing"
)

func TestLimitReached(t *testing.T) {
	for _, tt := range []struct {
		name, screen string
		want         bool
	}{
		{"claude", "> fix the redirect\n\n● I'll read the code first.\n  ⎿  5-hour limit reached ∙ resets 6:50pm\n     /upgrade to increase your usage limit.\n\n╭────╮\n│ >  │\n╰────╯\n", true},
		{"claude older", "  ⎿  Claude usage limit reached. Your limit will reset at 6pm (Asia/Jakarta).\n\n> \n", true},
		{"codex", "■ You've hit your usage limit. Upgrade to Pro (https://openai.com/chatgpt/pricing) or try again in 2 hours 13 minutes.\n\n› Ask Codex to do anything\n", true},
		{"a reply about limits", "● Added a rate limiter: requests past the limit get a 429.\n\n> \n", false},
		{"long ago", "  ⎿  Usage limit reached · limit resets 6:50pm\n" + strings.Repeat("● more work\n", 20), false},
	} {
		if got := LimitReached(tt.screen); got != tt.want {
			t.Errorf("%s: LimitReached = %v, want %v", tt.name, got, tt.want)
		}
	}
}
