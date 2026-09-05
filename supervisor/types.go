package supervisor

import "time"

type Process struct {
	Name        string
	Path        string
	Arguments   []string
	Environment []string
}

type attempt struct {
	process   Process
	failures  int
	startedAt time.Time
}
