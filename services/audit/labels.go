package audit

import "strconv"

var labels = map[string]string{
	DatabaseCreated:  "Database created",
	DatabaseImported: "Database loaded from a file",
	DatabaseForked:   "Database copied",
	DatabaseDeleted:  "Database deleted",
	DatabaseExported: "Database downloaded",
	SettingsSaved:    "Configuration saved",
	TokenMinted:      "Token created",
	TokenRevoked:     "Token revoked",
	NetworkAllowed:   "Address range allowed",
	NetworkRemoved:   "Address range removed",
	RowsChanged:      "Rows changed",
	RowsDeleted:      "Rows deleted",
	RowInserted:      "Record added",
	StatementRun:     "Statement run",
	AccountChanged:   "Account details changed",
	PasswordChanged:  "Password changed",
	SignedIn:         "Signed in",
	SignedOut:        "Signed out",
}

var tones = map[string]string{
	DatabaseCreated:  GoodTone,
	DatabaseImported: GoodTone,
	DatabaseForked:   GoodTone,
	DatabaseDeleted:  BadTone,
	TokenMinted:      GoodTone,
	TokenRevoked:     BadTone,
	NetworkAllowed:   GoodTone,
	NetworkRemoved:   BadTone,
	RowsDeleted:      BadTone,
	PasswordChanged:  BadTone,
}

func LabelFor(action string) string {
	if held, matched := labels[action]; matched {
		return held
	}

	return action
}

func toneOf(action string) string {
	if held, matched := tones[action]; matched {
		return held
	}

	return PlainTone
}

func plural(count int, word string) string {
	if count == 1 {
		return strconv.Itoa(count) + " " + word + AgoSuffix
	}

	return strconv.Itoa(count) + " " + word + Plural + AgoSuffix
}
