package audit

import "testing"

func TestEveryRecordedActionHasALabel(t *testing.T) {
	actions := []string{
		DatabaseCreated, DatabaseImported, DatabaseForked, DatabaseDeleted, DatabaseExported,
		SettingsSaved, TokenMinted, TokenRevoked, NetworkAllowed, NetworkRemoved,
		RowsChanged, RowsDeleted, RowInserted, StatementRun,
		AccountChanged, PasswordChanged, SignedIn, SignedOut,
	}

	for _, action := range actions {
		label := LabelFor(action)

		if label == action {
			t.Errorf("%s has no label, so the log would show the raw key", action)
		}
	}
}

func TestUnknownActionsFallBackToTheirKey(t *testing.T) {
	if held := LabelFor("something.new"); held != "something.new" {
		t.Errorf("LabelFor gave %q, wanted the key back", held)
	}

	if held := toneOf("something.new"); held != PlainTone {
		t.Errorf("toneOf gave %s, wanted %s", held, PlainTone)
	}
}

func TestDestructiveActionsReadAsBad(t *testing.T) {
	bad := []string{DatabaseDeleted, TokenRevoked, NetworkRemoved, RowsDeleted, PasswordChanged}

	for _, action := range bad {
		if toneOf(action) != BadTone {
			t.Errorf("%s reads as %s, wanted %s", action, toneOf(action), BadTone)
		}
	}

	good := []string{DatabaseCreated, TokenMinted, NetworkAllowed}

	for _, action := range good {
		if toneOf(action) != GoodTone {
			t.Errorf("%s reads as %s, wanted %s", action, toneOf(action), GoodTone)
		}
	}
}

func TestOneOfSomethingIsNotPluralised(t *testing.T) {
	cases := []struct {
		count  int
		word   string
		wanted string
	}{
		{1, MinuteWord, "1 minute ago"},
		{2, MinuteWord, "2 minutes ago"},
		{1, HourWord, "1 hour ago"},
		{5, DayWord, "5 days ago"},
	}

	for _, held := range cases {
		if found := plural(held.count, held.word); found != held.wanted {
			t.Errorf("plural(%d, %s) gave %q, wanted %q", held.count, held.word, found, held.wanted)
		}
	}
}
