package databases

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"errors"
	"github.com/gofiber/fiber/v3"
	"sqldash/config"
	"sqldash/engines"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/pgtransfer"
	"sqldash/transferpolicy"
)

func authorizeExport(ctx context.Context, name string) (context.Context, *models.Database, *fiber.Error) {
	fresh, record, err := Authorize(ctx, name, permissions.DatabaseExport, true)
	if err != nil {
		return fresh, nil, err
	}
	if expected := engines.CurrentDatabase(ctx); expected != nil && expected.Name == name && expected.ID != record.ID {
		return fresh, nil, fiber.NewError(fiber.StatusNotFound, DatabaseMissing)
	}
	return fresh, record, nil
}

func WriteFile(requestContext context.Context, databaseName string, writer io.Writer) error {
	fresh, record, err := authorizeExport(requestContext, databaseName)
	if err != nil {
		return err
	}
	if record.Kind() != models.EngineLibSQL {
		return errors.New(EngineRefused)
	}
	requestContext = fresh
	stamp := time.Now().UnixNano()
	staged := filepath.Join(config.ImportsPath(), fmt.Sprintf(StagedNameFormat, databaseName, stamp))
	built := filepath.Join(config.ImportsPath(), fmt.Sprintf(FileNameFormat, databaseName, stamp))

	defer os.Remove(staged)
	defer os.Remove(built)

	dump, createError := os.Create(staged)
	if createError != nil {
		return createError
	}

	if dumpError := WriteDump(requestContext, databaseName, dump); dumpError != nil {
		dump.Close()
		return dumpError
	}

	dump.Close()

	if buildError := fileFromDump(staged, built); buildError != nil {
		return buildError
	}

	handle, openError := os.Open(built)
	if openError != nil {
		return openError
	}

	defer handle.Close()

	_, copyError := transferpolicy.Copy(writer, handle)

	return copyError
}

func WriteDump(requestContext context.Context, databaseName string, writer io.Writer) error {
	fresh, record, err := authorizeExport(requestContext, databaseName)
	if err != nil {
		return err
	}
	if record.Kind() == models.EnginePostgres {
		return WritePostgres(fresh, databaseName, writer, "sql")
	}
	if record.Kind() == models.EngineRedis {
		return WriteRedis(fresh, databaseName, writer)
	}
	return writeLibSQLDump(fresh, databaseName, transferpolicy.Writer(writer))
}

func WritePostgres(ctx context.Context, name string, writer io.Writer, format string) error {
	fresh, record, err := authorizeExport(ctx, name)
	if err != nil {
		return err
	}
	if record.Kind() != models.EnginePostgres {
		return errors.New(EngineRefused)
	}
	return pgtransfer.ExportWithOptions(fresh, record, writer, format, pgtransfer.Options{Authorize: func(step context.Context) error {
		_, current, err := Authorize(step, name, permissions.DatabaseExport, true)
		if err != nil {
			return err
		}
		if current.ID != record.ID {
			return errors.New(DatabaseMissing)
		}
		return nil
	}})
}

type stagedExport struct {
	*os.File
	path       string
	ctx        context.Context
	name       string
	databaseID uint
	checkedAt  time.Time
}

func (self *stagedExport) Read(buffer []byte) (int, error) {
	if self.ctx != nil && time.Since(self.checkedAt) >= time.Second {
		_, record, err := Authorize(self.ctx, self.name, permissions.DatabaseExport, true)
		if err != nil {
			return 0, err
		}
		if record.ID != self.databaseID {
			return 0, errors.New(DatabaseMissing)
		}
		self.checkedAt = time.Now()
	}
	return self.File.Read(buffer)
}

func (self *stagedExport) Close() error {
	err := self.File.Close()
	removeError := os.Remove(self.path)
	if err != nil {
		return err
	}
	return removeError
}

func PostgresExport(ctx context.Context, name, format string) (io.ReadCloser, error) {
	return stagedNativeExport(ctx, name, models.EnginePostgres, config.PostgresStagingPath(), func(parent context.Context, databaseName string, writer io.Writer) error {
		return WritePostgres(parent, databaseName, writer, format)
	})
}

func writeLibSQLDump(requestContext context.Context, databaseName string, writer io.Writer) error {
	if _, writeError := io.WriteString(writer, DumpHeader); writeError != nil {
		return writeError
	}

	schema, schemaError := engines.Query(requestContext, databaseName, SchemaSQL)
	if schemaError != nil {
		return schemaError
	}

	tables := make([]string, 0)

	for _, row := range schema.Rows {
		kind := fmt.Sprint(row[0])
		name := fmt.Sprint(row[1])
		statement := fmt.Sprint(row[2])

		if _, writeError := io.WriteString(writer, statement+StatementEnd); writeError != nil {
			return writeError
		}

		if kind == TableKind {
			tables = append(tables, name)
		}
	}

	for _, table := range tables {
		if writeError := writeRows(requestContext, databaseName, table, writer); writeError != nil {
			return writeError
		}
	}

	_, writeError := io.WriteString(writer, DumpFooter)

	return writeError
}

func writeRows(requestContext context.Context, databaseName string, table string, writer io.Writer) error {
	offset := 0

	for {
		fresh, _, authorityError := authorizeExport(requestContext, databaseName)
		if authorityError != nil {
			return authorityError
		}
		held, queryError := engines.Query(
			fresh, databaseName,
			fmt.Sprintf(PageSQL, quoted(table), ExportPageSize, offset),
		)

		if queryError != nil {
			return queryError
		}

		if len(held.Rows) == 0 {
			return nil
		}

		names := make([]string, 0, len(held.Columns))
		for _, column := range held.Columns {
			names = append(names, quoted(column.Name))
		}

		for _, row := range held.Rows {
			values := make([]string, 0, len(row))

			for _, cell := range row {
				values = append(values, literalOf(cell))
			}

			line := fmt.Sprintf(
				InsertFormat, quoted(table), strings.Join(names, ", "), strings.Join(values, ", "),
			)

			if _, writeError := io.WriteString(writer, line); writeError != nil {
				return writeError
			}
		}

		if len(held.Rows) < ExportPageSize {
			return nil
		}

		offset += ExportPageSize
	}
}

func literalOf(value any) string {
	switch held := value.(type) {
	case nil:
		return NullLiteral
	case int64:
		return fmt.Sprintf("%d", held)
	case float64:
		return fmt.Sprintf("%g", held)
	case []byte:
		return fmt.Sprintf(BlobFormat, held)
	default:
		return "'" + strings.ReplaceAll(fmt.Sprint(held), "'", "''") + "'"
	}
}

func quoted(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
