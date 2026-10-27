package config

import (
	"errors"
	"math"
	"net"
	"strconv"
	"strings"
	"time"
)

func VerifyRedis() error {
	if !Redis.Enabled {
		return nil
	}
	if Redis.BinaryPath == "" || Redis.ModulesDirectory == "" {
		return errors.New("Redis runtime paths must be configured")
	}
	if Redis.MemoryLimit < 0 || Redis.MemoryLimit > 0 && Redis.MemoryLimit < 1024*1024 || Redis.MemoryBudget < 0 || Redis.MemoryBudget > 0 && Redis.MemoryBudget < Redis.MemoryLimit || Redis.MaximumInstances < 0 {
		return errors.New("Redis memory and instance limits are invalid")
	}
	if Redis.MaximumConnections < 0 || Redis.MaximumConnections > math.MaxInt-RedisControlConnections || Redis.MaximumValueBytes < 1 || Redis.MaximumValueBytes > 512*1024*1024 {
		return errors.New("Redis connection and value limits are invalid")
	}
	if Redis.MaximumScanBatch < 1 || Redis.MaximumScanBatch > 10000 || Redis.MaximumStreamBuffer < 1 || Redis.MaximumStreamBuffer > 100000 {
		return errors.New("Redis scan and stream limits are invalid")
	}
	if Redis.StartupTimeout <= 0 || Redis.ShutdownTimeout <= 0 || Redis.CommandTimeout <= 0 || Redis.RevalidationInterval < 100*time.Millisecond || Redis.RevalidationInterval > 4*time.Second {
		return errors.New("Redis timeouts are invalid")
	}
	if Redis.PublicPort < 1 || Redis.PublicPort > 65535 || strings.ContainsAny(Redis.PublicHost, "/@\r\n\t ") {
		return errors.New("Redis public address is invalid")
	}
	if Redis.GatewayEnabled {
		_, portText, err := net.SplitHostPort(Redis.GatewayAddress)
		port, parseError := strconv.Atoi(portText)
		if err != nil || parseError != nil || port < 1 || port > 65535 {
			return errors.New("Redis gateway address is invalid")
		}
	}
	if (Redis.CertificatePath == "") != (Redis.PrivateKeyPath == "") {
		return errors.New("Redis certificate and private key must be configured together")
	}
	return nil
}
