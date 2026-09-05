package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"sqldash/config"
	"sqldash/utils/logger"
)

var (
	toDashboard *httputil.ReverseProxy
	toDatabase  *httputil.ReverseProxy
)

func init() {
	toDashboard = httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: config.Server.WebAddress})
	toDatabase = httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: config.Sqld.Address})

	toDashboard.ErrorHandler = refuseWith(DashboardDown)
	toDatabase.ErrorHandler = refuseWith(DatabaseDown)
}

func Serve() {
	server := &http.Server{
		Addr:              config.HTTPAddress(),
		Handler:           http.HandlerFunc(handle),
		ReadHeaderTimeout: ReadHeaderTimeout,
		IdleTimeout:       IdleTimeout,
	}

	logger.Successf(LogPrefix, StartedLog, server.Addr)

	if listenError := server.ListenAndServe(); listenError != nil {
		logger.Fatalf(LogPrefix, ListenFailedLog, server.Addr, listenError)
	}
}

func handle(writer http.ResponseWriter, request *http.Request) {
	database := databaseFor(incomingFrom(request))

	if database == "" {
		toDashboard.ServeHTTP(writer, request)
		return
	}

	if !known(database) {
		logger.Warnf(LogPrefix, UnknownDatabaseLog, database)
		http.Error(writer, UnknownDatabase, http.StatusNotFound)
		return
	}

	if status, refusal := permitted(request, database); status != 0 {
		http.Error(writer, refusal, status)
		return
	}

	request.Header.Set(NamespaceHeader, database)

	if carriesStatements(request) {
		measure(writer, request, database)
		return
	}

	toDatabase.ServeHTTP(writer, request)
}

func carriesStatements(request *http.Request) bool {
	if isUpgrade(request) {
		return false
	}

	switch request.URL.Path {
	case PipelinePathV2, PipelinePathV3:
		return request.Method == http.MethodPost
	default:
		return false
	}
}

func isUpgrade(request *http.Request) bool {
	return strings.EqualFold(request.Header.Get(UpgradeHeader), WebsocketUpgrade)
}

func refuseWith(message string) func(http.ResponseWriter, *http.Request, error) {
	return func(writer http.ResponseWriter, _ *http.Request, forwardError error) {
		logger.Errorf(LogPrefix, ForwardFailedLog, forwardError)
		http.Error(writer, message, http.StatusBadGateway)
	}
}
