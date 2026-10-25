package main

import (
	"context"
	"errors"
	"os/signal"
	"syscall"
	"time"

	"sqldash/certificates"
	"sqldash/config"
	"sqldash/nativetls"
	"sqldash/redisgateway"
	"sqldash/redishost"
	"sqldash/services/redisaccess"
	"sqldash/utils/logger"
)

func runRedisGateway() {
	if !config.Redis.Enabled || !config.Redis.GatewayEnabled {
		return
	}
	parent, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		probe, cancel := context.WithTimeout(parent, 3*time.Second)
		err := redishost.Ready(probe)
		cancel()
		if err == nil {
			break
		}
		select {
		case <-parent.Done():
			return
		case <-ticker.C:
		}
	}
	options := nativetls.Options{ConfigurationDirectory: config.RedisTLSPath(), Host: config.RedisHost(), CertificatePath: config.Redis.CertificatePath, KeyPath: config.Redis.PrivateKeyPath, AuthorityName: "sqldash Redis certificate authority", LeafName: "sqldash Redis"}
	if certificates.Wanted() {
		options.AutomaticIssuer = certificates.NativeSettings
	}
	material, err := nativetls.Configure(parent, options)
	if err != nil {
		logger.Fatalf(LogPrefix, "Could not configure native Redis TLS: %v", err)
	}
	gateway := redisgateway.Options{ListenAddress: config.Redis.GatewayAddress, TLSConfig: material.Config, AllowPlaintext: config.Redis.AllowPlaintext, MaximumConnections: config.Redis.MaximumConnections, MaximumSessionsPerDatabase: config.Redis.MaximumConnections, ValidationInterval: config.Redis.RevalidationInterval, Authorize: redisaccess.Authorize, Validate: redisaccess.Validate, Dial: redisaccess.Dial, Terminate: redisaccess.Terminate, Unsafe: redisaccess.Unsafe}
	if err := redisgateway.Run(parent, gateway); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatalf(LogPrefix, "Native Redis ingress stopped: %v", err)
	}
}
