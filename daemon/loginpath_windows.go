package daemon

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// useLoginPath adds what the user's PATH has gained since the daemon
// started: agent installers add their folder to the user's PATH in the
// registry, which a running process doesn't see.
func useLoginPath(log *slog.Logger) { refreshLoginPath(log) }

// refreshLoginPath reads the machine's and the user's PATH from the
// registry again, adding entries missing from the daemon's own, and
// %USERPROFILE%\.local\bin, where Claude Code installs.
func refreshLoginPath(*slog.Logger) {
	have := filepath.SplitList(os.Getenv("PATH"))
	seen := map[string]bool{}
	for _, p := range have {
		seen[strings.ToLower(filepath.Clean(p))] = true
	}
	var add []string
	extra := []string{
		registryPath(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`),
		registryPath(registry.CURRENT_USER, `Environment`),
	}
	if home, err := os.UserHomeDir(); err == nil {
		extra = append(extra, filepath.Join(home, ".local", "bin"))
	}
	for _, list := range extra {
		for _, p := range filepath.SplitList(list) {
			p = strings.TrimSpace(os.ExpandEnv(percentToDollar(p)))
			if p == "" {
				continue
			}
			if k := strings.ToLower(filepath.Clean(p)); !seen[k] {
				seen[k] = true
				add = append(add, p)
			}
		}
	}
	if len(add) > 0 {
		_ = os.Setenv("PATH", strings.Join(append(have, add...), string(os.PathListSeparator)))
	}
}

// registryPath is the Path value of key, "" when unreadable.
func registryPath(root registry.Key, key string) string {
	k, err := registry.OpenKey(root, key, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	v, _, err := k.GetStringValue("Path")
	if err != nil {
		return ""
	}
	return v
}

// percentToDollar turns %NAME% into ${NAME}, for os.ExpandEnv.
func percentToDollar(s string) string {
	var b strings.Builder
	for {
		i := strings.Index(s, "%")
		if i < 0 {
			break
		}
		j := strings.Index(s[i+1:], "%")
		if j < 0 {
			break
		}
		b.WriteString(s[:i] + "${" + s[i+1:i+1+j] + "}")
		s = s[i+2+j:]
	}
	b.WriteString(s)
	return b.String()
}
