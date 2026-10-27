package models

import "strings"

func (self *Database) Label() string {
	if label := strings.TrimSpace(self.DisplayName); label != "" {
		return label
	}
	return self.Name
}
