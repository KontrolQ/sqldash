package databases

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sqldash/config"
	"sqldash/sqld"
)

func WriteFile(requestContext context.Context, databaseName string, writer io.Writer) error {
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

	_, copyError := io.Copy(writer, handle)

	return copyError
}

func WriteDump(requestContext context.Context, databaseName string, writer io.Writer) error {
	if _, writeError := io.WriteString(writer, DumpHeader); writeError != nil {
		return writeError
	}

	schema, schemaError := sqld.Query(requestContext, databaseName, SchemaSQL)
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
		held, queryError := sqld.Query(
			requestContext, databaseName,
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
