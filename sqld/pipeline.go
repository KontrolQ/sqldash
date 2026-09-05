package sqld

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"sqldash/config"
)

type Column struct {
	Name string
	Kind string
}

type Result struct {
	Columns      []Column
	Rows         [][]any
	Affected     int64
	LastInsertID *int64
	RowsRead     int64
	RowsWritten  int64
	DurationMs   float64
}

type Statement struct {
	SQL       string
	Arguments []any
}

func Query(requestContext context.Context, database string, sql string, arguments ...any) (*Result, error) {
	held, runError := Run(requestContext, database, Statement{SQL: sql, Arguments: arguments})
	if runError != nil {
		return nil, runError
	}

	return held[0], nil
}

func Run(requestContext context.Context, database string, statements ...Statement) ([]*Result, error) {
	asked := make([]any, 0, len(statements)+1)

	for _, statement := range statements {
		values := make([]any, 0, len(statement.Arguments))

		for _, argument := range statement.Arguments {
			values = append(values, sentValue(argument))
		}

		asked = append(asked, map[string]any{
			"type": ExecuteRequest,
			"stmt": map[string]any{"sql": statement.SQL, "args": values},
		})
	}

	asked = append(asked, map[string]any{"type": CloseRequest})

	encoded, encodeError := json.Marshal(map[string]any{"requests": asked})
	if encodeError != nil {
		return nil, errors.New(EncodeFailed)
	}

	request, buildError := http.NewRequestWithContext(
		requestContext, http.MethodPost, config.SqldURL()+PipelinePath, bytes.NewReader(encoded),
	)

	if buildError != nil {
		return nil, errors.New(BuildFailed)
	}

	request.Header.Set(ContentTypeHeader, ApplicationJSON)
	request.Header.Set(NamespaceHeader, database)

	answer, sendError := client.Do(request)
	if sendError != nil {
		return nil, errors.New(SendFailed)
	}

	defer answer.Body.Close()

	body, readError := io.ReadAll(io.LimitReader(answer.Body, MaximumQueryResponse))
	if readError != nil {
		return nil, errors.New(ReadFailed)
	}

	if answer.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf(RefusedFormat, answer.Status, strings.TrimSpace(string(body)))
	}

	return readResults(body, len(statements))
}

func readResults(body []byte, wanted int) ([]*Result, error) {
	held := struct {
		Results []struct {
			Type  string `json:"type"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
			Response *struct {
				Result *struct {
					Columns []struct {
						Name     string  `json:"name"`
						DeclType *string `json:"decltype"`
					} `json:"cols"`
					Rows         [][]rawValue `json:"rows"`
					Affected     int64        `json:"affected_row_count"`
					LastInsertID *string      `json:"last_insert_rowid"`
					RowsRead     int64        `json:"rows_read"`
					RowsWritten  int64        `json:"rows_written"`
					DurationMs   float64      `json:"query_duration_ms"`
				} `json:"result"`
			} `json:"response"`
		} `json:"results"`
	}{}

	if decodeError := json.Unmarshal(body, &held); decodeError != nil {
		return nil, errors.New(DecodeFailed)
	}

	results := make([]*Result, 0, wanted)

	for index, one := range held.Results {
		if index >= wanted {
			break
		}

		if one.Type != OkResult {
			message := QueryRefused
			if one.Error != nil {
				message = one.Error.Message
			}

			return nil, errors.New(message)
		}

		result := &Result{}

		if one.Response != nil && one.Response.Result != nil {
			source := one.Response.Result

			for _, column := range source.Columns {
				kind := ""
				if column.DeclType != nil {
					kind = *column.DeclType
				}

				result.Columns = append(result.Columns, Column{Name: column.Name, Kind: kind})
			}

			for _, row := range source.Rows {
				held := make([]any, 0, len(row))

				for _, cell := range row {
					held = append(held, cell.plain())
				}

				result.Rows = append(result.Rows, held)
			}

			result.Affected = source.Affected
			result.RowsRead = source.RowsRead
			result.RowsWritten = source.RowsWritten
			result.DurationMs = source.DurationMs

			if source.LastInsertID != nil {
				if parsed, parseError := strconv.ParseInt(*source.LastInsertID, 10, 64); parseError == nil {
					result.LastInsertID = &parsed
				}
			}
		}

		results = append(results, result)
	}

	if len(results) < wanted {
		return nil, errors.New(QueryRefused)
	}

	return results, nil
}

type rawValue struct {
	Type   string          `json:"type"`
	Value  json.RawMessage `json:"value"`
	Base64 string          `json:"base64"`
}

func (self rawValue) plain() any {
	switch self.Type {
	case NullValue:
		return nil
	case IntegerValue:
		held := ""
		json.Unmarshal(self.Value, &held)

		if parsed, parseError := strconv.ParseInt(held, 10, 64); parseError == nil {
			return parsed
		}

		return held
	case FloatValue:
		var held float64
		json.Unmarshal(self.Value, &held)

		return held
	case BlobValue:
		decoded, decodeError := base64.StdEncoding.DecodeString(self.Base64)
		if decodeError != nil {
			return nil
		}

		return decoded
	default:
		held := ""
		json.Unmarshal(self.Value, &held)

		return held
	}
}

func sentValue(argument any) map[string]any {
	switch held := argument.(type) {
	case nil:
		return map[string]any{"type": NullValue}
	case int:
		return map[string]any{"type": IntegerValue, "value": strconv.Itoa(held)}
	case int64:
		return map[string]any{"type": IntegerValue, "value": strconv.FormatInt(held, 10)}
	case float64:
		return map[string]any{"type": FloatValue, "value": held}
	case bool:
		value := "0"
		if held {
			value = "1"
		}

		return map[string]any{"type": IntegerValue, "value": value}
	case []byte:
		return map[string]any{"type": BlobValue, "base64": base64.StdEncoding.EncodeToString(held)}
	default:
		return map[string]any{"type": TextValue, "value": fmt.Sprint(held)}
	}
}
