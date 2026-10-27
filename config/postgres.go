package config

import (
	"errors"
	"math"
	"net"
	"strconv"
	"strings"
)

func PostgresSessionLimit() int {
	if Postgres.MaximumConnections == 0 {
		return 0
	}
	return Postgres.MaximumConnections - PostgresReservedConnections
}

func VerifyPostgres() error {
	if !Postgres.Enabled {
		return nil
	}
	privateHost, privatePort, addressError := net.SplitHostPort(Postgres.Address)
	if addressError != nil || !validPostgresPort(privatePort) || privateHost == "" {
		return errors.New("PostgreSQL needs a valid private host and port")
	}
	if Postgres.Managed {
		privateAddress := net.ParseIP(privateHost)
		if privateAddress == nil || !privateAddress.IsLoopback() {
			return errors.New("managed PostgreSQL must listen on a literal loopback address")
		}
		if Postgres.BinaryDirectory == "" {
			return errors.New("the PostgreSQL binary directory is required")
		}
	}
	if Postgres.GatewayEnabled {
		_, gatewayPort, gatewayError := net.SplitHostPort(Postgres.GatewayAddress)
		if gatewayError != nil || !validPostgresPort(gatewayPort) || Postgres.GatewayAddress == Postgres.Address {
			return errors.New("PostgreSQL needs a distinct valid gateway address")
		}
	}
	if Postgres.PublicPort < 1 || Postgres.PublicPort > 65535 {
		return errors.New("the PostgreSQL public port is invalid")
	}
	if PostgresHost() == "" || strings.ContainsAny(PostgresHost(), " \t\r\n/\\'\"") {
		return errors.New("the PostgreSQL public hostname is invalid")
	}
	if !validPostgresUsername(Postgres.AdminUsername) {
		return errors.New("the PostgreSQL administrator username must be a lowercase identifier of at most 63 bytes")
	}
	if Postgres.MaximumConnections < 0 || Postgres.MaximumConnections > 0 && Postgres.MaximumConnections <= PostgresReservedConnections || Postgres.MaximumConnections > math.MaxInt32 {
		return errors.New("PostgreSQL maximum connections must be zero for the engine default, or leave room for reserved administrative connections")
	}
	if (Postgres.CertificatePath == "") != (Postgres.PrivateKeyPath == "") {
		return errors.New("PostgreSQL TLS needs both a certificate and a private key")
	}
	if Postgres.InitializationTimeout <= 0 || Postgres.StartupTimeout <= 0 || Postgres.ShutdownTimeout <= 0 {
		return errors.New("PostgreSQL timeouts must be positive")
	}
	return nil
}

func validPostgresPort(value string) bool {
	port, err := strconv.Atoi(value)
	return err == nil && port > 0 && port <= 65535
}

func validPostgresUsername(value string) bool {
	if len(value) == 0 || len(value) > 63 {
		return false
	}
	for index, character := range value {
		if character >= 'a' && character <= 'z' || character == '_' || index > 0 && character >= '0' && character <= '9' {
			continue
		}
		return false
	}
	return true
}
