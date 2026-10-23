package pgtransfer

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"strconv"
	"strings"

	"sqldash/config"

	"github.com/jackc/pgx/v5"
)

type remoteEndpoint struct {
	address         string
	port            string
	sslMode         string
	roots           *x509.CertPool
	certificatePath string
	passwordPath    string
	serverVersion   int
}

func (self *job) prepareRemote(source RemoteSource) (*remoteEndpoint, error) {
	if authorizeError := self.authorize(); authorizeError != nil {
		return nil, authorizeError
	}
	addresses := []net.IP{}
	if address := net.ParseIP(source.Host); address != nil {
		addresses = append(addresses, address)
	} else {
		resolved, resolveError := net.DefaultResolver.LookupIPAddr(self.context, source.Host)
		if resolveError != nil {
			return nil, self.safeError(RemoteConnectionFailed)
		}
		for _, address := range resolved {
			if address.Zone == "" && !address.IP.IsUnspecified() && !address.IP.IsMulticast() {
				addresses = append(addresses, address.IP)
			}
		}
	}
	if len(addresses) == 0 {
		return nil, RemoteConnectionFailed
	}
	address := addresses[0]
	if remotePrivateAddress(address, source.Port) {
		return nil, RemoteSourceDenied
	}
	endpoint := &remoteEndpoint{address: address.String(), port: strconv.Itoa(source.Port), sslMode: remoteSSLMode(source)}
	if credentialError := self.prepareRemoteCredentials(source, endpoint); credentialError != nil {
		return nil, credentialError
	}
	return endpoint, nil
}

func remotePrivateAddress(address net.IP, port int) bool {
	privateHost, privatePort, addressError := net.SplitHostPort(config.Postgres.Address)
	parsedPort, portError := strconv.Atoi(privatePort)
	privateAddress := net.ParseIP(privateHost)
	return addressError == nil && portError == nil && privateAddress != nil && port == parsedPort && address.Equal(privateAddress)
}

func (self *job) remoteConnection(source RemoteSource, endpoint *remoteEndpoint) (*pgx.Conn, error) {
	if authorizeError := self.authorize(); authorizeError != nil {
		return nil, authorizeError
	}
	configuration, configurationError := remoteConfiguration(source, endpoint)
	if configurationError != nil {
		return nil, configurationError
	}
	connection, connectError := pgx.ConnectConfig(self.context, configuration)
	if connectError != nil {
		return nil, self.safeError(RemoteConnectionFailed)
	}
	return connection, nil
}

func remoteConfiguration(source RemoteSource, endpoint *remoteEndpoint) (*pgx.ConnConfig, error) {
	configuration, configurationError := metadataConfiguration()
	if configurationError != nil {
		return nil, RemoteConnectionFailed
	}
	configuration.Host = source.Host
	configuration.Port = uint16(source.Port)
	configuration.User = source.Username
	configuration.Password = source.Password
	configuration.Database = source.Database
	configuration.Fallbacks = nil
	configuration.Tracer = nil
	configuration.ConnectTimeout = RemoteConnectionTimeout
	configuration.MaxProtocolMessageBodyLen = MaximumMetadataMessageBytes
	configuration.RuntimeParams = map[string]string{
		"application_name":              "sqldash-remote-snapshot",
		"client_encoding":               "UTF8",
		"standard_conforming_strings":   "on",
		"default_transaction_read_only": "on",
		"options":                       "-c statement_timeout=30000",
		"search_path":                   "pg_catalog",
	}
	configuration.TLSConfig = remoteTLS(source, endpoint)
	configuration.SSLNegotiation = "postgres"
	configuration.LookupFunc = func(context.Context, string) ([]string, error) {
		return []string{endpoint.address}, nil
	}
	dialer := &net.Dialer{Timeout: RemoteConnectionTimeout}
	configuration.DialFunc = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, net.JoinHostPort(endpoint.address, endpoint.port))
	}
	return configuration, nil
}

func remoteTLS(source RemoteSource, endpoint *remoteEndpoint) *tls.Config {
	if endpoint.sslMode == SSLDisable {
		return nil
	}
	configuration := &tls.Config{ServerName: strings.TrimSuffix(source.Host, "."), MinVersion: tls.VersionTLS12, RootCAs: endpoint.roots}
	switch endpoint.sslMode {
	case SSLRequire:
		configuration.InsecureSkipVerify = true
	case SSLVerifyCA:
		configuration.InsecureSkipVerify = true
		configuration.VerifyConnection = func(state tls.ConnectionState) error {
			if len(state.PeerCertificates) == 0 {
				return InvalidRemoteTLS
			}
			intermediates := x509.NewCertPool()
			for _, certificate := range state.PeerCertificates[1:] {
				intermediates.AddCert(certificate)
			}
			_, verifyError := state.PeerCertificates[0].Verify(x509.VerifyOptions{Roots: endpoint.roots, Intermediates: intermediates, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}})
			if verifyError != nil {
				return InvalidRemoteTLS
			}
			return nil
		}
	}
	return configuration
}
