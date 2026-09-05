package explorer

import "testing"

func TestFilterPieceCoversEveryOperator(t *testing.T) {
	wanted := map[string]struct {
		clause    string
		arguments []any
	}{
		OperatorContains: {`CAST("age" AS TEXT) LIKE ?`, []any{"%7%"}},
		OperatorMisses:   {`CAST("age" AS TEXT) NOT LIKE ?`, []any{"%7%"}},
		OperatorStarts:   {`CAST("age" AS TEXT) LIKE ?`, []any{"7%"}},
		OperatorEnds:     {`CAST("age" AS TEXT) LIKE ?`, []any{"%7"}},
		OperatorIs:       {`"age" = ?`, []any{"7"}},
		OperatorIsNot:    {`"age" <> ?`, []any{"7"}},
		OperatorAbove:    {`"age" > ?`, []any{"7"}},
		OperatorAtLeast:  {`"age" >= ?`, []any{"7"}},
		OperatorBelow:    {`"age" < ?`, []any{"7"}},
		OperatorAtMost:   {`"age" <= ?`, []any{"7"}},
		OperatorEmpty:    {`"age" IS NULL`, nil},
		OperatorFilled:   {`"age" IS NOT NULL`, nil},
	}

	for _, choice := range OperatorChoices() {
		expected, listed := wanted[choice.Value]
		if !listed {
			t.Fatalf("operator %s is offered but this test does not cover it", choice.Value)
		}

		clause, arguments := filterPiece(FilterView{Column: "age", Operator: choice.Value, Value: "7"})

		if clause != expected.clause {
			t.Errorf("%s built %q, wanted %q", choice.Value, clause, expected.clause)
		}

		if len(arguments) != len(expected.arguments) {
			t.Fatalf("%s bound %d values, wanted %d", choice.Value, len(arguments), len(expected.arguments))
		}

		for index, argument := range arguments {
			if argument != expected.arguments[index] {
				t.Errorf("%s bound %v at %d, wanted %v", choice.Value, argument, index, expected.arguments[index])
			}
		}
	}
}

func TestValuelessOperatorsSurviveAnEmptyValue(t *testing.T) {
	columns := []ColumnView{{Name: "note"}}

	kept := keptFilters(columns, []Filter{
		{Column: "note", Operator: OperatorEmpty},
		{Column: "note", Operator: OperatorContains},
	})

	if len(kept) != 1 {
		t.Fatalf("kept %d filters, wanted only the valueless one", len(kept))
	}

	if kept[0].Operator != OperatorEmpty {
		t.Errorf("kept %s, wanted %s", kept[0].Operator, OperatorEmpty)
	}
}

func TestUnknownColumnsAndOperatorsAreDropped(t *testing.T) {
	columns := []ColumnView{{Name: "note"}}

	kept := keptFilters(columns, []Filter{
		{Column: "missing", Operator: OperatorIs, Value: "x"},
		{Column: "note", Operator: "sideways", Value: "x"},
		{Column: "note", Operator: OperatorIs, Value: "x"},
	})

	if len(kept) != 1 {
		t.Fatalf("kept %d filters, wanted 1", len(kept))
	}

	if kept[0].Column != "note" || kept[0].Operator != OperatorIs {
		t.Errorf("kept %s %s, wanted note is", kept[0].Column, kept[0].Operator)
	}
}

func TestWhereClauseJoinsSearchAndFiltersWithAnd(t *testing.T) {
	columns := []ColumnView{{Name: "title"}, {Name: "year"}}
	filters := []FilterView{{Column: "year", Operator: OperatorAbove, Value: "1950"}}

	clause, arguments := whereClause(columns, "wizard", filters)

	expected := ` WHERE (CAST("title" AS TEXT) LIKE ? OR CAST("year" AS TEXT) LIKE ?) AND "year" > ?`
	if clause != expected {
		t.Errorf("built %q, wanted %q", clause, expected)
	}

	if len(arguments) != 3 {
		t.Fatalf("bound %d values, wanted 3", len(arguments))
	}

	if arguments[0] != "%wizard%" || arguments[2] != "1950" {
		t.Errorf("bound %v, wanted the search twice then the filter value", arguments)
	}
}

func TestWhereClauseIsEmptyWithoutSearchOrFilters(t *testing.T) {
	clause, arguments := whereClause([]ColumnView{{Name: "title"}}, "  ", nil)

	if clause != "" || arguments != nil {
		t.Errorf("built %q with %v, wanted nothing", clause, arguments)
	}
}

func TestQuotingClosesAnInjectedIdentifier(t *testing.T) {
	clause, _ := filterPiece(FilterView{Column: `a" OR 1=1 --`, Operator: OperatorIs, Value: "x"})

	expected := `"a"" OR 1=1 --" = ?`
	if clause != expected {
		t.Errorf("built %q, wanted %q", clause, expected)
	}
}
