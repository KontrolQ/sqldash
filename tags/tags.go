package tags

import (
	"sqldash/utils/logger"

	"github.com/flosch/pongo2/v6"
)

type TemplateTag struct {
	Name   string
	Parser pongo2.TagParser
}

type TemplateFilter struct {
	Name   string
	Filter pongo2.FilterFunction
}

func Initialize() {
	tags := []TemplateTag{
		{Name: "static", Parser: static},
		{Name: "url", Parser: url},
	}

	filters := []TemplateFilter{
		{Name: "active", Filter: active},
		{Name: "timeago", Filter: timeago},
	}

	for _, tag := range tags {
		if registrationError := pongo2.RegisterTag(tag.Name, tag.Parser); registrationError != nil {
			logger.Errorf(LogPrefix, RegistrationFailed, tag.Name)
		}
	}

	for _, filter := range filters {
		if registrationError := pongo2.RegisterFilter(filter.Name, filter.Filter); registrationError != nil {
			logger.Errorf(LogPrefix, RegistrationFailed, filter.Name)
		}
	}
}
