package merge_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/ChiragAgg5k/mailslot/internal/merge"
)

func rows(t *testing.T, text string) []merge.Row {
	t.Helper()
	read, _, err := merge.Read(strings.NewReader(text))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return read
}

func TestTemplateFill(t *testing.T) {
	row := rows(t, "name,email\nRobin,robin@example.com\n")[0]
	cases := map[string]string{
		"Hello {{name}}":                      "Hello Robin",
		"Hello {{ name }} <{{email}}>":        "Hello Robin <robin@example.com>",
		"no placeholders":                     "no placeholders",
		"{{name}}{{name}}":                    "RobinRobin",
		"open {{ name without close":          "open {{ name without close",
		"":                                    "",
		"before {{name}} and } after {{ only": "before Robin and } after {{ only",
	}
	for text, want := range cases {
		got, err := merge.Parse(text).Fill(row)
		if err != nil {
			t.Errorf("Fill(%q): %v", text, err)
			continue
		}
		if got != want {
			t.Errorf("Fill(%q) = %q, want %q", text, got, want)
		}
	}
}

func TestTemplateFillUnknownColumn(t *testing.T) {
	row := rows(t, "name\nRobin\n")[0]
	_, err := merge.Parse("Hi {{ nickname }}").Fill(row)
	if err == nil || !strings.Contains(err.Error(), "nickname") {
		t.Fatalf("error %v, want one naming nickname", err)
	}
}

func TestTemplateFields(t *testing.T) {
	got := merge.Parse("{{ b }} {{a}} {{b}} text").Fields()
	if want := []string{"b", "a"}; !slices.Equal(got, want) {
		t.Fatalf("fields %v, want %v", got, want)
	}
	if got := merge.Parse("plain").Fields(); len(got) != 0 {
		t.Fatalf("fields %v, want none", got)
	}
}

func TestReadHeader(t *testing.T) {
	read, header, err := merge.Read(strings.NewReader("\ufeff email , name\nrobin@example.com,Robin\nalex@example.com,\"Alex, Jr\"\n"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if want := []string{"email", "name"}; !slices.Equal(header, want) {
		t.Fatalf("header %v, want %v", header, want)
	}
	if len(read) != 2 {
		t.Fatalf("rows %d, want 2", len(read))
	}
	if value, found := read[1].Value("name"); !found || value != "Alex, Jr" {
		t.Fatalf("name %q %v, want quoted value", value, found)
	}
	if _, found := read[0].Value("missing"); found {
		t.Fatal("found a column the header lacks")
	}
}

func TestReadHeaderOnly(t *testing.T) {
	read, header, err := merge.Read(strings.NewReader("email\n"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(read) != 0 || len(header) != 1 {
		t.Fatalf("rows %d header %v, want no rows and one column", len(read), header)
	}
}

func TestReadRejects(t *testing.T) {
	cases := map[string]string{
		"empty":     "",
		"ragged":    "a,b\n1\n",
		"duplicate": "a, a\n1,2\n",
		"unnamed":   "a,,c\n1,2,3\n",
	}
	for name, text := range cases {
		if _, _, err := merge.Read(strings.NewReader(text)); err == nil {
			t.Errorf("%s: read succeeded, want an error", name)
		}
	}
}
