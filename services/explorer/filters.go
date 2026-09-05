package explorer

import "strings"

type Filter struct {
	Column   string
	Operator string
	Value    string
}

func OperatorChoices() []ChoiceView {
	return []ChoiceView{
		{Value: OperatorContains, Label: ContainsLabel},
		{Value: OperatorMisses, Label: MissesLabel},
		{Value: OperatorIs, Label: IsLabel},
		{Value: OperatorIsNot, Label: IsNotLabel},
		{Value: OperatorStarts, Label: StartsLabel},
		{Value: OperatorEnds, Label: EndsLabel},
		{Value: OperatorAbove, Label: AboveLabel},
		{Value: OperatorAtLeast, Label: AtLeastLabel},
		{Value: OperatorBelow, Label: BelowLabel},
		{Value: OperatorAtMost, Label: AtMostLabel},
		{Value: OperatorEmpty, Label: EmptyLabel},
		{Value: OperatorFilled, Label: FilledLabel},
	}
}

func LabelForOperator(operator string) string {
	for _, choice := range OperatorChoices() {
		if choice.Value == operator {
			return choice.Label
		}
	}

	return ContainsLabel
}

func keptFilters(columns []ColumnView, asked []Filter) []FilterView {
	kept := make([]FilterView, 0, len(asked))

	for _, filter := range asked {
		if !columnNamed(columns, filter.Column) {
			continue
		}

		if !knownOperator(filter.Operator) {
			continue
		}

		if filter.Value == "" && !valueless(filter.Operator) {
			continue
		}

		kept = append(kept, FilterView{
			Column:   filter.Column,
			Operator: filter.Operator,
			Value:    filter.Value,
			Label:    LabelForOperator(filter.Operator),
		})
	}

	return kept
}

func whereClause(columns []ColumnView, search string, filters []FilterView) (string, []any) {
	pieces := make([]string, 0, len(filters)+1)
	arguments := make([]any, 0, len(filters)+len(columns))

	if piece, held := searchPiece(columns, search); piece != "" {
		pieces = append(pieces, piece)
		arguments = append(arguments, held...)
	}

	for _, filter := range filters {
		piece, held := filterPiece(filter)
		pieces = append(pieces, piece)
		arguments = append(arguments, held...)
	}

	if len(pieces) == 0 {
		return "", nil
	}

	return " WHERE " + strings.Join(pieces, " AND "), arguments
}

func searchPiece(columns []ColumnView, search string) (string, []any) {
	search = strings.TrimSpace(search)
	if search == "" || len(columns) == 0 {
		return "", nil
	}

	pieces := make([]string, 0, len(columns))
	arguments := make([]any, 0, len(columns))

	for _, column := range columns {
		pieces = append(pieces, "CAST("+quoteIdentifier(column.Name)+" AS TEXT) LIKE ?")
		arguments = append(arguments, "%"+search+"%")
	}

	return "(" + strings.Join(pieces, " OR ") + ")", arguments
}

func filterPiece(filter FilterView) (string, []any) {
	name := quoteIdentifier(filter.Column)
	text := "CAST(" + name + " AS TEXT)"

	switch filter.Operator {
	case OperatorContains:
		return text + " LIKE ?", []any{"%" + filter.Value + "%"}
	case OperatorMisses:
		return text + " NOT LIKE ?", []any{"%" + filter.Value + "%"}
	case OperatorStarts:
		return text + " LIKE ?", []any{filter.Value + "%"}
	case OperatorEnds:
		return text + " LIKE ?", []any{"%" + filter.Value}
	case OperatorIsNot:
		return name + " <> ?", []any{filter.Value}
	case OperatorAbove:
		return name + " > ?", []any{filter.Value}
	case OperatorAtLeast:
		return name + " >= ?", []any{filter.Value}
	case OperatorBelow:
		return name + " < ?", []any{filter.Value}
	case OperatorAtMost:
		return name + " <= ?", []any{filter.Value}
	case OperatorEmpty:
		return name + " IS NULL", nil
	case OperatorFilled:
		return name + " IS NOT NULL", nil
	}

	return name + " = ?", []any{filter.Value}
}

func knownOperator(operator string) bool {
	for _, choice := range OperatorChoices() {
		if choice.Value == operator {
			return true
		}
	}

	return false
}

func valueless(operator string) bool {
	return operator == OperatorEmpty || operator == OperatorFilled
}

func columnChoices(columns []ColumnView) []ChoiceView {
	choices := make([]ChoiceView, 0, len(columns))

	for _, column := range columns {
		choices = append(choices, ChoiceView{Value: column.Name, Label: column.Name})
	}

	return choices
}
