package css

import (
	"fmt"
	"strings"
)

func Format(source []byte) ([]byte, error) {
	text := strings.ReplaceAll(string(source), "\r\n", "\n")
	lines := []string{}
	depth, parentheses, brackets := 0, 0, 0
	var pending strings.Builder
	emit := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		lines = append(lines, strings.Repeat("  ", depth)+value)
	}
	declaration := func(semicolon bool) {
		value := strings.TrimSpace(pending.String())
		pending.Reset()
		if value == "" {
			return
		}
		if separator := strings.IndexByte(value, ':'); separator > 0 {
			name := strings.TrimSpace(value[:separator])
			property := true
			for _, character := range name {
				if character != '-' && character != '_' && (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') {
					property = false
				}
			}
			if property {
				value = name + ": " + strings.TrimSpace(value[separator+1:])
			}
		}
		if semicolon {
			value += ";"
		}
		emit(value)
	}
	for position := 0; position < len(text); position++ {
		character := text[position]
		if character == '\'' || character == '"' {
			start, quote := position, character
			position++
			for position < len(text) && text[position] != quote {
				if text[position] == '\\' {
					position++
				}
				position++
			}
			if position >= len(text) {
				return nil, fmt.Errorf("unterminated CSS string")
			}
			pending.WriteString(text[start : position+1])
			continue
		}
		if character == '\\' {
			pending.WriteByte(character)
			if position+1 < len(text) {
				position++
				pending.WriteByte(text[position])
			}
			continue
		}
		if character == '/' && position+1 < len(text) && text[position+1] == '*' {
			ending := strings.Index(text[position+2:], "*/")
			if ending < 0 {
				return nil, fmt.Errorf("unterminated CSS comment")
			}
			comment := text[position : position+ending+4]
			if strings.TrimSpace(pending.String()) == "" {
				emit(comment)
				pending.Reset()
			} else {
				pending.WriteString(comment)
			}
			position += ending + 3
			continue
		}
		switch character {
		case '(':
			parentheses++
		case ')':
			parentheses--
			if parentheses < 0 {
				return nil, fmt.Errorf("unmatched CSS parenthesis")
			}
		case '[':
			brackets++
		case ']':
			brackets--
			if brackets < 0 {
				return nil, fmt.Errorf("unmatched CSS bracket")
			}
		}
		if parentheses == 0 && brackets == 0 {
			switch character {
			case '{':
				if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "}" {
					lines = append(lines, "")
				}
				emit(strings.TrimSpace(pending.String()) + " {")
				pending.Reset()
				depth++
				continue
			case '}':
				if depth == 0 {
					return nil, fmt.Errorf("unmatched CSS brace")
				}
				declaration(true)
				depth--
				emit("}")
				continue
			case ';':
				declaration(true)
				continue
			}
		}
		if character == ' ' || character == '\t' || character == '\n' || character == '\r' {
			value := pending.String()
			if len(value) > 0 && value[len(value)-1] != ' ' {
				pending.WriteByte(' ')
			}
			continue
		}
		pending.WriteByte(character)
	}
	if depth != 0 || parentheses != 0 || brackets != 0 {
		return nil, fmt.Errorf("unclosed CSS block")
	}
	declaration(false)
	if len(lines) == 0 {
		return []byte{}, nil
	}
	return []byte(strings.Join(lines, "\n") + "\n"), nil
}
