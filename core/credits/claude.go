package credits

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hustlex/hustlex/proto"
)

const claudeUsageURL = "https://api.anthropic.com/api/oauth/usage"

// claudeCreds is Claude Code's login, as its keychain item or
// .credentials.json holds it.
type claudeCreds struct {
	OAuth *struct {
		AccessToken      string `json:"accessToken"`
		ExpiresAt        int64  `json:"expiresAt"` // Unix milliseconds
		SubscriptionType string `json:"subscriptionType"`
	} `json:"claudeAiOauth"`
}

// claudeLimit is one limit of Claude's usage answer.
type claudeLimit struct {
	Utilization *float64 `json:"utilization"`
	ResetsAt    string   `json:"resets_at"`
}

// claudeLabels names Claude's limits, in the order they show.
var claudeLabels = []struct{ key, label string }{
	{"five_hour", "5 hours"},
	{"seven_day", "Week"},
	{"seven_day_opus", "Week · Opus"},
	{"seven_day_sonnet", "Week · Sonnet"},
}

// claudeKeychainBase is the keychain item of Claude Code's own
// configuration folder; another folder's has a suffix of its path's hash.
const claudeKeychainBase = "Claude Code-credentials"

// claudeServiceOf names the keychain item that holds the login of the
// Claude Code configuration folder at dir.
func claudeServiceOf(dir string) string {
	sum := sha256.Sum256([]byte(dir))
	return claudeKeychainBase + "-" + hex.EncodeToString(sum[:])[:8]
}

// keychainItem is a login in the keychain, by the item's name.
type keychainItem struct {
	service string
	raw     []byte
}

func claudeDir() string {
	if d := os.Getenv("CLAUDE_CONFIG_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

// claudeLogin is the Claude Code login of one account: ref is its
// proto.CreditAccount.Ref.
type claudeLogin struct {
	raw []byte
	ref string
}

// claudeLogins finds the logins of the default account and of each of
// homes' accounts: in the keychain, and in the .credentials.json of their
// folders. Other logins on the machine (old configuration folders leave
// theirs behind) aren't hustleX's, and aren't read.
func claudeLogins(ctx context.Context, homes []Home) []claudeLogin {
	byService := map[string]string{claudeKeychainBase: proto.RefDefault}
	services := []string{claudeKeychainBase}
	for _, h := range homes {
		if h.Agent == "claude" {
			svc := claudeServiceOf(h.Dir)
			byService[svc] = h.ID
			services = append(services, svc)
		}
	}
	var out []claudeLogin
	for _, item := range claudeKeychain(ctx, services) {
		out = append(out, claudeLogin{item.raw, byService[item.service]})
	}
	if file, err := os.ReadFile(filepath.Join(claudeDir(), ".credentials.json")); err == nil {
		out = append(out, claudeLogin{file, proto.RefDefault})
	}
	for _, h := range homes {
		if file, err := os.ReadFile(filepath.Join(h.Dir, ".credentials.json")); err == nil && h.Agent == "claude" {
			out = append(out, claudeLogin{file, h.ID})
		}
	}
	return out
}

// readClaude reads the plan of the default Claude Code account and of each
// other account of homes, in that order. An account with no login yet is
// listed too, to sign it in.
func readClaude(ctx context.Context, homes []Home) []proto.CreditAccount {
	refs := []string{proto.RefDefault}
	emails := map[string]string{
		proto.RefDefault: claudeEmail(filepath.Join(userHome(), ".claude.json"), os.Getenv("CLAUDE_CONFIG_DIR")),
	}
	for _, h := range homes {
		if h.Agent == "claude" {
			refs = append(refs, h.ID)
			emails[h.ID] = claudeEmail(filepath.Join(h.Dir, ".claude.json"), "")
		}
	}
	// An account has one login: when its keychain item and its file
	// differ, the first live one is the one it has.
	got := map[string]proto.CreditAccount{}
	for _, login := range claudeLogins(ctx, homes) {
		var creds claudeCreds
		if json.Unmarshal(login.raw, &creds) != nil || creds.OAuth == nil || creds.OAuth.AccessToken == "" {
			continue
		}
		if prev, ok := got[login.ref]; ok && prev.Error != errSignedOut.Error() {
			continue
		}
		if exp := creds.OAuth.ExpiresAt; exp > 0 && time.Now().After(time.UnixMilli(exp)) {
			if _, ok := got[login.ref]; !ok {
				got[login.ref] = proto.CreditAccount{Agent: "claude", Ref: login.ref, Account: emails[login.ref], Error: errSignedOut.Error()}
			}
			continue
		}
		acc := readClaudeLogin(ctx, creds)
		acc.Ref, acc.Account = login.ref, emails[login.ref]
		got[login.ref] = acc
	}
	var out []proto.CreditAccount
	for _, ref := range refs {
		switch acc, ok := got[ref]; {
		case ok:
			out = append(out, acc)
		case ref != proto.RefDefault:
			out = append(out, proto.CreditAccount{Agent: "claude", Ref: ref, Account: emails[ref], Error: ErrNotSignedIn})
		}
	}
	return out
}

func userHome() string {
	home, _ := os.UserHomeDir()
	return home
}

// readClaudeLogin asks Anthropic for the limits of one login.
func readClaudeLogin(ctx context.Context, creds claudeCreds) proto.CreditAccount {
	acc := proto.CreditAccount{Agent: "claude", Plan: titleCase(creds.OAuth.SubscriptionType)}
	var answer map[string]json.RawMessage
	err := getJSON(ctx, claudeUsageURL, map[string]string{
		"Authorization":  "Bearer " + creds.OAuth.AccessToken,
		"anthropic-beta": "oauth-2025-04-20",
	}, &answer)
	if err != nil {
		acc.Error = err.Error()
		return acc
	}
	return claudeAccount(acc, answer)
}

// claudeListed is a limit of Claude's usage answer's list, the limits
// Claude Code itself shows: the session, the week, and the week of one
// model.
type claudeListed struct {
	Kind     string  `json:"kind"`
	Percent  float64 `json:"percent"`
	ResetsAt string  `json:"resets_at"`
	Scope    *struct {
		Model *struct {
			Name string `json:"display_name"`
		} `json:"model"`
	} `json:"scope"`
}

// claudeListedLabel names a listed limit, "" for kinds hustleX doesn't know.
func claudeListedLabel(l claudeListed) string {
	switch l.Kind {
	case "session":
		return "5 hours"
	case "weekly_all":
		return "Week"
	case "weekly_scoped":
		if l.Scope != nil && l.Scope.Model != nil && l.Scope.Model.Name != "" {
			return "Week · " + l.Scope.Model.Name
		}
	}
	return ""
}

// claudeAccount fills acc from Claude's usage answer: the limits it lists,
// as Claude Code shows them, else (an older answer) its known limits in
// order. Other entries carry internal names, and aren't shown.
func claudeAccount(acc proto.CreditAccount, answer map[string]json.RawMessage) proto.CreditAccount {
	var listed []claudeListed
	if json.Unmarshal(answer["limits"], &listed) == nil {
		for _, l := range listed {
			label := claudeListedLabel(l)
			if label == "" {
				continue
			}
			w := proto.CreditWindow{Label: label, Used: l.Percent}
			if t, err := time.Parse(time.RFC3339Nano, l.ResetsAt); err == nil {
				w.ResetsAt = t
			}
			acc.Windows = append(acc.Windows, w)
		}
	}
	if len(acc.Windows) > 0 {
		return claudeExtra(acc, answer)
	}
	add := func(label string, raw json.RawMessage) {
		var l claudeLimit
		if json.Unmarshal(raw, &l) != nil || l.Utilization == nil {
			return
		}
		w := proto.CreditWindow{Label: label, Used: *l.Utilization}
		if t, err := time.Parse(time.RFC3339Nano, l.ResetsAt); err == nil {
			w.ResetsAt = t
		}
		acc.Windows = append(acc.Windows, w)
	}
	for _, k := range claudeLabels {
		if raw, ok := answer[k.key]; ok {
			add(k.label, raw)
		}
	}
	return claudeExtra(acc, answer)
}

// claudeExtra notes the extra usage beyond the plan, when it is on.
func claudeExtra(acc proto.CreditAccount, answer map[string]json.RawMessage) proto.CreditAccount {
	var extra struct {
		Enabled     bool     `json:"is_enabled"`
		Utilization *float64 `json:"utilization"`
	}
	if raw, ok := answer["extra_usage"]; ok && json.Unmarshal(raw, &extra) == nil && extra.Enabled {
		acc.Note = "Extra usage on"
		if extra.Utilization != nil {
			acc.Note = fmt.Sprintf("Extra usage: %.0f%% of the monthly cap", *extra.Utilization)
		}
	}
	return acc
}

// claudeEmail is the account the Claude Code configuration at path says
// it is signed in as; with a CLAUDE_CONFIG_DIR set (configDir) it is that
// folder's.
func claudeEmail(path, configDir string) string {
	if configDir != "" {
		path = filepath.Join(configDir, ".claude.json")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var cfg struct {
		OAuthAccount struct {
			Email string `json:"emailAddress"`
			Org   string `json:"organizationName"`
		} `json:"oauthAccount"`
	}
	_ = json.Unmarshal(data, &cfg)
	return cfg.OAuthAccount.Email
}
