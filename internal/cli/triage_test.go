package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChiragAgg5k/mailslot/internal/mail"
)

const (
	triageReadOnlyVariable = "MAILSLOT_READ_ONLY"
	triageDryRunNotice     = "Dry run; nothing changed."
)

// triageChange mirrors one row of a triage command's JSON output.
type triageChange struct {
	ID        int    `json:"id"`
	MessageID string `json:"messageId"`
	Mailbox   string `json:"mailbox"`
	Change    string `json:"change"`
	NewID     *int   `json:"newId"`
	DryRun    bool   `json:"dryRun"`
}

// triageCommands are one valid invocation of each triage command against the Work account.
var triageCommands = map[string][]string{
	"mark":    {"mark", "1003", "--read"},
	"flag":    {"flag", "1001", "--color", "green"},
	"move":    {"move", "1001", "--to", "Projects"},
	"archive": {"archive", "1001"},
	"delete":  {"delete", "1001"},
}

func triageRun(t *testing.T, options setup, arguments ...string) []triageChange {
	t.Helper()
	var rows []triageChange
	executeWith(t, options, append(arguments, "--json")...).succeeded(t).decode(t, &rows)
	return rows
}

func triageRead(t *testing.T, options setup, arguments ...string) mail.Message {
	t.Helper()
	var message mail.Message
	executeWith(t, options, append([]string{"read"}, append(arguments, "--json")...)...).succeeded(t).decode(t, &message)
	return message
}

func triageExpectCode(t *testing.T, outcome result, code int) {
	t.Helper()
	if outcome.Code != code {
		t.Fatalf("exit code %d, want %d\nstdout:\n%s\nstderr:\n%s", outcome.Code, code, outcome.Stdout, outcome.Stderr)
	}
}

func triageExpectMove(t *testing.T, rows []triageChange, mailbox string, ids ...int) {
	t.Helper()
	if len(rows) != len(ids) {
		t.Fatalf("rows %+v, want %d", rows, len(ids))
	}
	for index, row := range rows {
		if row.ID != ids[index] || row.Change != "moved to "+mailbox || row.DryRun {
			t.Errorf("row %+v, want id %d moved to %s", row, ids[index], mailbox)
		}
		if row.NewID == nil || *row.NewID == 0 || *row.NewID == row.ID {
			t.Errorf("row %+v lacks a new id", row)
		}
		if row.MessageID == "" {
			t.Errorf("row %+v lacks its Message-ID", row)
		}
	}
}

func TestMarkReadAndUnread(t *testing.T) {
	rows := triageRun(t, setup{}, "-a", "Work", "mark", "1003", "--read")
	if len(rows) != 1 || rows[0].ID != 1003 || rows[0].Change != "marked read" || rows[0].NewID != nil || rows[0].DryRun {
		t.Fatalf("rows %+v", rows)
	}
	if !triageRead(t, setup{}, "-a", "Work", "1003").Read {
		t.Fatal("1003 is still unread")
	}
	outcome := execute(t, "-a", "Work", "mark", "1003", "--unread").succeeded(t)
	if !strings.Contains(outcome.Stdout, "marked unread") || !strings.Contains(outcome.Stdout, "quarterly-3@mail.example.com") {
		t.Fatalf("table lacks the change:\n%s", outcome.Stdout)
	}
	if triageRead(t, setup{}, "-a", "Work", "1003").Read {
		t.Fatal("1003 is still read")
	}
}

func TestMarkSeveralReferences(t *testing.T) {
	rows := triageRun(t, setup{}, "-a", "Work", "mark", "1009", "newsletter-1010@news.example.org", "--read")
	if len(rows) != 2 || rows[0].ID != 1009 || rows[1].ID != 1010 {
		t.Fatalf("rows %+v", rows)
	}
	var unread []mail.Summary
	execute(t, "-a", "Work", "list", "--unread", "--json").succeeded(t).decode(t, &unread)
	for _, summary := range unread {
		if summary.ID == 1009 || summary.ID == 1010 {
			t.Errorf("%d is still unread", summary.ID)
		}
	}
}

func TestMarkLocal(t *testing.T) {
	options := setup{Data: localData}
	rows := triageRun(t, options, "-a", "local", "mark", "2001", "-m", "Receipts", "--unread")
	if len(rows) != 1 || rows[0].Change != "marked unread" {
		t.Fatalf("rows %+v", rows)
	}
	if triageRead(t, options, "-a", "local", "-m", "Receipts", "2001").Read {
		t.Fatal("2001 is still read")
	}
}

func TestMarkNeedsExactlyOneState(t *testing.T) {
	triageExpectCode(t, execute(t, "-a", "Work", "mark", "1003"), 2)
	triageExpectCode(t, execute(t, "-a", "Work", "mark", "1003", "--read", "--unread"), 2)
}

func TestFlagColorAndClear(t *testing.T) {
	rows := triageRun(t, setup{}, "-a", "Work", "flag", "1001", "1003", "--color", "Green")
	if len(rows) != 2 || rows[0].Change != "flagged green" || rows[1].Change != "flagged green" {
		t.Fatalf("rows %+v", rows)
	}
	if !triageRead(t, setup{}, "-a", "Work", "1001").Flagged {
		t.Fatal("1001 is not flagged")
	}
	rows = triageRun(t, setup{}, "-a", "Work", "flag", "1005", "--clear")
	if len(rows) != 1 || rows[0].Change != "flag cleared" {
		t.Fatalf("rows %+v", rows)
	}
	if triageRead(t, setup{}, "-a", "Work", "1005").Flagged {
		t.Fatal("1005 is still flagged")
	}
}

func TestFlagLocal(t *testing.T) {
	rows := triageRun(t, setup{}, "-a", "local", "flag", "1022", "-m", "Receipts", "--color", "red")
	if len(rows) != 1 || rows[0].Change != "flagged red" {
		t.Fatalf("rows %+v", rows)
	}
	if !triageRead(t, setup{}, "-a", "local", "-m", "Receipts", "1022").Flagged {
		t.Fatal("1022 is not flagged")
	}
}

func TestFlagNeedsColorOrClear(t *testing.T) {
	triageExpectCode(t, execute(t, "-a", "Work", "flag", "1001", "--color", "pink"), 2)
	triageExpectCode(t, execute(t, "-a", "Work", "flag", "1001"), 2)
	triageExpectCode(t, execute(t, "-a", "Work", "flag", "1001", "--color", "red", "--clear"), 2)
	if triageRead(t, setup{}, "-a", "Work", "1001").Flagged {
		t.Fatal("a refused flag changed 1001")
	}
}

func TestMoveReportsNewID(t *testing.T) {
	rows := triageRun(t, setup{}, "-a", "Work", "move", "1015", "-m", "Projects", "--to", "Archive")
	triageExpectMove(t, rows, "Archive", 1015)
	outcome := execute(t, "-a", "Work", "read", "1015", "-m", "Projects")
	triageExpectCode(t, outcome, 4)
	if !strings.Contains(outcome.Stderr, "Message-ID") {
		t.Fatalf("stale id error lacks the retry hint:\n%s", outcome.Stderr)
	}
	moved := triageRead(t, setup{}, "-a", "Work", "migration-1015@mail.example.com")
	if moved.Mailbox != "Archive" || moved.ID != *rows[0].NewID {
		t.Fatalf("moved message %d in %s, want %d in Archive", moved.ID, moved.Mailbox, *rows[0].NewID)
	}
	table := execute(t, "-a", "Work", "move", "migration-1015@mail.example.com", "--to", "Projects").succeeded(t)
	if !strings.Contains(table.Stdout, "NEW ID") || !strings.Contains(table.Stdout, "moved to Projects") {
		t.Fatalf("table lacks the move:\n%s", table.Stdout)
	}
}

func TestMoveSeveralReferences(t *testing.T) {
	rows := triageRun(t, setup{}, "-a", "Work", "move", "1004", "1005", "1006", "--to", "projects")
	triageExpectMove(t, rows, "projects", 1004, 1005, 1006)
	var listed []mail.Summary
	execute(t, "-a", "Work", "list", "-m", "Projects", "--json").succeeded(t).decode(t, &listed)
	if len(listed) != 4 {
		t.Fatalf("Projects holds %d messages, want 4", len(listed))
	}
}

func TestMoveLocal(t *testing.T) {
	options := setup{Data: localData}
	rows := triageRun(t, options, "-a", "local", "move", "2001", "-m", "Receipts", "--to", "Archive")
	triageExpectMove(t, rows, "Archive", 2001)
	if moved := triageRead(t, options, "-a", "local", "local-2001@shop.example.com"); moved.Mailbox != "Archive" {
		t.Fatalf("2001 is in %s", moved.Mailbox)
	}
}

func TestMoveRefusals(t *testing.T) {
	triageExpectCode(t, execute(t, "-a", "Work", "move", "1001"), 2)
	triageExpectCode(t, execute(t, "-a", "Work", "move", "1001", "--to", "inbox"), 2)
	triageExpectCode(t, execute(t, "-a", "Work", "move", "quarterly-1@mail.example.com", "--to", "Inbox"), 2)
	triageExpectCode(t, execute(t, "-a", "Work", "move", "1001", "--to", "Nowhere"), 4)
	if moved := triageRead(t, setup{}, "-a", "Work", "quarterly-1@mail.example.com"); moved.Mailbox != mail.Inbox || moved.ID != 1001 {
		t.Fatalf("a refused move changed 1001: %d in %s", moved.ID, moved.Mailbox)
	}
}

func TestArchiveWork(t *testing.T) {
	triageExpectMove(t, triageRun(t, setup{}, "-a", "Work", "archive", "1001", "1003"), "Archive", 1001, 1003)
	triageExpectCode(t, execute(t, "-a", "Work", "archive", "1011", "-m", "Archive"), 2)
}

func TestArchivePersonalUsesAllMail(t *testing.T) {
	triageExpectMove(t, triageRun(t, setup{}, "-a", "Personal", "archive", "1017"), "All Mail", 1017)
	if moved := triageRead(t, setup{}, "-a", "Personal", "book-club-1017@mail.example.org"); moved.Mailbox != "All Mail" {
		t.Fatalf("1017 is in %s", moved.Mailbox)
	}
}

func TestArchiveLocal(t *testing.T) {
	triageExpectMove(t, triageRun(t, setup{}, "-a", "local", "archive", "1022", "-m", "Receipts"), "Archive", 1022)
}

func TestArchiveWithoutArchiveMailbox(t *testing.T) {
	outcome := execute(t, "-a", "Old", "archive", "1021")
	triageExpectCode(t, outcome, 4)
	if !strings.Contains(outcome.Stderr, "mailslot move --to <mailbox>") {
		t.Fatalf("missing archive error lacks the hint:\n%s", outcome.Stderr)
	}
}

func TestDeleteWorkLandsInTrash(t *testing.T) {
	triageExpectMove(t, triageRun(t, setup{}, "-a", "Work", "delete", "1001", "standup-1009@mail.example.com"), "Trash", 1001, 1009)
	if moved := triageRead(t, setup{}, "-a", "Work", "standup-1009@mail.example.com"); moved.Mailbox != "Trash" {
		t.Fatalf("1009 is in %s", moved.Mailbox)
	}
}

func TestDeletePersonalLandsInDeletedItems(t *testing.T) {
	triageExpectMove(t, triageRun(t, setup{}, "-a", "Personal", "delete", "1020"), "Deleted Items", 1020)
}

func TestDeleteLocal(t *testing.T) {
	triageExpectMove(t, triageRun(t, setup{Data: localData}, "-a", "local", "delete", "2002", "-m", "Import/fixture"), "Trash", 2002)
}

func TestDeleteRefusesTrash(t *testing.T) {
	for _, arguments := range [][]string{
		{"-a", "Work", "delete", "1014", "-m", "Trash"},
		{"-a", "Work", "delete", "promo-1014@offers.example.net"},
		{"-a", "Work", "delete", "promo-1014@offers.example.net", "--dry-run"},
		{"-a", "Work", "delete", "1001", "promo-1014@offers.example.net"},
	} {
		outcome := execute(t, arguments...)
		triageExpectCode(t, outcome, 2)
		if !strings.Contains(outcome.Stderr, "never deletes permanently") {
			t.Fatalf("refusal lacks its reason:\n%s", outcome.Stderr)
		}
		if strings.Contains(outcome.Stdout, "moved to") {
			t.Fatalf("refusal still reported a move:\n%s", outcome.Stdout)
		}
	}
	if kept := triageRead(t, setup{}, "-a", "Work", "-m", "Trash", "1014"); kept.ID != 1014 {
		t.Fatalf("1014 changed: %+v", kept)
	}
	if first := triageRead(t, setup{}, "-a", "Work", "quarterly-1@mail.example.com"); first.Mailbox != mail.Inbox || first.ID != 1001 {
		t.Fatalf("a refused delete moved 1001: %d in %s", first.ID, first.Mailbox)
	}
}

func TestMoveCollapsesDuplicateReferences(t *testing.T) {
	cases := map[string][]string{
		"numeric and Message-ID": {"1001", "quarterly-1@mail.example.com"},
		"lowercase mailbox":      {"1001", "quarterly-1@mail.example.com", "-m", "inbox"},
	}
	for name, references := range cases {
		t.Run(name, func(t *testing.T) {
			rows := triageRun(t, setup{}, append([]string{"-a", "Work", "move"}, append(references, "--to", "Projects")...)...)
			triageExpectMove(t, rows, "Projects", 1001)
			if moved := triageRead(t, setup{}, "-a", "Work", "quarterly-1@mail.example.com"); moved.Mailbox != "Projects" {
				t.Fatalf("1001 is in %s", moved.Mailbox)
			}
		})
	}
}

func TestTriageNeedsReference(t *testing.T) {
	for name, arguments := range triageCommands {
		outcome := execute(t, append([]string{"-a", "Work", name}, arguments[2:]...)...)
		triageExpectCode(t, outcome, 2)
		if !strings.Contains(outcome.Stderr, "requires at least 1 arg") {
			t.Errorf("%s without a reference: %s", name, outcome.Stderr)
		}
	}
}

func TestTriageDryRunChangesNothing(t *testing.T) {
	for name, arguments := range triageCommands {
		var rows []triageChange
		execute(t, append(append([]string{"-a", "Work", "--json"}, arguments...), "--dry-run")...).succeeded(t).decode(t, &rows)
		if len(rows) != 1 || !rows[0].DryRun || rows[0].NewID != nil {
			t.Errorf("%s dry run rows %+v", name, rows)
		}
		table := execute(t, append(append([]string{"-a", "Work"}, arguments...), "--dry-run")...).succeeded(t)
		if !strings.HasPrefix(table.Stdout, triageDryRunNotice) {
			t.Errorf("%s dry run table lacks the notice:\n%s", name, table.Stdout)
		}
	}
	first := triageRead(t, setup{}, "-a", "Work", "1001")
	if first.Mailbox != mail.Inbox || first.Flagged {
		t.Fatalf("dry runs changed 1001: %+v", first)
	}
	if triageRead(t, setup{}, "-a", "Work", "1003").Read {
		t.Fatal("dry run marked 1003 read")
	}
}

func TestTriageReadOnlyFromEnvironment(t *testing.T) {
	triageExpectReadOnly(t, setup{Environment: map[string]string{triageReadOnlyVariable: "1"}})
}

func TestTriageReadOnlyFromConfig(t *testing.T) {
	prepare(t, setup{})
	path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "mailslot", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("read_only = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	triageExpectReadOnly(t, setup{})
}

func triageExpectReadOnly(t *testing.T, options setup) {
	t.Helper()
	for name, arguments := range triageCommands {
		outcome := executeWith(t, options, append([]string{"-a", "Work"}, arguments...)...)
		if outcome.Code != 6 {
			t.Errorf("%s exit code %d, want 6\n%s", name, outcome.Code, outcome.Stderr)
		}
		executeWith(t, options, append(append([]string{"-a", "Work"}, arguments...), "--dry-run")...).succeeded(t)
	}
	if first := triageRead(t, options, "-a", "Work", "1001"); first.Mailbox != mail.Inbox || first.Flagged {
		t.Fatalf("read-only commands changed 1001: %+v", first)
	}
}

func TestTriagePersistsAcrossCalls(t *testing.T) {
	execute(t, "-a", "Work", "flag", "1009", "--color", "blue").succeeded(t)
	execute(t, "-a", "Work", "mark", "1009", "--read").succeeded(t)
	rows := triageRun(t, setup{}, "-a", "Work", "archive", "1009")
	triageExpectMove(t, rows, "Archive", 1009)
	archived := triageRead(t, setup{}, "-a", "Work", "standup-1009@mail.example.com")
	if archived.Mailbox != "Archive" || !archived.Read || !archived.Flagged || archived.ID != *rows[0].NewID {
		t.Fatalf("changes did not persist: %+v", archived)
	}
}
