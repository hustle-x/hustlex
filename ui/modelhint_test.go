package ui

import (
	"testing"

	"github.com/hustlex/hustlex/ui/creditview"
)

func TestSmallTask(t *testing.T) {
	for prompt, want := range map[string]bool{
		"Fix the typo in the README":                            true,
		"bump the version to 2.1":                               true,
		"Change the button label to Save":                       true,
		"Refactor the auth module and fix the typo":             false,
		"Investigate why the login redirect loses the page":     false,
		"Fix the login redirect after sign in":                  false,
		"":                                                      false,
		"rename the variable\nand\nalso the other three things": false,
	} {
		if got := smallTask(prompt); got != want {
			t.Errorf("smallTask(%q) = %v, want %v", prompt, got, want)
		}
	}
	r, ok := creditview.PriceRatio(hintCheaper, hintDefault)
	if !ok || costShare(r) != "half" {
		t.Fatalf("Sonnet 5.5 costs %v of Opus 5.5 (%q)", r, costShare(r))
	}
	if modelLabel(hintCheaper) != "Sonnet 5.5" || modelLabel(hintDefault) != "Opus 5.5" {
		t.Fatal("model labels")
	}
}
