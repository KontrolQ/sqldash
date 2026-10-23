package main

import (
	standardContext "context"
	"github.com/gofiber/fiber/v3"
	"io"
	"net"
	"net/http"
	"os"
	"sqldash/config"
	"sqldash/database"
	"sqldash/pgcluster/control"
	"sqldash/sqld"
	"sqldash/utils/collections"
	"time"
)

func readyHealth(context fiber.Ctx) error {
	parent, cancel := readyContext(context.Context())
	defer cancel()
	stateReady := database.Verify() == nil
	libSQLReady := sqld.Ready(parent)
	postgresReady := !config.Postgres.Enabled
	gatewayReady := !config.Postgres.Enabled || !config.Postgres.GatewayEnabled
	if config.Postgres.Enabled {
		postgresReady = control.Ready(parent) == nil
		if config.Postgres.GatewayEnabled {
			connection, connectError := (&net.Dialer{}).DialContext(parent, "tcp", loopbackAddress(config.Postgres.GatewayAddress))
			gatewayReady = connectError == nil
			if connection != nil {
				_ = connection.Close()
			}
		}
	}
	status := fiber.StatusOK
	if !stateReady || !libSQLReady || !postgresReady || !gatewayReady {
		status = fiber.StatusServiceUnavailable
	}
	return context.Status(status).JSON(collections.Record[string, any]{"ready": status == fiber.StatusOK, "state": stateReady, "libsql": libSQLReady, "postgres": postgresReady, "native": gatewayReady})
}

func readyContext(parent standardContext.Context) (standardContext.Context, standardContext.CancelFunc) {
	return standardContext.WithTimeout(parent, 3*time.Second)
}

func loopbackAddress(address string) string {
	host, port, splitError := net.SplitHostPort(address)
	if splitError != nil {
		return address
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port)
}

func runHealth() {
	client := http.Client{Timeout: 5 * time.Second}
	response, requestError := client.Get("http://" + loopbackAddress(config.Server.WebAddress) + "/health/ready")
	if requestError != nil {
		os.Exit(1)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	if response.StatusCode != http.StatusOK {
		os.Exit(1)
	}
}
