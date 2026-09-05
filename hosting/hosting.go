package hosting

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"sqldash/config"
	"sqldash/utils/logger"
)

var client = &http.Client{Timeout: RequestTimeout}

func Wanted() bool {
	return config.Hosting.Address != "" && config.Hosting.Password != "" && config.Hosting.App != ""
}

func Publish(requestContext context.Context, databaseName string) {
	if !Wanted() {
		return
	}

	host := databaseName + "." + config.Server.Domain

	token, loginError := signIn(requestContext)
	if loginError != nil {
		logger.Errorf(LogPrefix, SignInFailedLog, loginError)
		return
	}

	if addError := ask(requestContext, token, DomainPath, map[string]any{
		AppField: config.Hosting.App, DomainField: host,
	}); addError != nil {
		logger.Warnf(LogPrefix, DomainFailedLog, host, addError)
		return
	}

	logger.Successf(LogPrefix, DomainAddedLog, host)

	if sslError := ask(requestContext, token, SecurePath, map[string]any{
		AppField: config.Hosting.App, DomainField: host,
	}); sslError != nil {
		logger.Warnf(LogPrefix, SecureFailedLog, host, sslError)
		return
	}

	logger.Successf(LogPrefix, SecuredLog, host)
}

func Withdraw(requestContext context.Context, databaseName string) {
	if !Wanted() {
		return
	}

	host := databaseName + "." + config.Server.Domain

	token, loginError := signIn(requestContext)
	if loginError != nil {
		logger.Errorf(LogPrefix, SignInFailedLog, loginError)
		return
	}

	if removeError := ask(requestContext, token, RemovePath, map[string]any{
		AppField: config.Hosting.App, DomainField: host,
	}); removeError != nil {
		logger.Warnf(LogPrefix, RemoveFailedLog, host, removeError)
		return
	}

	logger.Successf(LogPrefix, DomainRemovedLog, host)
}

func signIn(requestContext context.Context) (string, error) {
	held := struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}{}

	if sendError := send(requestContext, "", LoginPath, map[string]any{PasswordField: config.Hosting.Password}, &held); sendError != nil {
		return "", sendError
	}

	return held.Data.Token, nil
}

func ask(requestContext context.Context, token string, path string, body map[string]any) error {
	return send(requestContext, token, path, body, nil)
}

func send(requestContext context.Context, token string, path string, body map[string]any, into any) error {
	encoded, encodeError := json.Marshal(body)
	if encodeError != nil {
		return encodeError
	}

	address := strings.TrimRight(config.Hosting.Address, "/") + APIPrefix + path

	request, buildError := http.NewRequestWithContext(requestContext, http.MethodPost, address, bytes.NewReader(encoded))
	if buildError != nil {
		return buildError
	}

	request.Header.Set(ContentTypeHeader, ApplicationJSON)
	request.Header.Set(NamespaceHeader, CaptainNamespace)

	if token != "" {
		request.Header.Set(AuthHeader, token)
	}

	answer, sendError := client.Do(request)
	if sendError != nil {
		return sendError
	}

	defer answer.Body.Close()

	raw, readError := io.ReadAll(io.LimitReader(answer.Body, MaximumResponse))
	if readError != nil {
		return readError
	}

	outcome := struct {
		Status      int    `json:"status"`
		Description string `json:"description"`
	}{}

	if decodeError := json.Unmarshal(raw, &outcome); decodeError != nil {
		return decodeError
	}

	if outcome.Status != Accepted && outcome.Status != AlsoAccepted {
		return &refusal{status: outcome.Status, description: outcome.Description}
	}

	if into == nil {
		return nil
	}

	return json.Unmarshal(raw, into)
}

type refusal struct {
	status      int
	description string
}

func (self *refusal) Error() string {
	return self.description
}

func init() {
	client.Timeout = RequestTimeout + time.Second
}
