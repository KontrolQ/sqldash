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
	"sqldash/permissions"
	"sqldash/sqld"
	"sqldash/transferpolicy"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func CreateFromUpload(ctx context.Context, asked, fileName string, contents io.Reader) *fiber.Error {
	return CreateFromUploadEngine(ctx, asked, models.EngineLibSQL, fileName, contents)
}

func CreateFromUploadEngine(ctx context.Context, asked, engine, fileName string, contents io.Reader) *fiber.Error {
	engine = selectedEngine(engine)
	fresh, actor, authorityError := creationActor(ctx, engine, permissions.DatabaseImport)
	if authorityError != nil {
		return authorityError
	}
	if !actor.Can(permissions.ForEngine(permissions.DatabaseCreate, engine)) {
		return shortcuts.ServiceError(http.StatusForbidden, AuthorityRefused)
	}
	name, nameError := acceptableEngineName(asked, engine)
	if nameError != nil {
		return nameError
	}
	var staged string
	var stageError error
	switch engine {
	case models.EnginePostgres:
		staged, stageError = stageNative(contents)
	case models.EngineRedis:
		staged, stageError = stageNativeDirectory(config.RedisStagingPath(), contents)
	default:
		staged, stageError = stage(name, fileName, contents)
	}
	if stageError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, ImportRefused)
	}
	record, operation, reserveError := reserveDatabase(fresh, name, engine, actor.Account.ID, "import", 0, operationPayload{Path: staged})
	if reserveError != nil {
		os.Remove(staged)
		return reserveError
	}
	return executeOperation(fresh, record, operation)
}

func stageNative(contents io.Reader) (string, error) {
	return stageNativeDirectory(config.PostgresStagingPath(), contents)
}

func stageNativeDirectory(directory string, contents io.Reader) (string, error) {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return "", err
	}
	handle, err := os.CreateTemp(directory, "upload-*")
	if err != nil {
		return "", err
	}
	path := handle.Name()
	if err := handle.Chmod(0600); err != nil {
		handle.Close()
		os.Remove(path)
		return "", err
	}
	_, copyError := transferpolicy.Copy(handle, contents)
	closeError := handle.Close()
	if copyError != nil || closeError != nil {
		os.Remove(path)
		return "", errors.New(ImportRefused)
	}
	return path, nil
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

	if _, copyError := transferpolicy.Copy(handle, contents); copyError != nil {
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

	if _, copyError := transferpolicy.Copy(handle, contents); copyError != nil {
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

func Fork(ctx context.Context, from, asked string, at *time.Time) *fiber.Error {
	fresh, source, authorityError := Authorize(ctx, from, permissions.DatabaseFork, true)
	if authorityError != nil {
		return authorityError
	}
	fresh, actor, authorityError := creationActor(fresh, source.Kind(), permissions.DatabaseCreate)
	if authorityError != nil {
		return authorityError
	}
	if !actor.CanDatabase(source, permissions.DataRead) {
		return shortcuts.ServiceError(http.StatusForbidden, AuthorityRefused)
	}
	if source.Kind() != models.EngineLibSQL && at != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, MomentUnusable)
	}
	if at != nil && at.After(time.Now()) {
		return shortcuts.ServiceError(http.StatusBadRequest, MomentAhead)
	}
	name, nameError := acceptableEngineName(asked, source.Kind())
	if nameError != nil {
		return nameError
	}
	record, operation, reserveError := reserveDatabase(fresh, name, source.Kind(), actor.Account.ID, "clone", source.ID, operationPayload{Moment: at})
	if reserveError != nil {
		return reserveError
	}
	return executeOperation(fresh, record, operation)
}
