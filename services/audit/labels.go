package audit

import (
	"strconv"
	"strings"
	"unicode"
)

var labels = map[string]string{
	DatabaseCreated:                 "Database created",
	DatabaseImported:                "Database loaded from a file",
	DatabaseForked:                  "Database copied",
	DatabaseDeleted:                 "Database deleted",
	DatabaseExported:                "Database downloaded",
	SettingsSaved:                   "Configuration saved",
	TokenMinted:                     "Token created",
	TokenRevoked:                    "Token revoked",
	NetworkAllowed:                  "Address range allowed",
	NetworkRemoved:                  "Address range removed",
	RowsChanged:                     "Rows changed",
	RowsDeleted:                     "Rows deleted",
	RowInserted:                     "Record added",
	StatementRun:                    "SQL console submitted",
	StatementCancelled:              "SQL console cancelled",
	SnippetSaved:                    "Saved query saved",
	SnippetRemoved:                  "Saved query removed",
	CredentialViewed:                "PostgreSQL credentials viewed",
	CredentialRevealed:              "PostgreSQL credential reveal requested",
	DatabaseCloned:                  "Database cloned from a remote source",
	ObjectCreated:                   "Database object created",
	ObjectDropped:                   "Database object dropped",
	ObjectRenamed:                   "Database object renamed",
	ObjectRefreshed:                 "Materialized view refreshed",
	AccountChanged:                  "Account details changed",
	PasswordChanged:                 "Password changed",
	SignedIn:                        "Signed in",
	SignedOut:                       "Signed out",
	"administration.changed":        "Administration settings changed",
	"invitation.created":            "Invitation link generated",
	"invitation.cancelled":          "Invitation cancelled",
	"invitation.accepted":           "Invitation accepted",
	"recovery.created":              "Recovery link generated",
	"recovery.completed":            "Password recovered",
	"user.access.changed":           "User roles and database access changed",
	"user.activated":                "User reactivated",
	"user.deactivated":              "User deactivated",
	"user.sessions.revoked":         "User sessions revoked",
	"role.saved":                    "Role permissions saved",
	"role.deleted":                  "Role deleted",
	"role.users.changed":            "Role membership changed",
	"role.reordered":                "Role order changed",
	"smtp.saved":                    "SMTP settings saved",
	"smtp.cleared":                  "SMTP configuration cleared",
	"smtp.test":                     "SMTP test email sent",
	"ownership.requested":           "Ownership transfer requested",
	"ownership.responded":           "Ownership transfer answered",
	"ownership.accepted":            "Site ownership transferred",
	"ownership.declined":            "Ownership transfer declined",
	"ownership.cancelled":           "Ownership transfer cancelled",
	"database.shared":               "Database access shared",
	"database.unshared":             "Database access removed",
	"postgres.object.changed":       "Database structure changed",
	"postgres.activity.cancelled":   "PostgreSQL query cancelled",
	"postgres.activity.terminated":  "PostgreSQL connection terminated",
	"postgres.extensions.installed": "PostgreSQL extension installed",
	"postgres.extensions.removed":   "PostgreSQL extension removed",
	"postgres.connection.created":   "PostgreSQL connection user created",
	"postgres.connection.rotated":   "PostgreSQL password rotated",
	"postgres.connection.revoked":   "PostgreSQL connection revoked",
	"postgres.settings.saved":       "PostgreSQL settings saved",
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
	ObjectCreated:    GoodTone,
	ObjectDropped:    BadTone,
	SnippetRemoved:   BadTone,
	DatabaseCloned:   GoodTone,
}

func ObjectAction(operation string) string {
	switch operation {
	case "create":
		return ObjectCreated
	case "drop":
		return ObjectDropped
	case "rename":
		return ObjectRenamed
	case "refresh":
		return ObjectRefreshed
	default:
		return ObjectChanged
	}
}

func LabelFor(action string) string {
	if held, matched := labels[action]; matched {
		return held
	}

	words := strings.Fields(strings.NewReplacer(".", " ", "_", " ", "-", " ").Replace(action))
	for index, word := range words {
		characters := []rune(word)
		if len(characters) > 0 {
			characters[0] = unicode.ToUpper(characters[0])
			words[index] = string(characters)
		}
	}
	return strings.Join(words, " ")
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
