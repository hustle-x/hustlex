package ui

import (
	"github.com/hustlex/hustlex/proto"
)

// Near its limit an account stops being the one new sessions start as, and
// the one with the most room takes over: running sessions are left alone
// (the chip's "Continue here" moves one by hand).
const (
	switchAtUsed = 95.0 // the active account's tightest limit, in percent
	switchToUsed = 80.0 // another account must be under this to take over
)

// usedOf is how much of its tightest limit an account has used, and whether
// there is a reading to go by.
func usedOf(acc proto.CreditAccount) (float64, bool) {
	if acc.Error != "" || len(acc.Windows) == 0 {
		return 0, false
	}
	used := 0.0
	for _, w := range acc.Windows {
		used = max(used, w.Used)
	}
	return used, true
}

// autoSwitchAccounts moves each agent's active account on when it is full,
// if the user turned that on. It is cheap, so the view calls it every frame.
func (a *App) autoSwitchAccounts() {
	if !a.state.AutoSwitchAccounts {
		return
	}
	byAgent := map[string][]proto.CreditAccount{}
	for _, acc := range a.credits.Accounts() {
		byAgent[acc.Agent] = append(byAgent[acc.Agent], acc)
	}
	for agentID, accs := range byAgent {
		if len(accs) < 2 {
			continue
		}
		active := a.activeRef(agentID)
		var current *proto.CreditAccount
		for i := range accs {
			if accs[i].Ref == active {
				current = &accs[i]
			}
		}
		if current == nil {
			continue
		}
		if used, ok := usedOf(*current); !ok || used < switchAtUsed {
			continue
		}
		best, bestUsed := -1, switchToUsed
		for i, acc := range accs {
			if u, ok := usedOf(acc); ok && acc.Ref != active && u < bestUsed {
				best, bestUsed = i, u
			}
		}
		if best < 0 {
			continue
		}
		name := a.accountChipName(agentID, accs[best].Ref)
		if name == "" {
			name = accs[best].Account
		}
		a.switchAccount(agentID, accs[best].Ref, name+" (the other is nearly full)")
	}
}
