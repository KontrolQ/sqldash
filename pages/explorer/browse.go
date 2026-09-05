package explorer

import (
	"net/url"
	"strconv"

	service "sqldash/services/explorer"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Browse(context fiber.Ctx) error {
	page, _ := strconv.Atoi(context.Query(PageParameter))
	name := context.Params(NameParameter)

	data, dataError := service.Browse(context.Context(), service.Ask{
		Database:  name,
		Table:     context.Query(TableParameter),
		Page:      page,
		Sort:      context.Query(SortParameter),
		Direction: context.Query(DirectionParameter),
		Search:    context.Query(SearchParameter),
		Filters:   filtersAsked(context),
	})

	if dataError != nil {
		return dataError
	}

	meta.SetPageTitle(context, data.Title)
	describe(context, name, DataLabel, DataSection)

	return shortcuts.Render(context, BrowseTemplate, data)
}

func Console(context fiber.Ctx) error {
	name := context.Params(NameParameter)
	chosen, _ := strconv.Atoi(context.Query(SnippetParameter))

	statement := ""
	if chosen > 0 {
		statement = service.SnippetStatement(name, uint(chosen))
	}

	data, dataError := service.RunConsole(context.Context(), name, "")
	if dataError != nil {
		return dataError
	}

	data.Statement = statement
	data.SnippetID = uint(chosen)

	for _, one := range data.Snippets {
		if one.ID == data.SnippetID {
			data.Name = one.Name
		}
	}

	meta.SetPageTitle(context, data.Title)
	describe(context, name, ConsoleLabel, ConsoleSection)

	return shortcuts.Render(context, ConsoleTemplate, data)
}

func describe(context fiber.Ctx, name string, label string, section string) {
	meta.SetCrumbs(context,
		meta.Crumb{Label: DatabasesLabel, URL: DatabasesPath},
		meta.Crumb{Label: name, URL: DatabasePath + url.PathEscape(name)},
		meta.Crumb{Label: label},
	)
	meta.SetSection(context, name, section)
}

func filtersAsked(context fiber.Ctx) []service.Filter {
	arguments := context.RequestCtx().QueryArgs()

	columns := arguments.PeekMulti(ColumnParameter)
	operators := arguments.PeekMulti(OperatorParameter)
	values := arguments.PeekMulti(ValueParameter)

	asked := make([]service.Filter, 0, len(columns))

	for index, column := range columns {
		if index >= len(operators) || index >= len(values) {
			break
		}

		asked = append(asked, service.Filter{
			Column:   string(column),
			Operator: string(operators[index]),
			Value:    string(values[index]),
		})
	}

	return asked
}
