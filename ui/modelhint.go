package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/hustlex/hustlex/core/agent"
	"github.com/hustlex/hustlex/ui/creditview"
	"github.com/hustlex/hustlex/ui/icon"
	"github.com/hustlex/hustlex/ui/theme"
)

// A small task doesn't need Claude Code's default model: the New session
// form offers a cheaper one when the first prompt reads like one.

const (
	hintDefault = "claude-opus-5-5" // what Claude Code starts on unless told
	hintCheaper = "claude-sonnet-5-5"
)

// smallWords say a prompt asks for a small change; bigWords that it
// doesn't, whatever else it says.
var (
	smallWords = []string{"typo", "rename", "bump", "version", "comment", "readme", "spelling", "wording", "label", "copy text",
		"button text", "color", "colour", "padding", "margin", "lint", "format", "unused", "small", "tweak", "minor", "quick", "one line", "log message"}
	bigWords = []string{"refactor", "architect", "redesign", "migrate", "implement", "build", "investigate", "debug", "why", "design",
		"rewrite", "entire", "whole", "all the", "every", "performance", "security", "plan"}
)

// smallTask reports whether prompt reads like a small change: short, one
// paragraph, saying so, and not saying it is more.
func smallTask(prompt string) bool {
	p := strings.ToLower(strings.TrimSpace(prompt))
	if p == "" || len([]rune(p)) > 160 || strings.Count(p, "\n") > 1 {
		return false
	}
	for _, w := range bigWords {
		if strings.Contains(p, w) {
			return false
		}
	}
	for _, w := range smallWords {
		if strings.Contains(p, w) {
			return true
		}
	}
	return false
}

// costShare says a price ratio in words: "half", "a fifth".
func costShare(r float64) string {
	switch {
	case r >= 0.4 && r <= 0.6:
		return "half"
	case r >= 0.28 && r < 0.4:
		return "a third"
	case r >= 0.22 && r < 0.28:
		return "a quarter"
	case r >= 0.17 && r < 0.22:
		return "a fifth"
	}
	return fmt.Sprintf("%.0f%%", r*100)
}

// modelHint is the form's offer of a cheaper model for a small task, or
// the model picked from it.
func (a *App) modelHint(c *ui.Context) {
	f := &a.newSession
	if agent.Base(f.agent) != "claude" {
		return
	}
	cheap, deflt := modelLabel(hintCheaper), modelLabel(hintDefault)
	if f.model == hintCheaper {
		hintBox(c, func() {
			ui.Icon(c, icon.Check).FontSize(12).TextColor(theme.Done)
			ui.Row(c).Grow(1).MinWidth(0).Gap(4).Wrap().Children(func() {
				ui.Text(c, "This session runs on").FontSize(12).TextColor(theme.TextSoft)
				ui.Text(c, cheap+".").FontSize(12).TextColor(theme.Text)
			})
			if hintButton(c.Key("undo"), "Use "+deflt) {
				f.model = ""
				f.hintOff = true
			}
		})
		return
	}
	r, ok := creditview.PriceRatio(hintCheaper, hintDefault)
	if f.hintOff || !ok || !smallTask(f.prompt) {
		return
	}
	hintBox(c, func() {
		ui.Icon(c, icon.Lightbulb).FontSize(12).TextColor(theme.Working)
		ui.Row(c).Grow(1).MinWidth(0).Gap(4).Wrap().Children(func() {
			words(c, "Looks like a small task.", theme.TextSoft)
			ui.Text(c, cheap).FontSize(12).TextColor(theme.Text)
			words(c, "does it for about "+costShare(r)+" of "+deflt+"'s cost.", theme.TextSoft)
		})
		if hintButton(c.Key("use"), "Use "+strings.Fields(cheap)[0]) {
			f.model = hintCheaper
		}
		if hintButton(c.Key("keep"), "Keep "+strings.Fields(deflt)[0]) {
			f.hintOff = true
		}
	})
}

// words draws text a word at a time, so a wrapping row breaks it
// between words.
func words(c *ui.Context, text string, col ui.Color) {
	for i, w := range strings.Fields(text) {
		ui.Text(c.Key(text+strconv.Itoa(i)), w).FontSize(12).TextColor(col)
	}
}

// modelLabel is Claude Code's name for model id: "Sonnet 5.5".
func modelLabel(id string) string {
	for _, m := range agent.SettingsOf("claude").Models {
		if m.ID == id {
			return m.Label
		}
	}
	return id
}

func hintBox(c *ui.Context, content func()) {
	ui.Row(c.Key("model-hint")).FillWidth().Padding(10, 12).Gap(10).Radius(10).AlignItems(ui.Center).
		Background(theme.Fade(theme.Working, 0.07)).Border(1, theme.Fade(theme.Working, 0.2)).Children(content)
}

func hintButton(c *ui.Context, label string) bool {
	b := ui.ButtonBase(c).Height(24).PaddingX(9).Radius(7).Border(1, theme.BorderStrong).Cursor(ui.CursorPointer).Label(label).Shrink(0)
	col := theme.TextSoft
	if b.Hovered() {
		col = theme.Text
	}
	b.Children(func() { ui.Text(c, label).FontSize(11.5).TextColor(col).SingleLine() })
	return b.Clicked()
}
