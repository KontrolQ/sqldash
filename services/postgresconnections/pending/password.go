package pending

import "sqldash/pgcluster/control"

func GeneratePassword() ([]byte, error) {
	password, err := control.GeneratePassword()
	return []byte(password), err
}
