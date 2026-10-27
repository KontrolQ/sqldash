package settings

import (
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	"sqldash/pictures"
	repository "sqldash/repositories/account"
	"sqldash/services/access"
	"sqldash/utils/logger"

	"github.com/gofiber/fiber/v3"
)

func ShowPicture(context fiber.Ctx) error {
	context.Set("Cache-Control", "no-store")
	actor := access.FromContext(context.Context())
	if actor == nil || !actor.Account.Active {
		return context.Status(http.StatusNotFound).Send(nil)
	}
	reference := pictures.ReferencePrefix + context.Params("filename")
	filename, mediaType, validationError := pictures.StoredName(reference)
	if validationError != nil {
		return context.Status(http.StatusNotFound).Send(nil)
	}
	visible, readingError := repository.PictureVisible(context.Context(), actor.Account.ID, reference, mediaType)
	if readingError != nil {
		logger.Errorf(LogPrefix, PictureReadFailedLog, readingError)
		return context.Status(http.StatusInternalServerError).Send(nil)
	}
	if !visible {
		return context.Status(http.StatusNotFound).Send(nil)
	}

	etag := `"` + filename + `"`
	context.Set("Cache-Control", "private, no-cache")
	context.Set("ETag", etag)
	context.Set("Content-Type", mediaType)
	context.Set("X-Content-Type-Options", "nosniff")
	for _, offered := range strings.Split(context.Get("If-None-Match"), ",") {
		offered = strings.TrimSpace(offered)
		if offered == "*" || strings.TrimPrefix(offered, "W/") == etag {
			return context.Status(http.StatusNotModified).Send(nil)
		}
	}
	file, _, openingError := pictures.Open(reference)
	if openingError != nil {
		if !errors.Is(openingError, os.ErrNotExist) {
			logger.Errorf(LogPrefix, PictureReadFailedLog, openingError)
		}
		context.Set("Cache-Control", "no-store")
		return context.Status(http.StatusNotFound).Send(nil)
	}
	information, inspectionError := file.Stat()
	if inspectionError != nil {
		file.Close()
		return context.Status(http.StatusInternalServerError).Send(nil)
	}
	context.Set("Content-Length", strconv.FormatInt(information.Size(), 10))
	if context.Method() == http.MethodHead {
		file.Close()
		return context.Send(nil)
	}
	return context.SendStream(file, int(information.Size()))
}
