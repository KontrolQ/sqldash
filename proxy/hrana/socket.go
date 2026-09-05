package hrana

import "encoding/json"

type sentFrame struct {
	Type      string       `json:"type"`
	RequestID *int64       `json:"request_id"`
	Request   *sentRequest `json:"request"`
}

type sentRequest struct {
	Type  string     `json:"type"`
	Stmt  *askedStmt `json:"stmt"`
	Batch *sentBatch `json:"batch"`
}

type sentBatch struct {
	Steps []struct {
		Stmt *askedStmt `json:"stmt"`
	} `json:"steps"`
}

type heldFrame struct {
	Type      string           `json:"type"`
	RequestID *int64           `json:"request_id"`
	Response  *heldResponse    `json:"response"`
	Error     *json.RawMessage `json:"error"`
}

func StatementsSent(frame []byte) (int64, []string) {
	sent := sentFrame{}

	if decodeError := json.Unmarshal(frame, &sent); decodeError != nil {
		return 0, nil
	}

	if sent.Type != RequestFrame || sent.RequestID == nil || sent.Request == nil {
		return 0, nil
	}

	statements := make([]string, 0, 1)

	switch sent.Request.Type {
	case ExecuteRequest:
		if sent.Request.Stmt != nil {
			statements = append(statements, sent.Request.Stmt.SQL)
		}
	case BatchRequest:
		if sent.Request.Batch != nil {
			for _, step := range sent.Request.Batch.Steps {
				if step.Stmt != nil {
					statements = append(statements, step.Stmt.SQL)
				}
			}
		}
	}

	return *sent.RequestID, statements
}

func StatementsHeld(frame []byte) (int64, []Statement, bool) {
	held := heldFrame{}

	if decodeError := json.Unmarshal(frame, &held); decodeError != nil {
		return 0, nil, false
	}

	if held.RequestID == nil {
		return 0, nil, false
	}

	if held.Type == ResponseErrorFrame {
		return *held.RequestID, []Statement{{Failed: true}}, true
	}

	if held.Type != ResponseOkFrame || held.Response == nil {
		return 0, nil, false
	}

	measured := make([]Statement, 0, 1)

	if held.Response.Result != nil {
		measured = append(measured, fromResult(held.Response.Result))
	}

	for _, step := range held.Response.StepResults {
		if step != nil {
			measured = append(measured, fromResult(step))
		}
	}

	return *held.RequestID, measured, true
}

func fromResult(result *heldResult2) Statement {
	return Statement{
		DurationMs:   result.DurationMs,
		RowsRead:     result.RowsRead,
		RowsWritten:  result.RowsWritten,
		RowsReturned: int64(len(result.Rows)),
	}
}
