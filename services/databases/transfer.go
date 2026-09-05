package databases

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sqldash/config"
	"sqldash/models"
	repository "sqldash/repositories/database"
	"sqldash/sqld"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func CreateFromUpload(requestContext context.Context, asked string, fileName string, contents io.Reader) *fiber.Error {
	name, guardError := AcceptableName(asked)
	if guardError != nil {
		return guardError
	}

	staged, stageError := stage(name, fileName, contents)
	if stageError != nil {
		logger.Errorf(LogPrefix, ImportFailedLog, name, stageError)
		return shortcuts.ServiceError(http.StatusInternalServerError, ImportRefused)
	}

	defer os.Remove(staged)

	if createError := sqld.CreateFromDump(requestContext, name, staged); createError != nil {
		logger.Errorf(LogPrefix, ImportFailedLog, name, createError)
		return shortcuts.ServiceError(http.StatusBadGateway, ImportRefused)
	}

	if registerError := repository.Create(&models.Database{Name: name}); registerError != nil {
		logger.Errorf(LogPrefix, RegisterLog, name, registerError)
		return shortcuts.ServiceError(http.StatusInternalServerError, ImportRefused)
	}

	return nil
}

func stage(name string, fileName string, contents io.Reader) (string, error) {
	staged := filepath.Join(config.ImportsPath(), fmt.Sprintf(StagedNameFormat, name, time.Now().UnixNano()))

	handle, createError := os.Create(staged)
	if createError != nil {
		return "", createError
	}

	defer handle.Close()

	if strings.HasSuffix(strings.ToLower(fileName), DumpSuffix) {
		if _, copyError := io.Copy(handle, contents); copyError != nil {
			return "", copyError
		}

		return staged, nil
	}

	return "", fmt.Errorf(OnlyDumpsFormat, fileName)
}

func Fork(requestContext context.Context, from string, asked string, at *time.Time) *fiber.Error {
	name, guardError := AcceptableName(asked)
	if guardError != nil {
		return guardError
	}

	source, findError := repository.FindByName(from)
	if findError != nil || source == nil {
		return shortcuts.ServiceError(http.StatusNotFound, DatabaseMissing)
	}

	if forkError := sqld.Fork(requestContext, from, name, at); forkError != nil {
		logger.Errorf(LogPrefix, ForkFailedLog, from, forkError)
		return shortcuts.ServiceError(http.StatusBadGateway, ForkRefused)
	}

	if registerError := repository.Create(&models.Database{Name: name}); registerError != nil {
		logger.Errorf(LogPrefix, RegisterLog, name, registerError)
		return shortcuts.ServiceError(http.StatusInternalServerError, ForkRefused)
	}

	return nil
}
