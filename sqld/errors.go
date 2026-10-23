package sqld

import "fmt"

type AdminError struct {
	Status  int
	Message string
}

func (self *AdminError) Error() string {
	return fmt.Sprintf("libSQL administration returned %d: %s", self.Status, self.Message)
}
