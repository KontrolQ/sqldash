package audit

import "sqldash/utils/collections"

type EntryView struct {
	When     string
	Ago      string
	Database string
	Action   string
	Label    string
	Tone     string
	Detail   string
	Actor    string
}

type LogContext struct {
	Title        string
	Heading      string
	Database     string
	Action       string
	Entries      []EntryView
	Choices      []collections.Option
	Carry        string
	Page         int
	Pages        int
	Total        int64
	PreviousPage int
	NextPage     int
	FirstRow     int
	LastRow      int
}
