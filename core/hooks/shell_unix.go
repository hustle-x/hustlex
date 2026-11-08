//go:build !windows

package hooks

func shell(command string) (string, []string) { return "sh", []string{"-c", command} }
