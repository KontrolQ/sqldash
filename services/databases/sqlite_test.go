package databases

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestOnlyARealSQLiteHeaderIsAccepted(t *testing.T) {
	cases := []struct {
		what    string
		content string
		wanted  bool
	}{
		{"a real header", SQLiteMagic + "the rest of the file", true},
		{"a dump", "PRAGMA foreign_keys=OFF;\nBEGIN TRANSACTION;\n", false},
		{"an empty file", "", false},
		{"a truncated header", "SQLite for", false},
		{"the header without its terminator", "SQLite format 3 ", false},
	}

	for _, held := range cases {
		reader := bytes.NewReader([]byte(held.content))

		looks, checkError := looksLikeSQLite(reader)
		if checkError != nil {
			t.Fatalf("%s: %v", held.what, checkError)
		}

		if looks != held.wanted {
			t.Errorf("%s: read %v, wanted %v", held.what, looks, held.wanted)
		}

		at, seekError := reader.Seek(0, 1)
		if seekError != nil {
			t.Fatalf("%s: %v", held.what, seekError)
		}

		if at != 0 {
			t.Errorf("%s: left the reader at %d rather than rewound", held.what, at)
		}
	}
}

func TestStatementsSplitOnUnquotedSemicolonsOnly(t *testing.T) {
	dump := strings.Join([]string{
		"INSERT INTO a VALUES ('one; two')",
		`INSERT INTO b VALUES ("three; four")`,
		"INSERT INTO c VALUES ('it''s here')",
		"SELECT 1",
	}, ";")

	reader := bufio.NewReader(strings.NewReader(dump))
	found := make([]string, 0, 4)

	for {
		statement, more, readError := nextStatement(reader)
		if readError != nil {
			t.Fatalf("split failed: %v", readError)
		}

		found = append(found, strings.TrimSpace(statement))

		if !more {
			break
		}
	}

	if len(found) != 4 {
		t.Fatalf("split into %d statements, wanted 4: %q", len(found), found)
	}

	if found[0] != "INSERT INTO a VALUES ('one; two')" {
		t.Errorf("a semicolon inside single quotes split the statement: %q", found[0])
	}

	if found[1] != `INSERT INTO b VALUES ("three; four")` {
		t.Errorf("a semicolon inside double quotes split the statement: %q", found[1])
	}

	if found[2] != "INSERT INTO c VALUES ('it''s here')" {
		t.Errorf("an escaped quote broke the split: %q", found[2])
	}
}

func TestLeadingCommentsAreStrippedButStatementsSurvive(t *testing.T) {
	cases := []struct {
		given  string
		wanted string
	}{
		{"-- a note\nSELECT 1", "SELECT 1"},
		{"-- one\n-- two\nSELECT 2", "SELECT 2"},
		{"SELECT 'a -- b'", "SELECT 'a -- b'"},
		{"-- only a comment", ""},
		{"   ", ""},
	}

	for _, held := range cases {
		found := meaningful(held.given)

		if found != held.wanted {
			t.Errorf("meaningful(%q) gave %q, wanted %q", held.given, found, held.wanted)
		}
	}
}

func TestTransactionControlIsNotReplayed(t *testing.T) {
	skipped := []string{
		"BEGIN TRANSACTION",
		"COMMIT",
		"ROLLBACK",
		"PRAGMA foreign_keys=OFF",
		"VACUUM",
		"INSERT INTO sqlite_sequence VALUES ('a', 1)",
		"",
	}

	for _, statement := range skipped {
		if replayable(statement) {
			t.Errorf("%q would be replayed, but it should be skipped", statement)
		}
	}

	kept := []string{
		"CREATE TABLE a (id INTEGER)",
		"INSERT INTO a VALUES (1)",
		"CREATE VIEW b AS SELECT 1",
	}

	for _, statement := range kept {
		if !replayable(statement) {
			t.Errorf("%q would be skipped, but it should be replayed", statement)
		}
	}
}

func TestLiteralsAndIdentifiersAreEscaped(t *testing.T) {
	if held := literalOf("it's"); held != "'it''s'" {
		t.Errorf("literalOf gave %q, wanted %q", held, "'it''s'")
	}

	if held := literalOf(nil); held != NullLiteral {
		t.Errorf("literalOf(nil) gave %q, wanted %q", held, NullLiteral)
	}

	if held := quoted(`a"b`); held != `"a""b"` {
		t.Errorf("quoted gave %q, wanted %q", held, `"a""b"`)
	}
}
