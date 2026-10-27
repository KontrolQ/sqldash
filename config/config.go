package config

import (
	"sqldash/utils/logger"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

var (
	Server       serverSettings
	Data         dataSettings
	Transfer     transferSettings
	History      historySettings
	Query        querySettings
	Sqld         sqldSettings
	Postgres     postgresSettings
	Redis        redisSettings
	Certificates certificateSettings
	Hosting      hostingSettings
	Session      sessionSettings
	Access       accessSettings
)

func init() {
	_ = godotenv.Load(EnvironmentFileName)

	for _, target := range []any{&Server, &Data, &Transfer, &History, &Query, &Sqld, &Postgres, &Redis, &Certificates, &Hosting, &Session, &Access} {
		if parseError := env.ParseWithOptions(target, env.Options{Prefix: EnvironmentPrefix}); parseError != nil {
			logger.Fatalf(LogPrefix, ParseFailedLog, parseError)
		}
	}

	logger.SetDebug(Server.Debug)

	if verifyError := verifyConfig(); verifyError != nil {
		logger.Fatalf(LogPrefix, VerifyFailedLog, verifyError)
	}

	logger.Successf(LogPrefix, LoadedLog, Server.Domain)
}
