package databases

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"

	_ "github.com/glebarez/go-sqlite"
)

func looksLikeSQLite(handle io.ReadSeeker) (bool, error) {
	header := make([]byte, len(SQLiteMagic))

	read, readError := io.ReadFull(handle, header)
	if readError != nil && readError != io.ErrUnexpectedEOF && readError != io.EOF {
		return false, readError
	}

	if _, seekError := handle.Seek(0, io.SeekStart); seekError != nil {
		return false, seekError
	}

	return read == len(SQLiteMagic) && string(header) == SQLiteMagic, nil
}

func dumpFromFile(path string, writer io.Writer) error {
	handle, openError := sql.Open(SQLiteDriver, ReadOnlySource+path)
	if openError != nil {
		return openError
	}

	defer handle.Close()

	if _, writeError := io.WriteString(writer, DumpHeader); writeError != nil {
		return writeError
	}

	schema, schemaError := handle.Query(SchemaSQL)
	if schemaError != nil {
		return schemaError
	}

	tables := make([]string, 0)

	for schema.Next() {
		var kind, name, statement string

		if scanError := schema.Scan(&kind, &name, &statement); scanError != nil {
			schema.Close()
			return scanError
		}

		if _, writeError := io.WriteString(writer, statement+StatementEnd); writeError != nil {
			schema.Close()
			return writeError
		}

		if kind == TableKind {
			tables = append(tables, name)
		}
	}

	schema.Close()

	if schemaError = schema.Err(); schemaError != nil {
		return schemaError
	}

	for _, table := range tables {
		if writeError := writeFileRows(handle, table, writer); writeError != nil {
			return writeError
		}
	}

	_, writeError := io.WriteString(writer, DumpFooter)

	return writeError
}

func writeFileRows(handle *sql.DB, table string, writer io.Writer) error {
	rows, queryError := handle.Query(fmt.Sprintf(EveryRowSQL, quoted(table)))
	if queryError != nil {
		return queryError
	}

	defer rows.Close()

	columns, columnsError := rows.Columns()
	if columnsError != nil {
		return columnsError
	}

	names := make([]string, 0, len(columns))
	for _, column := range columns {
		names = append(names, quoted(column))
	}

	for rows.Next() {
		cells := make([]any, len(columns))
		holders := make([]any, len(columns))

		for index := range cells {
			holders[index] = &cells[index]
		}

		if scanError := rows.Scan(holders...); scanError != nil {
			return scanError
		}

		values := make([]string, 0, len(cells))
		for _, cell := range cells {
			values = append(values, literalOf(cell))
		}

		line := fmt.Sprintf(InsertFormat, quoted(table), strings.Join(names, ", "), strings.Join(values, ", "))

		if _, writeError := io.WriteString(writer, line); writeError != nil {
			return writeError
		}
	}

	return rows.Err()
}

func fileFromDump(dumpPath string, filePath string) error {
	if removeError := os.Remove(filePath); removeError != nil && !os.IsNotExist(removeError) {
		return removeError
	}

	handle, openError := sql.Open(SQLiteDriver, filePath)
	if openError != nil {
		return openError
	}

	defer handle.Close()

	dump, readError := os.Open(dumpPath)
	if readError != nil {
		return readError
	}

	defer dump.Close()

	reader := bufio.NewReaderSize(dump, ReadBufferSize)

	for {
		statement, more, statementError := nextStatement(reader)
		if statementError != nil {
			return statementError
		}

		if strings.TrimSpace(statement) != "" {
			if _, runError := handle.Exec(statement); runError != nil {
				return fmt.Errorf(ReplayFailedFormat, shortened(statement), runError)
			}
		}

		if !more {
			return nil
		}
	}
}

func nextStatement(reader *bufio.Reader) (string, bool, error) {
	var builder strings.Builder

	quote := rune(0)

	for {
		letter, _, readError := reader.ReadRune()
		if readError == io.EOF {
			return builder.String(), false, nil
		}

		if readError != nil {
			return "", false, readError
		}

		if quote != 0 {
			builder.WriteRune(letter)

			if letter == quote {
				quote = 0
			}

			continue
		}

		switch letter {
		case '\'', '"', '`':
			quote = letter
			builder.WriteRune(letter)
		case ';':
			return builder.String(), true, nil
		default:
			builder.WriteRune(letter)
		}
	}
}

func shortened(statement string) string {
	trimmed := strings.TrimSpace(statement)

	if len(trimmed) > SnippetLength {
		return trimmed[:SnippetLength] + Ellipsis
	}

	return trimmed
}
