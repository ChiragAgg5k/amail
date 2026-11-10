package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const mergeRecipients = "email,name\nrobin@example.com,Robin\nalex@example.com,Alex\n"

// mergeReport is the JSON form of one merge row.
type mergeReport struct {
	Row     int      `json:"row"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
}

func mergeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "people.csv")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write CSV: %v", err)
	}
	return path
}

func mergeDrafts(t *testing.T) []mail.Summary {
	t.Helper()
	var summaries []mail.Summary
	execute(t, "list", "-a", "Work", "-m", "Drafts", "--json").succeeded(t).decode(t, &summaries)
	return summaries
}

func mergeArguments(path string, extra ...string) []string {
	return append([]string{"merge", path, "-a", "Work", "--to", "{{email}}", "--subject", "Hello {{ name }}", "--body", "Hi {{name}},\nsee you soon.\n"}, extra...)
}

func TestMergeDryRun(t *testing.T) {
	path := mergeFile(t, mergeRecipients)
	before := len(mergeDrafts(t))
	outcome := execute(t, mergeArguments(path, "--dry-run")...).succeeded(t)
	for _, want := range []string{"Dry run; nothing changed.", "ROW", "robin@example.com", "Hello Robin", "alex@example.com", "Hello Alex"} {
		if !strings.Contains(outcome.Stdout, want) {
			t.Errorf("dry run output lacks %q:\n%s", want, outcome.Stdout)
		}
	}
	if after := len(mergeDrafts(t)); after != before {
		t.Fatalf("drafts %d after dry run, want %d", after, before)
	}
}

func TestMergeCreatesDrafts(t *testing.T) {
	path := mergeFile(t, mergeRecipients)
	before := mergeDrafts(t)
	outcome := execute(t, mergeArguments(path)...).succeeded(t)
	if !strings.Contains(outcome.Stdout, "Opened 2 drafts in Mail; nothing was sent.") {
		t.Fatalf("merge output lacks the summary:\n%s", outcome.Stdout)
	}
	after := mergeDrafts(t)
	if len(after) != len(before)+2 {
		t.Fatalf("drafts %d, want %d", len(after), len(before)+2)
	}
	subjects := map[string]bool{}
	for _, summary := range after {
		subjects[summary.Subject] = true
	}
	for _, want := range []string{"Hello Robin", "Hello Alex"} {
		if !subjects[want] {
			t.Errorf("no draft with subject %q", want)
		}
	}
}

func TestMergeUnknownPlaceholder(t *testing.T) {
	path := mergeFile(t, mergeRecipients)
	before := len(mergeDrafts(t))
	outcome := execute(t, "merge", path, "-a", "Work", "--to", "{{email}}", "--subject", "Hi {{ nickname }}", "--body", "Hello")
	if outcome.Code != 2 {
		t.Fatalf("exit code %d, want 2\n%s", outcome.Code, outcome.Stderr)
	}
	if !strings.Contains(outcome.Stderr, "nickname") {
		t.Errorf("error does not name the column:\n%s", outcome.Stderr)
	}
	if after := len(mergeDrafts(t)); after != before {
		t.Fatalf("drafts %d after a failed merge, want %d", after, before)
	}
}

func TestMergeRejectsBadInput(t *testing.T) {
	cases := map[string][]string{
		"empty CSV":    mergeArguments(mergeFile(t, "")),
		"header only":  mergeArguments(mergeFile(t, "email,name\n")),
		"empty To":     mergeArguments(mergeFile(t, "email,name\nrobin@example.com,Robin\n,Alex\n")),
		"foreign From": mergeArguments(mergeFile(t, mergeRecipients), "--from", "robin@example.com"),
		"no body":      {"merge", mergeFile(t, mergeRecipients), "-a", "Work", "--to", "{{email}}", "--subject", "Hi"},
		"send flag":    mergeArguments(mergeFile(t, mergeRecipients), "--send"),
	}
	before := len(mergeDrafts(t))
	for name, arguments := range cases {
		if outcome := execute(t, arguments...); outcome.Code != 2 {
			t.Errorf("%s: exit code %d, want 2\n%s", name, outcome.Code, outcome.Stderr)
		}
	}
	if after := len(mergeDrafts(t)); after != before {
		t.Fatalf("drafts %d after rejected merges, want %d", after, before)
	}
}

func TestMergeFrom(t *testing.T) {
	path := mergeFile(t, mergeRecipients)
	execute(t, mergeArguments(path, "--from", "sam@example.com", "--signature", "Short")...).succeeded(t)
}

func TestMergeReadOnly(t *testing.T) {
	path := mergeFile(t, mergeRecipients)
	readOnly := setup{Environment: map[string]string{"MAILSLOT_READ_ONLY": "1"}}
	if outcome := executeWith(t, readOnly, mergeArguments(path)...); outcome.Code != 6 {
		t.Fatalf("exit code %d, want 6\n%s", outcome.Code, outcome.Stderr)
	}
	executeWith(t, readOnly, mergeArguments(path, "--dry-run")...).succeeded(t)
}

func TestMergeJSON(t *testing.T) {
	path := mergeFile(t, mergeRecipients)
	var preview []mergeReport
	execute(t, mergeArguments(path, "--dry-run", "--json")...).succeeded(t).decode(t, &preview)
	if len(preview) != 2 {
		t.Fatalf("rows %d, want 2", len(preview))
	}
	first := preview[0]
	if first.Row != 1 || len(first.To) != 1 || first.To[0] != "robin@example.com" || first.Subject != "Hello Robin" || first.Body != "Hi Robin,\nsee you soon.\n" {
		t.Fatalf("first row %+v", first)
	}

	var created []mergeReport
	execute(t, mergeArguments(path, "--json")...).succeeded(t).decode(t, &created)
	if len(created) != 2 || created[1].Row != 2 || created[1].Subject != "Hello Alex" || created[1].Body != "" {
		t.Fatalf("created rows %+v", created)
	}
}
