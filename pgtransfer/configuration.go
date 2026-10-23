package pgtransfer

import (
	"os"

	"github.com/jackc/pgx/v5"
)

func metadataConfiguration() (*pgx.ConnConfig, error) {
	if os.Getenv("PGSERVICE") != "" {
		return nil, InvalidOptions
	}
	return pgx.ParseConfig("host=127.0.0.1 port=55432 user=sqldash_transfer dbname=postgres password='' sslmode=disable passfile=/dev/null connect_timeout=10 target_session_attrs=any sslnegotiation=postgres min_protocol_version=3.0 max_protocol_version=3.0 channel_binding=prefer require_auth=''")
}
