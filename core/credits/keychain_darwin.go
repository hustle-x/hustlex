package credits

import (
	"bytes"
	"context"
	"os/exec"
)

// claudeKeychain reads the Claude Code logins kept in the macOS keychain
// under these item names. Reading an item is what macOS may ask the user
// to allow, so only the items wanted are read.
func claudeKeychain(ctx context.Context, services []string) []keychainItem {
	var out []keychainItem
	for _, name := range services {
		item, err := exec.CommandContext(ctx, "security", "find-generic-password", "-s", name, "-w").Output()
		if err == nil {
			out = append(out, keychainItem{name, bytes.TrimSpace(item)})
		}
	}
	return out
}
