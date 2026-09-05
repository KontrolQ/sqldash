package databases

import (
	"bufio"
	"context"
	"errors"
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
		return shortcuts.ServiceError(http.StatusBadRequest, fmt.Sprintf(ImportRefusedFormat, stageError))
	}

	defer os.Remove(staged)

	if createError := sqld.Create(requestContext, name); createError != nil {
		logger.Errorf(LogPrefix, ImportFailedLog, name, createError)
		return shortcuts.ServiceError(http.StatusBadGateway, ImportRefused)
	}

	if replayError := replay(requestContext, name, staged); replayError != nil {
		logger.Errorf(LogPrefix, ImportFailedLog, name, replayError)
		discard(requestContext, name)

		return shortcuts.ServiceError(http.StatusBadRequest, fmt.Sprintf(ImportRefusedFormat, replayError))
	}

	if registerError := repository.Create(&models.Database{Name: name}); registerError != nil {
		logger.Errorf(LogPrefix, RegisterLog, name, registerError)
		discard(requestContext, name)

		return shortcuts.ServiceError(http.StatusInternalServerError, ImportRefused)
	}

	return nil
}

func discard(requestContext context.Context, name string) {
	if removeError := sqld.Delete(requestContext, name); removeError != nil {
		logger.Errorf(LogPrefix, DeleteFailedLog, name, removeError)
	}
}

func stage(name string, fileName string, contents io.Reader) (string, error) {
	seekable, isSeekable := contents.(io.ReadSeeker)

	if isSeekable {
		sqliteFile, checkError := looksLikeSQLite(seekable)
		if checkError != nil {
			return "", checkError
		}

		if sqliteFile {
			return stageFile(name, seekable)
		}
	}

	if !strings.HasSuffix(strings.ToLower(fileName), DumpSuffix) {
		return "", fmt.Errorf(OnlyKnownFormat, fileName)
	}

	staged := stagedPath(name, StagedNameFormat)

	handle, createError := os.Create(staged)
	if createError != nil {
		return "", createError
	}

	defer handle.Close()

	if _, copyError := io.Copy(handle, contents); copyError != nil {
		os.Remove(staged)
		return "", copyError
	}

	return staged, nil
}

func stageFile(name string, contents io.Reader) (string, error) {
	held := stagedPath(name, FileNameFormat)

	handle, createError := os.Create(held)
	if createError != nil {
		return "", createError
	}

	if _, copyError := io.Copy(handle, contents); copyError != nil {
		handle.Close()
		os.Remove(held)

		return "", copyError
	}

	handle.Close()

	defer os.Remove(held)

	staged := stagedPath(name, StagedNameFormat)

	dump, dumpCreateError := os.Create(staged)
	if dumpCreateError != nil {
		return "", dumpCreateError
	}

	defer dump.Close()

	if dumpError := dumpFromFile(held, dump); dumpError != nil {
		os.Remove(staged)
		return "", dumpError
	}

	return staged, nil
}

func stagedPath(name string, format string) string {
	return filepath.Join(config.ImportsPath(), fmt.Sprintf(format, name, time.Now().UnixNano()))
}

func replay(requestContext context.Context, name string, dumpPath string) error {
	handle, openError := os.Open(dumpPath)
	if openError != nil {
		return openError
	}

	defer handle.Close()

	reader := bufio.NewReaderSize(handle, ReadBufferSize)
	batch := make([]sqld.Statement, 0, ReplayBatch)
	replayed := 0

	for {
		raw, more, readError := nextStatement(reader)
		if readError != nil {
			return readError
		}

		if statement := meaningful(raw); replayable(statement) {
			batch = append(batch, sqld.Statement{SQL: statement})
		}

		if len(batch) >= ReplayBatch || (!more && len(batch) > 0) {
			sent := append([]sqld.Statement{{SQL: ReplayGuard}}, batch...)

			if _, runError := sqld.Run(requestContext, name, sent...); runError != nil {
				return fmt.Errorf(StatementRefusedFormat, shortened(blamed(sent, runError)), runError)
			}

			replayed += len(batch)
			batch = batch[:0]
		}

		if !more {
			if replayed == 0 {
				return errors.New(NothingToReplay)
			}

			return nil
		}
	}
}

func blamed(sent []sqld.Statement, runError error) string {
	refused := &sqld.StatementError{}

	if errors.As(runError, &refused) && refused.At < len(sent) {
		return sent[refused.At].SQL
	}

	return sent[0].SQL
}

func meaningful(statement string) string {
	lines := strings.Split(statement, "\n")

	for index, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), CommentMarker) {
			return strings.TrimSpace(strings.Join(lines[index:], "\n"))
		}
	}

	return ""
}

func replayable(statement string) bool {
	if statement == "" {
		return false
	}

	upper := strings.ToUpper(statement)

	for _, skipped := range SkippedPrefixes {
		if strings.HasPrefix(upper, skipped) {
			return false
		}
	}

	return !strings.Contains(upper, SequenceTable)
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

	if at != nil && at.After(time.Now()) {
		return shortcuts.ServiceError(http.StatusBadRequest, MomentAhead)
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
