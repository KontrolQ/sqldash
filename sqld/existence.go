package sqld

import (
	"context"
	"errors"
	"net/http"
)

func NamespaceExists(parent context.Context, name string) (bool, error) {
	_, queryError := Configuration(parent, name)
	if queryError == nil {
		return true, nil
	}
	var response *AdminError
	if errors.As(queryError, &response) && response.Status == http.StatusNotFound {
		return false, nil
	}
	return false, queryError
}
