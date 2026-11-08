// Package credits reads what each agent account signed in on this machine
// has left of its plan, with the agent's own login: Claude Code's and
// Codex's tokens go to the vendor's usage endpoint, the only place they are
// sent, and Kiro CLI is asked for its /usage. Tokens are read when needed,
// never kept, logged or refreshed: refreshing one would sign the agent out.
package credits

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/hustlex/hustlex/proto"
)

// ErrNotSignedIn is the Error of an account that has no login yet: it is
// read again soon (every minGap), to show it once it has one.
const ErrNotSignedIn = "Not signed in yet"

// transientGap is how soon an account whose reading failed for now (the
// vendor limits requests, the network is down) is read again: not sooner,
// or the limit only lasts.
const transientGap = 2 * time.Minute

// Readings older than staleAfter are read again when asked for; a refresh
// asked for sooner than minGap after the last one waits for it.
const (
	staleAfter  = 5 * time.Minute
	minGap      = 30 * time.Second
	readTimeout = 45 * time.Second
)

// Home is where one of an agent's other accounts keeps its sign-in.
type Home struct {
	ID    string // proto.Account.ID
	Agent string
	Dir   string
}

// provider reads the accounts of one agent, its own and those of homes. It
// returns none when the agent is not installed or not signed in.
type provider func(ctx context.Context, homes []Home) []proto.CreditAccount

var providers = []provider{readClaude, readCodex, readKiro}

// Checker keeps the latest reading. Its fields are guarded by mu; the
// reading itself runs on a goroutine of its own, one at a time.
type Checker struct {
	// Homes lists the other accounts the host keeps; nil for none.
	Homes func() []Home

	mu       sync.Mutex
	report   proto.CreditsReport
	started  time.Time
	checking bool
	retry    time.Duration // when an account needs it, how soon to read again
}

// Invalidate makes the next Report read again: an account was added or
// removed, so the last reading no longer lists them all.
func (c *Checker) Invalidate() {
	c.mu.Lock()
	c.started = time.Time{}
	c.mu.Unlock()
}

// Report returns the latest reading, and starts a new one when it is
// stale, or when refresh asks and the last one is not too recent.
func (c *Checker) Report(refresh bool) proto.CreditsReport {
	c.mu.Lock()
	defer c.mu.Unlock()
	since := time.Since(c.started)
	if !c.checking && (since > staleAfter || refresh && since > minGap || c.retry > 0 && since > c.retry) {
		c.checking, c.started = true, time.Now()
		go c.read()
	}
	r := c.report
	r.Checking = c.checking
	return r
}

func (c *Checker) read() {
	ctx, cancel := context.WithTimeout(context.Background(), readTimeout)
	defer cancel()
	var homes []Home
	if c.Homes != nil {
		homes = c.Homes()
	}
	results := make([][]proto.CreditAccount, len(providers))
	var wg sync.WaitGroup
	for i, p := range providers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = p(ctx, homes)
		}()
	}
	wg.Wait()
	accounts := []proto.CreditAccount{}
	for _, r := range results {
		accounts = append(accounts, r...)
	}
	c.mu.Lock()
	c.retry = retryAfter(accounts)
	c.report = proto.CreditsReport{Accounts: keepLastReading(accounts, c.report.Accounts), Checked: time.Now()}
	c.checking = false
	c.mu.Unlock()
}

// signInProblem reports whether an account has no reading for want of a
// sign-in, rather than for now.
func signInProblem(a proto.CreditAccount) bool {
	return a.Error == ErrNotSignedIn || a.Error == errSignedOut.Error()
}

// retryAfter is how soon accounts should be read again, before they are
// stale: soon when one waits for a sign-in, later when one's reading
// failed for now; 0 for neither.
func retryAfter(accounts []proto.CreditAccount) time.Duration {
	var after time.Duration
	for _, a := range accounts {
		switch {
		case a.Error == ErrNotSignedIn:
			return minGap
		case a.Error != "" && !signInProblem(a):
			after = transientGap
		}
	}
	return after
}

// keepLastReading gives an account whose reading failed for now its last
// good one, so a chip doesn't go blank over a hiccup.
func keepLastReading(accounts, last []proto.CreditAccount) []proto.CreditAccount {
	for i, a := range accounts {
		if a.Ref == "" || a.Error == "" || signInProblem(a) || len(a.Windows) > 0 {
			continue
		}
		j := slices.IndexFunc(last, func(o proto.CreditAccount) bool {
			return o.Agent == a.Agent && o.Ref == a.Ref && len(o.Windows) > 0
		})
		if j >= 0 {
			accounts[i] = last[j]
			accounts[i].Note = "Last reading kept: " + a.Error
		}
	}
	return accounts
}
