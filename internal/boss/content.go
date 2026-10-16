package boss

import (
	"strings"
)

// Entry is one message in the fake conversation.
type Entry struct {
	Role string // user, assistant, tool, output, info
	Text string
}

func initialConversation() []Entry {
	return []Entry{
		{Role: "user", Text: "refactor parseConfig to wrap errors and add a test"},
		{Role: "assistant", Text: "I'll look at the config parsing first, then wrap the errors."},
		{Role: "tool", Text: "Read  internal/config/config.go"},
		{Role: "output", Text: "read 84 lines"},
		{Role: "tool", Text: "Edit  internal/config/config.go"},
		{Role: "output", Text: "Applied 2 edits  +6 -1"},
		{Role: "assistant", Text: "Errors are now wrapped with %w, and I added a test covering a corrupt config file."},
		{Role: "tool", Text: "Bash  go test ./internal/config/"},
		{Role: "output", Text: "ok   github.com/acme/api/internal/config   0.021s"},
	}
}

// liveScript is appended one entry at a time while the boss screen is shown, so
// the session looks like it keeps making progress.
func liveScript() []Entry {
	return []Entry{
		{Role: "assistant", Text: "Config loader is wired behind a --config flag now."},
		{Role: "tool", Text: "Read  cmd/api/main.go"},
		{Role: "output", Text: "read 132 lines"},
		{Role: "tool", Text: "Edit  cmd/api/main.go"},
		{Role: "output", Text: "Applied 3 edits  +18 -4"},
		{Role: "tool", Text: "Bash  go build ./... && go vet ./..."},
		{Role: "output", Text: "(no output — success)"},
		{Role: "assistant", Text: "Build and vet are clean. Running the full suite."},
		{Role: "tool", Text: "Bash  go test ./... -count=1"},
		{Role: "output", Text: "ok   github.com/acme/api/internal/config   0.017s\nok   github.com/acme/api/internal/store    0.041s\nok   github.com/acme/api/cmd/api            0.009s"},
		{Role: "assistant", Text: "All green — 58 tests across 3 packages."},
		{Role: "assistant", Text: "Next I'll add a regression test for the empty-config path."},
		{Role: "tool", Text: "Grep  \"func Test\" -n internal/config"},
		{Role: "output", Text: "internal/config/config_test.go:8\ninternal/config/config_test.go:19"},
		{Role: "assistant", Text: "Found the existing cases; appending one more for the empty file."},
	}
}

// replyFor produces a plausible reply for a user message. It is deterministic
// (keyword based) and never touches the network.
func replyFor(input string) []Entry {
	q := strings.ToLower(input)
	switch {
	case containsAny(q, "test", "spec"):
		return []Entry{
			{Role: "tool", Text: "Bash  go test ./... -count=1"},
			{Role: "output", Text: "ok   github.com/acme/api/internal/config   0.018s\nok   github.com/acme/api/internal/store    0.044s"},
			{Role: "assistant", Text: "All passing — 41 cases across both packages."},
		}
	case containsAny(q, "read", "file", "open", "show"):
		return []Entry{
			{Role: "tool", Text: "Read  internal/store/progress.go"},
			{Role: "output", Text: "read 122 lines"},
			{Role: "assistant", Text: "It stores progress as JSON keyed by a hash of the book path, written atomically via a temp file and rename."},
		}
	case containsAny(q, "fix", "bug", "error", "broken", "fail"):
		return []Entry{
			{Role: "tool", Text: "Edit  internal/store/progress.go"},
			{Role: "output", Text: "Applied 1 edit  +4 -2"},
			{Role: "assistant", Text: "Fixed: reading a corrupt JSON file no longer returns an error, it falls back to an empty store. Want the diff?"},
		}
	case containsAny(q, "run", "build", "compile"):
		return []Entry{
			{Role: "tool", Text: "Bash  go build ./..."},
			{Role: "output", Text: "(no output — success)"},
			{Role: "assistant", Text: "Builds clean with no warnings."},
		}
	default:
		return []Entry{
			{Role: "assistant", Text: "On it. Let me look at the relevant code first, then make the smallest change that works."},
			{Role: "tool", Text: "Grep  \"func.*Update\" -n internal"},
			{Role: "output", Text: "internal/app/app.go:58\ninternal/reader/reader.go:41"},
			{Role: "assistant", Text: "Two Update methods; I'll trace how state flows before editing."},
		}
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
