package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"path/filepath"
	"sqldash/certificates"
	"sqldash/config"
	"sqldash/database"
	"sqldash/models"
	"sqldash/pgcluster/control"
	"sqldash/pggateway"
	"sqldash/services/pgaccess"
	primarycredentials "sqldash/services/postgresconnections/primary"
	"sqldash/utils/logger"
	"syscall"
	"time"
)

func runPostgresGateway() {
	parent, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if !config.Postgres.Enabled || !config.Postgres.GatewayEnabled {
		return
	}
	if _, fileError := os.Lstat(postgresUnsafePath()); fileError == nil || !errors.Is(fileError, os.ErrNotExist) {
		logger.Fatalf(LogPrefix, "Native PostgreSQL ingress is blocked pending enforcement recovery")
	}
	if readyError := waitForPostgres(parent); readyError != nil {
		return
	}
	var records []models.Database
	if findError := database.DB.WithContext(parent).Where("engine = ? AND state = ?", models.EnginePostgres, models.DatabaseReady).Find(&records).Error; findError != nil {
		logger.Fatalf(LogPrefix, "Could not verify PostgreSQL registrations")
	}
	for index := range records {
		if reconcileError := control.Provision(parent, &records[index]); reconcileError != nil {
			logger.Fatalf(LogPrefix, "Could not reconcile PostgreSQL database privileges")
		}
		if reconcileError := primarycredentials.Reconcile(parent, &records[index]); reconcileError != nil {
			logger.Fatalf(LogPrefix, "Could not reconcile PostgreSQL application privileges")
		}
	}
	tlsOptions := pggateway.TLSOptions{ConfigurationDirectory: config.PostgresTLSPath(), Host: config.PostgresHost(), CertificatePath: config.Postgres.CertificatePath, KeyPath: config.Postgres.PrivateKeyPath}
	if certificates.Wanted() {
		tlsOptions.AutomaticIssuer = certificates.NativeSettings
	}
	material, configureError := pggateway.ConfigureTLS(parent, tlsOptions)
	if configureError != nil {
		logger.Fatalf(LogPrefix, "Could not configure native PostgreSQL TLS: %v", configureError)
	}
	options := pggateway.Options{ListenAddress: config.Postgres.GatewayAddress, BackendAddress: config.Postgres.Address, TLSConfig: material.Config, AllowPlaintext: true, MaximumSessions: config.PostgresSessionLimit(), Authorize: pgaccess.Authorize, Validate: pgaccess.Validate, Terminate: pgaccess.Terminate, Unsafe: blockPostgres}
	if runError := pggateway.Run(parent, options); runError != nil && !errors.Is(runError, context.Canceled) {
		logger.Fatalf(LogPrefix, "Native PostgreSQL gateway stopped: %v", runError)
	}
}

func waitForPostgres(parent context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		probe, cancel := context.WithTimeout(parent, 3*time.Second)
		readyError := control.Ready(probe)
		cancel()
		if readyError == nil {
			return nil
		}
		select {
		case <-parent.Done():
			return parent.Err()
		case <-ticker.C:
		}
	}
}

func postgresUnsafePath() string {
	return filepath.Join(config.PostgresRuntimePath(), "enforcement-blocked")
}

func blockPostgres(failure error) {
	logger.Errorf(LogPrefix, "PostgreSQL session enforcement could not be confirmed; native ingress stopped")
	file, openError := os.OpenFile(postgresUnsafePath(), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if openError != nil {
		if !errors.Is(openError, os.ErrExist) {
			logger.Errorf(LogPrefix, "Could not signal the PostgreSQL supervisor: %v", openError)
		}
		return
	}
	_, _ = file.WriteString(time.Now().UTC().Format(time.RFC3339))
	_ = file.Sync()
	_ = file.Close()
}
