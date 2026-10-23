package runtime

import (
	"net"
	"path/filepath"
	"strconv"
	"strings"

	"sqldash/config"
	"sqldash/supervisor"
)

func writeConfiguration(credentials *supervisor.Credentials) error {
	host, port, err := net.SplitHostPort(config.Postgres.Address)
	if err != nil {
		return err
	}
	configuration := strings.Join([]string{
		"listen_addresses = " + configurationString(host),
		"port = " + port,
		"data_directory = " + configurationString(config.PostgresDataPath()),
		"hba_file = " + configurationString(filepath.Join(config.PostgresConfigPath(), hostAuthenticationFileName)),
		"unix_socket_directories = " + configurationString(config.PostgresRuntimePath()),
		"unix_socket_permissions = 0700",
		"password_encryption = 'scram-sha-256'",
		"max_connections = " + strconv.Itoa(config.Postgres.MaximumConnections),
		"superuser_reserved_connections = 5",
		"shared_preload_libraries = 'pg_stat_statements'",
		"pg_stat_statements.track = 'all'",
		"ssl = off",
		"logging_collector = off",
		"log_statement = 'none'",
		"log_min_error_statement = 'panic'",
		"log_parameter_max_length = 0",
		"log_parameter_max_length_on_error = 0",
		"log_connections = off",
		"log_disconnections = off",
		"log_line_prefix = '%m [%p] '",
		"",
	}, "\n")
	authentication := strings.Join([]string{
		"local all all scram-sha-256",
		"host all all 127.0.0.1/32 scram-sha-256",
		"host all all ::1/128 scram-sha-256",
		"local replication all reject",
		"host replication all 0.0.0.0/0 reject",
		"host replication all ::0/0 reject",
		"host all all 0.0.0.0/0 reject",
		"host all all ::0/0 reject",
		"",
	}, "\n")
	if err = writePrivateFile(filepath.Join(config.PostgresConfigPath(), hostAuthenticationFileName), []byte(authentication), credentials); err != nil {
		return err
	}
	return writePrivateFile(filepath.Join(config.PostgresConfigPath(), configurationFileName), []byte(configuration), credentials)
}

func configurationString(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
