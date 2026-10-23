package databases

import (
	"context"
	"strconv"

	"sqldash/engines"
	"sqldash/models"
	explorer "sqldash/services/explorer"
)

func tableCount(ctx context.Context, record *models.Database) string {
	if record.Kind() == models.EnginePostgres {
		backend, err := engines.For(record)
		if err != nil {
			return "—"
		}
		catalog, supported := backend.(engines.Catalog)
		if !supported {
			return "—"
		}
		relations, err := catalog.Relations(ctx, record, "")
		if err != nil {
			return "—"
		}
		return strconv.Itoa(len(relations))
	}
	tables, err := explorer.Tables(ctx, record.Name)
	if err != nil {
		return "—"
	}
	return strconv.Itoa(len(tables))
}
