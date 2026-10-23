package postgres

import (
	"context"
	"strconv"
	"time"

	"sqldash/engines"
	"sqldash/models"
	"sqldash/pgcluster/control"

	"github.com/jackc/pgx/v5"
)

type Backend struct{}

var _ engines.Backend = (*Backend)(nil)
var _ engines.Catalog = (*Backend)(nil)

func init() {
	engines.Register(models.EnginePostgres, New())
}

func New() *Backend {
	return &Backend{}
}

func Ready(ctx context.Context) error {
	return control.Ready(ctx)
}

func Connect(ctx context.Context, record *models.Database, mode engines.Mode) (*pgx.Conn, error) {
	return control.Connect(ctx, record, control.SessionMode(mode))
}

func (self *Backend) Features() engines.Features {
	return engines.Features{
		Schemas:           true,
		TypedValues:       true,
		CompositeKeys:     true,
		NativeConnections: true,
		Activity:          true,
		Extensions:        true,
		MeasuredQueries:   true,
	}
}

func (self *Backend) Create(ctx context.Context, record *models.Database) error {
	return control.Provision(ctx, record)
}

func (self *Backend) Delete(ctx context.Context, record *models.Database) error {
	return control.Drop(ctx, record)
}

func (self *Backend) Statistics(ctx context.Context, record *models.Database) (*engines.StatisticsRecord, error) {
	statistics, statisticsError := control.Statistics(ctx, record)
	if statisticsError != nil {
		return nil, statisticsError
	}
	return &engines.StatisticsRecord{
		Identifier:       statistics.ClusterID + ":" + strconv.FormatUint(uint64(statistics.DatabaseOID), 10),
		StorageBytesUsed: statistics.StorageBytes,
	}, nil
}

func (self *Backend) Configuration(ctx context.Context, record *models.Database) (*engines.ConfigurationRecord, error) {
	if _, identityError := control.DatabaseRoles(record); identityError != nil {
		return nil, identityError
	}
	return &engines.ConfigurationRecord{}, nil
}

func (self *Backend) SetConfiguration(ctx context.Context, record *models.Database, configuration engines.ConfigurationRecord) error {
	return engines.Unsupported
}

func (self *Backend) Fork(ctx context.Context, source, target *models.Database, timestamp *time.Time) error {
	return engines.Unsupported
}
