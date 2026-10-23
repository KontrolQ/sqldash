package postgres

import (
	"strings"

	"sqldash/engines"
)

func validateAtomicStatements(statements []engines.Statement) error {
	for _, statement := range statements {
		switch firstKeyword(statement.SQL) {
		case "BEGIN", "START", "COMMIT", "END", "ROLLBACK", "ABORT", "SAVEPOINT", "RELEASE", "PREPARE":
			return engines.Unsupported
		}
	}
	return nil
}

func firstKeyword(statement string) string {
	for {
		statement = strings.TrimLeft(statement, " \t\r\n\v\f")
		if strings.HasPrefix(statement, "--") {
			end := strings.IndexAny(statement, "\r\n")
			if end < 0 {
				return ""
			}
			statement = statement[end:]
			continue
		}
		if strings.HasPrefix(statement, "/*") {
			depth, position := 1, 2
			for depth > 0 && position < len(statement) {
				if strings.HasPrefix(statement[position:], "/*") {
					depth++
					position += 2
				} else if strings.HasPrefix(statement[position:], "*/") {
					depth--
					position += 2
				} else {
					position++
				}
			}
			if depth != 0 {
				return ""
			}
			statement = statement[position:]
			continue
		}
		break
	}
	end := 0
	for end < len(statement) && (statement[end] >= 'a' && statement[end] <= 'z' || statement[end] >= 'A' && statement[end] <= 'Z') {
		end++
	}
	return strings.ToUpper(statement[:end])
}
