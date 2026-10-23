package control

import "strings"

func redactSQLValues(statement string) string {
	var redacted strings.Builder
	for position := 0; position < len(statement); {
		if strings.HasPrefix(statement[position:], "--") {
			end := strings.IndexAny(statement[position:], "\r\n")
			redacted.WriteString("-- <private>\n")
			if end < 0 {
				break
			}
			position += end
		} else if strings.HasPrefix(statement[position:], "/*") {
			position += 2
			depth := 1
			for position < len(statement) && depth > 0 {
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
			redacted.WriteString("/* <private> */")
		} else if statement[position] == '\'' {
			position++
			for position < len(statement) {
				if statement[position] == '\\' {
					position++
					if position < len(statement) {
						position++
					}
				} else if statement[position] == '\'' {
					position++
					if position < len(statement) && statement[position] == '\'' {
						position++
					} else {
						break
					}
				} else {
					position++
				}
			}
			redacted.WriteString("'<private>'")
		} else if statement[position] == '"' {
			start := position
			position++
			for position < len(statement) {
				if statement[position] == '"' {
					position++
					if position < len(statement) && statement[position] == '"' {
						position++
					} else {
						break
					}
				} else {
					position++
				}
			}
			redacted.WriteString(statement[start:position])
		} else if delimiter := dollarDelimiter(statement[position:]); delimiter != "" {
			position += len(delimiter)
			end := strings.Index(statement[position:], delimiter)
			if end < 0 {
				position = len(statement)
			} else {
				position += end + len(delimiter)
			}
			redacted.WriteString("'<private>'")
		} else {
			redacted.WriteByte(statement[position])
			position++
		}
	}
	return redacted.String()
}

func dollarDelimiter(statement string) string {
	if len(statement) < 2 || statement[0] != '$' {
		return ""
	}
	if statement[1] == '$' {
		return "$$"
	}
	if !dollarTagCharacter(statement[1], true) {
		return ""
	}
	for position := 2; position < len(statement); position++ {
		if statement[position] == '$' {
			return statement[:position+1]
		}
		if !dollarTagCharacter(statement[position], false) {
			return ""
		}
	}
	return ""
}

func dollarTagCharacter(character byte, first bool) bool {
	return character == '_' || character >= 128 || character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || !first && character >= '0' && character <= '9'
}
