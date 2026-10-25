package redisgateway

import "crypto/tls"

func normalize(options Options) (Options, error) {
	if options.ListenAddress == "" {
		options.ListenAddress = ":6379"
	}
	if options.MaximumConnections == 0 {
		options.MaximumConnections = DefaultMaximumConnections
	}
	if options.MaximumSessionsPerDatabase == 0 {
		options.MaximumSessionsPerDatabase = DefaultMaximumSessionsPerDatabase
	}
	if options.MaximumHandshakes == 0 {
		options.MaximumHandshakes = DefaultMaximumHandshakes
	}
	if options.HandshakeTimeout == 0 {
		options.HandshakeTimeout = DefaultHandshakeTimeout
	}
	if options.ValidationInterval == 0 {
		options.ValidationInterval = DefaultValidationInterval
	}
	if options.ValidationTimeout == 0 {
		options.ValidationTimeout = DefaultValidationTimeout
	}
	if options.TerminationTimeout == 0 {
		options.TerminationTimeout = DefaultTerminationTimeout
	}
	if options.MaximumConnections < 1 || options.MaximumSessionsPerDatabase < 1 || options.MaximumHandshakes < 1 || options.HandshakeTimeout <= 0 || options.ValidationInterval <= 0 || options.ValidationTimeout <= 0 || options.TerminationTimeout <= 0 {
		return options, InvalidOptions
	}
	if options.ValidationInterval+options.ValidationTimeout > MaximumPolicyDelay || !options.AllowPlaintext && options.TLSConfig == nil {
		return options, InvalidOptions
	}
	if options.Authorize == nil || options.Validate == nil || options.Dial == nil || options.Terminate == nil || options.Unsafe == nil {
		return options, InvalidOptions
	}
	if options.TLSConfig != nil {
		options.TLSConfig = options.TLSConfig.Clone()
		if options.TLSConfig.MinVersion < tls.VersionTLS12 {
			options.TLSConfig.MinVersion = tls.VersionTLS12
		}
		if options.TLSConfig.MaxVersion != 0 && options.TLSConfig.MaxVersion < options.TLSConfig.MinVersion || options.TLSConfig.GetConfigForClient != nil {
			return options, InvalidOptions
		}
	}
	return options, nil
}
