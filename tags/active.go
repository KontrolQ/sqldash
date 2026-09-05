package tags

import (
	"sqldash/utils/urls"

	"github.com/flosch/pongo2/v6"
)

func active(value *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	routeName := value.String()

	routePath, exists := urls.GetFullPath(routeName)
	if !exists {
		return pongo2.AsValue(false), nil
	}

	requestPath := param.String()

	return pongo2.AsValue(requestPath == routePath), nil
}
