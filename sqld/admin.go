package sqld

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"sqldash/config"
	"sqldash/transferpolicy"
)

var (
	client         = &http.Client{Timeout: AdminTimeout}
	queryClient    = &http.Client{Timeout: config.Sqld.QueryTimeout}
	transferClient = &http.Client{}
	adminAt        string
)

func init() {
	adminAt = strings.TrimRight(config.SqldAdminURL(), "/")
}

func Create(requestContext context.Context, database string) error {
	return send(requestContext, http.MethodPost, namespacePath(database, CreateAction), map[string]any{}, nil)
}

func Delete(requestContext context.Context, database string) error {
	return send(requestContext, http.MethodDelete, namespacePath(database), nil, nil)
}

func Fork(requestContext context.Context, from string, to string, at *time.Time) error {
	path := namespacePath(from, ForkAction, to)

	if at != nil {
		path += "?" + TimestampParameter + "=" + at.UTC().Format(time.RFC3339)
	}

	work, cancel := transferpolicy.Context(requestContext, 0)
	defer cancel()
	return sendWithClient(work, transferClient, http.MethodPost, path, nil, nil)
}

func Statistics(requestContext context.Context, database string) (*StatisticsRecord, error) {
	held := &StatisticsRecord{}

	if sendError := send(requestContext, http.MethodGet, namespacePath(database, StatsAction), nil, held); sendError != nil {
		return nil, sendError
	}

	return held, nil
}

func Configuration(requestContext context.Context, database string) (*ConfigurationRecord, error) {
	held := &ConfigurationRecord{}

	if sendError := send(requestContext, http.MethodGet, namespacePath(database, ConfigAction), nil, held); sendError != nil {
		return nil, sendError
	}

	return held, nil
}

func SetConfiguration(requestContext context.Context, database string, wanted ConfigurationRecord) error {
	return send(requestContext, http.MethodPost, namespacePath(database, ConfigAction), wanted, nil)
}

func Ready(requestContext context.Context) bool {
	request, buildError := http.NewRequestWithContext(requestContext, http.MethodGet, adminAt+"/", nil)
	if buildError != nil {
		return false
	}

	answer, sendError := client.Do(request)
	if sendError != nil {
		return false
	}

	defer answer.Body.Close()
	io.Copy(io.Discard, answer.Body)

	return answer.StatusCode < http.StatusInternalServerError
}

func send(requestContext context.Context, method string, path string, body any, into any) error {
	return sendWithClient(requestContext, client, method, path, body, into)
}

func sendWithClient(requestContext context.Context, sender *http.Client, method string, path string, body any, into any) error {
	var payload io.Reader

	if body != nil {
		encoded, encodeError := json.Marshal(body)
		if encodeError != nil {
			return errors.New(EncodeFailed)
		}

		payload = bytes.NewReader(encoded)
	}

	request, buildError := http.NewRequestWithContext(requestContext, method, adminAt+path, payload)
	if buildError != nil {
		return errors.New(BuildFailed)
	}

	if body != nil {
		request.Header.Set(ContentTypeHeader, ApplicationJSON)
	}

	answer, sendError := sender.Do(request)
	if sendError != nil {
		return errors.New(SendFailed)
	}

	defer answer.Body.Close()

	held, readError := io.ReadAll(io.LimitReader(answer.Body, MaximumAdminResponse))
	if readError != nil {
		return errors.New(ReadFailed)
	}

	if answer.StatusCode >= http.StatusBadRequest {
		return &AdminError{Status: answer.StatusCode, Message: strings.TrimSpace(string(held))}
	}

	if into == nil || len(bytes.TrimSpace(held)) == 0 {
		return nil
	}

	if decodeError := json.Unmarshal(held, into); decodeError != nil {
		return errors.New(DecodeFailed)
	}

	return nil
}
