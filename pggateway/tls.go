package pggateway

import (
	"context"
	"sqldash/nativetls"
)

func ConfigureTLS(ctx context.Context, options TLSOptions) (*TLSMaterial, error) {
	if options.AuthorityName == "" {
		options.AuthorityName = certificateAuthorityName
	}
	if options.LeafName == "" {
		options.LeafName = certificateLeafName
	}
	options.ApplicationProtocols = []string{postgresqlApplicationProtocol}
	return nativetls.Configure(ctx, options)
}
