package django

import (
	"fmt"
	"strings"
)

type token struct {
	text     string
	name     string
	markup   bool
	closing  bool
	void     bool
	raw      bool
	template bool
}

func tokenize(text string) ([]token, error) {
	var tokens []token
	for position := 0; position < len(text); {
		if strings.HasPrefix(text[position:], "{#") {
			ending := strings.Index(text[position+2:], "#}")
			if ending < 0 {
				return nil, fmt.Errorf("unterminated template comment")
			}
			ending += position + 4
			tokens = append(tokens, token{text: text[position:ending], raw: true})
			position = ending
			continue
		}
		if strings.HasPrefix(text[position:], "{%") {
			ending, scanError := templateEnd(text, position, "%}")
			if scanError != nil {
				return nil, scanError
			}
			value := text[position:ending]
			fields := strings.Fields(value[2 : len(value)-2])
			if len(fields) == 0 {
				return nil, fmt.Errorf("empty template tag")
			}
			current := token{text: value, name: fields[0], template: true}
			if current.name == "verbatim" || current.name == "comment" {
				endTag := "{% end" + current.name + " %}"
				endPosition := strings.Index(text[ending:], endTag)
				if endPosition < 0 {
					return nil, fmt.Errorf("unclosed %s tag", current.name)
				}
				ending += endPosition + len(endTag)
				current.text, current.raw, current.template = text[position:ending], true, false
			}
			tokens = append(tokens, current)
			position = ending
			continue
		}
		if strings.HasPrefix(text[position:], "<!--") {
			ending := strings.Index(text[position+4:], "-->")
			if ending < 0 {
				return nil, fmt.Errorf("unterminated HTML comment")
			}
			ending += position + 7
			tokens = append(tokens, token{text: text[position:ending], raw: true})
			position = ending
			continue
		}
		if text[position] == '<' && position+1 < len(text) && (isLetter(text[position+1]) || text[position+1] == '/' || text[position+1] == '!') {
			ending, scanError := markupEnd(text, position)
			if scanError != nil {
				return nil, scanError
			}
			value := text[position:ending]
			nameStart := 1
			closing := strings.HasPrefix(value, "</")
			if closing {
				nameStart++
			}
			nameEnd := nameStart
			for nameEnd < len(value) && (isLetter(value[nameEnd]) || value[nameEnd] >= '0' && value[nameEnd] <= '9' || value[nameEnd] == '-') {
				nameEnd++
			}
			name := strings.ToLower(value[nameStart:nameEnd])
			current := token{text: value, name: name, markup: true, closing: closing, void: voidElements[name] || strings.HasSuffix(value, "/>") || name == ""}
			if !closing && rawElements[name] {
				lower := strings.ToLower(text[ending:])
				closeStart := strings.Index(lower, "</"+name)
				if closeStart < 0 {
					return nil, fmt.Errorf("unclosed %s element", name)
				}
				closeEnd, scanError := markupEnd(text, ending+closeStart)
				if scanError != nil {
					return nil, scanError
				}
				ending = closeEnd
				current.text, current.raw = text[position:ending], true
			}
			tokens = append(tokens, current)
			position = ending
			continue
		}
		start := position
		for position < len(text) {
			if strings.HasPrefix(text[position:], "{{") {
				ending, scanError := templateEnd(text, position, "}}")
				if scanError != nil {
					return nil, scanError
				}
				position = ending
				continue
			}
			if position != start && (text[position] == '<' || strings.HasPrefix(text[position:], "{%") || strings.HasPrefix(text[position:], "{#")) {
				break
			}
			position++
		}
		tokens = append(tokens, token{text: text[start:position]})
	}
	return tokens, nil
}

func isLetter(character byte) bool {
	return character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z'
}

func templateEnd(text string, start int, ending string) (int, error) {
	var quote byte
	for position := start + 2; position < len(text); position++ {
		character := text[position]
		if quote != 0 {
			if character == '\\' {
				position++
				continue
			}
			if character == quote {
				quote = 0
			}
			continue
		}
		if character == '\'' || character == '"' {
			quote = character
			continue
		}
		if strings.HasPrefix(text[position:], ending) {
			return position + 2, nil
		}
	}
	return 0, fmt.Errorf("unterminated template expression")
}

func markupEnd(text string, start int) (int, error) {
	var quote byte
	for position := start + 1; position < len(text); position++ {
		if strings.HasPrefix(text[position:], "{{") || strings.HasPrefix(text[position:], "{%") {
			ending := "}}"
			if text[position+1] == '%' {
				ending = "%}"
			}
			end, scanError := templateEnd(text, position, ending)
			if scanError != nil {
				return 0, scanError
			}
			position = end - 1
			continue
		}
		character := text[position]
		if quote != 0 {
			if character == quote {
				quote = 0
			}
			continue
		}
		if character == '\'' || character == '"' {
			quote = character
			continue
		}
		if character == '>' {
			return position + 1, nil
		}
	}
	return 0, fmt.Errorf("unterminated HTML tag")
}
