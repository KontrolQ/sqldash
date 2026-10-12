package django

import "strings"

var voidElements = map[string]bool{"area": true, "base": true, "br": true, "col": true, "embed": true, "hr": true, "img": true, "input": true, "link": true, "meta": true, "param": true, "source": true, "track": true, "wbr": true}
var rawElements = map[string]bool{"pre": true, "textarea": true, "script": true, "style": true}
var blockElements = map[string]bool{"link": true, "meta": true, "base": true, "hr": true, "html": true, "head": true, "body": true, "main": true, "div": true, "section": true, "header": true, "footer": true, "aside": true, "nav": true, "form": true, "fieldset": true, "legend": true, "table": true, "thead": true, "tbody": true, "tfoot": true, "tr": true, "th": true, "td": true, "ul": true, "ol": true, "li": true, "dl": true, "dt": true, "dd": true, "p": true, "h1": true, "h2": true, "h3": true, "h4": true, "details": true, "summary": true}
var templateBlocks = map[string]bool{"if": true, "for": true, "block": true, "with": true, "autoescape": true, "filter": true, "macro": true, "spaceless": true, "ifchanged": true}

func Format(source []byte) ([]byte, error) {
	text := strings.ReplaceAll(string(source), "\r\n", "\n")
	tokens, tokenError := tokenize(text)
	if tokenError != nil {
		return nil, tokenError
	}
	var lines []string
	depth := 0
	var inline strings.Builder
	emit := func(value string) {
		value = strings.TrimSpace(value)
		for _, line := range strings.Split(value, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				lines = append(lines, strings.Repeat("  ", depth)+line)
			}
		}
	}
	flush := func() { emit(inline.String()); inline.Reset() }
	for index := 0; index < len(tokens); index++ {
		current := tokens[index]
		if current.raw {
			flush()
			lines = append(lines, strings.Repeat("  ", depth)+strings.TrimSpace(current.text))
			continue
		}
		if current.markup {
			flush()
			if !current.closing && !current.void {
				ending := inlineEnding(tokens, index)
				if ending > index {
					var leaf strings.Builder
					for _, item := range tokens[index : ending+1] {
						leaf.WriteString(item.text)
					}
					lines = append(lines, strings.Repeat("  ", depth)+strings.TrimSpace(leaf.String()))
					index = ending
					continue
				}
			}
			if current.closing && depth > 0 {
				depth--
			}
			emit(current.text)
			if !current.closing && !current.void {
				depth++
			}
			continue
		}
		if current.template {
			name := current.name
			ending := strings.HasPrefix(name, "end") && templateBlocks[strings.TrimPrefix(name, "end")]
			branch := name == "else" || name == "elif" || name == "empty"
			if templateBlocks[name] || ending || branch {
				flush()
				if (ending || branch) && depth > 0 {
					depth--
				}
				emit(current.text)
				if templateBlocks[name] || branch {
					depth++
				}
				continue
			}
			if inline.Len() == 0 {
				emit(current.text)
				continue
			}
		}
		inline.WriteString(current.text)
	}
	flush()
	if len(lines) == 0 {
		return []byte{}, nil
	}
	return []byte(strings.Join(lines, "\n") + "\n"), nil
}

func inlineEnding(tokens []token, opening int) int {
	name := tokens[opening].name
	nested := 0
	for index := opening + 1; index < len(tokens); index++ {
		current := tokens[index]
		if current.raw {
			return -1
		}
		if current.markup {
			if current.name == name {
				if current.closing {
					if nested == 0 {
						return index
					}
					nested--
				} else if !current.void {
					nested++
				}
			} else if blockElements[current.name] {
				return -1
			}
		}
	}
	return -1
}
