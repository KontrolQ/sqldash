package pggateway

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

func (self Options) normalized() (Options, error) {
	integerDefaults := []struct {
		value    *int
		fallback int
	}{
		{&self.MaximumConnections, DefaultMaximumConnections},
		{&self.MaximumSessions, DefaultMaximumSessions},
		{&self.MaximumHandshakes, DefaultMaximumHandshakes},
		{&self.MaximumStartupBytes, DefaultMaximumStartupBytes},
		{&self.MaximumHandshakeFrameBytes, DefaultMaximumHandshakeFrameBytes},
		{&self.MaximumHandshakeBytes, DefaultMaximumHandshakeBytes},
		{&self.MaximumCancelRequests, DefaultMaximumCancelRequests},
		{&self.MaximumSessionCancels, DefaultMaximumSessionCancels},
	}
	for _, setting := range integerDefaults {
		if *setting.value < 0 {
			return self, ErrInvalidOptions
		}
		if *setting.value == 0 {
			*setting.value = setting.fallback
		}
	}
	durationDefaults := []struct {
		value    *time.Duration
		fallback time.Duration
	}{
		{&self.HandshakeTimeout, DefaultHandshakeTimeout},
		{&self.TLSHandshakeTimeout, DefaultTLSHandshakeTimeout},
		{&self.BackendConnectTimeout, DefaultBackendConnectTimeout},
		{&self.CancelTimeout, DefaultCancelTimeout},
		{&self.ValidationInterval, DefaultValidationInterval},
		{&self.ValidationTimeout, DefaultValidationTimeout},
		{&self.TerminationTimeout, DefaultTerminationTimeout},
	}
	for _, setting := range durationDefaults {
		if *setting.value < 0 {
			return self, ErrInvalidOptions
		}
		if *setting.value == 0 {
			*setting.value = setting.fallback
		}
	}
	if self.Authorize == nil || self.Validate == nil || self.Terminate == nil || self.Unsafe == nil || self.TLSConfig == nil {
		return self, ErrInvalidOptions
	}
	if self.ListenAddress == "" || self.BackendAddress == "" || self.ValidationInterval > MaximumPolicyDelay-self.ValidationTimeout {
		return self, ErrInvalidOptions
	}
	if self.MaximumSessions > self.MaximumConnections || self.MaximumHandshakes > self.MaximumConnections || self.MaximumStartupBytes < maximumCancelPacketBytes || self.MaximumStartupBytes > 1024*1024 {
		return self, ErrInvalidOptions
	}
	if self.MaximumHandshakeFrameBytes < 512 || self.MaximumHandshakeFrameBytes > 1024*1024 || self.MaximumHandshakeBytes < self.MaximumHandshakeFrameBytes || self.MaximumHandshakeBytes > 16*1024*1024 {
		return self, ErrInvalidOptions
	}
	if self.ValidationTimeout >= MaximumPolicyDelay || self.ValidationInterval <= 0 {
		return self, ErrInvalidOptions
	}
	if _, _, addressError := net.SplitHostPort(self.ListenAddress); addressError != nil {
		return self, fmt.Errorf("%w: listen address", ErrInvalidOptions)
	}
	if _, _, addressError := net.SplitHostPort(self.BackendAddress); addressError != nil {
		return self, fmt.Errorf("%w: backend address", ErrInvalidOptions)
	}
	self.TLSConfig = self.TLSConfig.Clone()
	if self.TLSConfig.GetConfigForClient != nil || (len(self.TLSConfig.Certificates) == 0 && self.TLSConfig.GetCertificate == nil) || (self.TLSConfig.MaxVersion != 0 && self.TLSConfig.MaxVersion < tls.VersionTLS12) {
		return self, ErrInvalidTLS
	}
	if self.TLSConfig.MinVersion < tls.VersionTLS12 {
		self.TLSConfig.MinVersion = tls.VersionTLS12
	}
	if self.TLSConfig.MaxVersion != 0 && self.TLSConfig.MaxVersion < self.TLSConfig.MinVersion {
		return self, ErrInvalidTLS
	}
	self.TLSConfig.NextProtos = []string{postgresqlApplicationProtocol}
	if self.Registry == nil {
		self.Registry = &Registry{}
	}
	return self, nil
}
