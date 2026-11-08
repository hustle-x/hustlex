package transcript

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTokenCounterCountsAnswersOnceAndOnlyWhatIsNew(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.jsonl")
	write := func(s string, appendTo bool) {
		flag := os.O_CREATE | os.O_WRONLY
		if appendTo {
			flag |= os.O_APPEND
		}
		f, err := os.OpenFile(path, flag, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		f.WriteString(s)
		f.Close()
	}
	line := func(id string, in, out int) string {
		return `{"type":"assistant","message":{"id":"` + id + `","model":"m","usage":{"input_tokens":` +
			itoa(in) + `,"output_tokens":` + itoa(out) + `}}}` + "\n"
	}
	write(line("a", 10, 5)+line("a", 10, 5)+`{"type":"user"}`+"\n", false)
	var c TokenCounter
	got, _ := c.Count(path)
	if got["m"].Input != 10 || got["m"].Output != 5 {
		t.Fatalf("a streamed answer counted twice: %+v", got["m"])
	}
	write(line("b", 1, 2), true)
	got, _ = c.Count(path)
	if got["m"].Input != 11 || got["m"].Output != 7 {
		t.Fatalf("appended answer missing: %+v", got["m"])
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for ; n > 0; n /= 10 {
		s = string(rune('0'+n%10)) + s
	}
	return s
}
