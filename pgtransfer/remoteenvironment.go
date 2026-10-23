package pgtransfer

func (self *job) remoteEnvironment(source RemoteSource, endpoint *remoteEndpoint) []string {
	environment := append(self.processEnvironmentVersion(true, endpoint.serverVersion),
		"PGHOST="+source.Host, "PGHOSTADDR="+endpoint.address, "PGPORT="+endpoint.port,
		"PGPASSFILE="+endpoint.passwordPath, "PGSSLMODE="+endpoint.sslMode,
		"PGTARGETSESSIONATTRS=any", "PGSSLSNI=1", "PGSSLMINPROTOCOLVERSION=TLSv1.2",
	)
	if endpoint.certificatePath != "" {
		environment = append(environment, "PGSSLROOTCERT="+endpoint.certificatePath)
	}
	return environment
}
