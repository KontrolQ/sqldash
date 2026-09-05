package shortcuts

import (
	"fmt"
	"path"

	"sqldash/utils/collections"

	"github.com/gofiber/fiber/v3"
)

func Render(context fiber.Ctx, templateName string, data any) error {
	templateData := make(collections.Record[string, any])

	mergeContextValues(context, templateData)

	if data != nil {
		if mergeError := mergeBindData(templateData, data); mergeError != nil {
			return mergeError
		}
	}

	return context.Render(resolveTemplate(context, templateName), templateData)
}

func RenderWithStatus(context fiber.Ctx, templateName string, data any, statusCode int) error {
	context.Status(statusCode)
	return Render(context, templateName, data)
}

func NoContent(context fiber.Ctx) error {
	return context.SendStatus(fiber.StatusNoContent)
}

func resolveTemplate(context fiber.Ctx, templateName string) string {
	if !isHtmxRequest(context) {
		return templateName
	}

	return fmt.Sprintf(PartialTemplateFormat, path.Dir(templateName), path.Base(templateName))
}
