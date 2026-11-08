package hooks

func shell(command string) (string, []string) { return "cmd", []string{"/C", command} }
