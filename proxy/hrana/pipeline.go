package hrana

import "encoding/json"

func Measure(asked []byte, held []byte) []Statement {
	sent := statementsAsked(asked)
	if len(sent) == 0 {
		return nil
	}

	answered := resultsHeld(held)
	measured := make([]Statement, 0, len(sent))

	for index, sql := range sent {
		statement := Statement{SQL: sql, Failed: true}

		if index < len(answered) {
			statement = answered[index]
			statement.SQL = sql
		}

		measured = append(measured, statement)
	}

	return measured
}

func statementsAsked(payload []byte) []string {
	pipeline := askedPipeline{}

	if decodeError := json.Unmarshal(payload, &pipeline); decodeError != nil {
		return nil
	}

	sent := make([]string, 0, len(pipeline.Requests))

	for _, request := range pipeline.Requests {
		switch request.Type {
		case ExecuteRequest:
			if request.Stmt != nil {
				sent = append(sent, request.Stmt.SQL)
			}
		case BatchRequest:
			for _, statement := range request.Stmts {
				sent = append(sent, statement.SQL)
			}
		}
	}

	return sent
}

func resultsHeld(payload []byte) []Statement {
	pipeline := heldPipeline{}

	if decodeError := json.Unmarshal(payload, &pipeline); decodeError != nil {
		return nil
	}

	answered := make([]Statement, 0, len(pipeline.Results))

	for _, result := range pipeline.Results {
		statement := Statement{Failed: result.Type != OkResult}

		if result.Response != nil && result.Response.Result != nil {
			held := result.Response.Result
			statement.DurationMs = held.DurationMs
			statement.RowsRead = held.RowsRead
			statement.RowsWritten = held.RowsWritten
			statement.RowsReturned = int64(len(held.Rows))
		}

		answered = append(answered, statement)
	}

	return answered
}
