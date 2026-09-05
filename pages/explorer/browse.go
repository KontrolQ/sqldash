package explorer

import (
	"strconv"

	service "sqldash/services/explorer"
	"sqldash/utils/meta"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Browse(context fiber.Ctx) error {
	page, _ := strconv.Atoi(context.Query(PageParameter))

	data, dataError := service.Browse(context.Context(), service.Ask{
		Database:  context.Params(NameParameter),
		Table:     context.Query(TableParameter),
		Page:      page,
		Sort:      context.Query(SortParameter),
		Direction: context.Query(DirectionParameter),
		Search:    context.Query(SearchParameter),
		Problem:   context.Query(ProblemParameter),
		Done:      context.Query(DoneParameter),
	})

	if dataError != nil {
		return dataError
	}

	meta.SetPageTitle(context, data.Title)

	return shortcuts.Render(context, BrowseTemplate, data)
}

func Console(context fiber.Ctx) error {
	data, dataError := service.RunConsole(
		context.Context(),
		context.Params(NameParameter),
		context.Query(StatementParameter),
	)

	if dataError != nil {
		return dataError
	}

	meta.SetPageTitle(context, data.Title)

	return shortcuts.Render(context, ConsoleTemplate, data)
}
