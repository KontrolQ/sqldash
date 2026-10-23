package analytics

import (
	"context"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"sqldash/database"
	"sqldash/models"
)

const databaseQueryMetadataBatch = 200

type DatabaseQuerySummary struct {
	QuerySummary
	DatabaseID   uint
	DatabaseName string
}

type databaseQueryIdentity struct {
	databaseID   uint
	databaseName string
	digest       string
}

func (self DatabaseQuerySummary) Key() string {
	return models.EngineLibSQL + ":" + strconv.FormatUint(uint64(self.DatabaseID), 10) + ":" + url.QueryEscape(self.DatabaseName) + ":" + url.QueryEscape(self.Digest)
}

func (self DatabaseQuerySummary) identity() databaseQueryIdentity {
	return databaseQueryIdentity{
		databaseID:   self.DatabaseID,
		databaseName: self.DatabaseName,
		digest:       self.Digest,
	}
}

func DatabaseQueries(ctx context.Context, databaseName string, resolution models.Resolution, since time.Time, until time.Time, allowed []string) ([]DatabaseQuerySummary, error) {
	if len(allowed) == 0 {
		return make([]DatabaseQuerySummary, 0), nil
	}

	records, findError := rollupsBetweenContext(ctx, databaseName, resolution, since, until, allowed)
	if findError != nil {
		return nil, findError
	}

	summaries := summarizeDatabaseQueries(records)
	if namingError := nameDatabaseQueries(ctx, databaseName, summaries, allowed); namingError != nil {
		return nil, namingError
	}

	return summaries, nil
}

func summarizeDatabaseQueries(records []models.Rollup) []DatabaseQuerySummary {
	gathered := make(map[databaseQueryIdentity]*DatabaseQuerySummary)

	for _, record := range records {
		identity := databaseQueryIdentity{
			databaseID:   record.DatabaseID,
			databaseName: record.DatabaseName,
			digest:       record.Digest,
		}

		held, seen := gathered[identity]
		if !seen {
			held = &DatabaseQuerySummary{
				QuerySummary: QuerySummary{Digest: record.Digest},
				DatabaseID:   record.DatabaseID,
				DatabaseName: record.DatabaseName,
			}
			gathered[identity] = held
		}

		held.absorb(record)
	}

	summaries := make([]DatabaseQuerySummary, 0, len(gathered))
	for _, held := range gathered {
		summaries = append(summaries, *held)
	}

	sort.Slice(summaries, func(firstIndex int, secondIndex int) bool {
		first := summaries[firstIndex]
		second := summaries[secondIndex]
		if first.TotalDuration != second.TotalDuration {
			return first.TotalDuration > second.TotalDuration
		}
		if first.DatabaseID != second.DatabaseID {
			return first.DatabaseID < second.DatabaseID
		}
		if first.DatabaseName != second.DatabaseName {
			return first.DatabaseName < second.DatabaseName
		}
		return first.Digest < second.Digest
	})

	return summaries
}

func nameDatabaseQueries(ctx context.Context, databaseName string, summaries []DatabaseQuerySummary, allowed []string) error {
	if len(allowed) == 0 || len(summaries) == 0 {
		return nil
	}

	query := currentLibSQL(database.DB.Model(&models.Fingerprint{}), "fingerprints").
		Select("database_id", "database_name", "digest", "normalised", "example").
		Where("database_name IN ?", allowed).
		Order("id asc")
	if databaseName != "" {
		query = query.Where("database_name = ?", databaseName)
	}

	for batchStart := 0; batchStart < len(summaries); batchStart += databaseQueryMetadataBatch {
		batchEnd := min(batchStart+databaseQueryMetadataBatch, len(summaries))
		matches := make([]string, 0, batchEnd-batchStart)
		values := make([]any, 0, 3*(batchEnd-batchStart))
		for _, summary := range summaries[batchStart:batchEnd] {
			matches = append(matches, "(database_id = ? AND database_name = ? AND digest = ?)")
			values = append(values, summary.DatabaseID, summary.DatabaseName, summary.Digest)
		}

		fingerprints := make([]models.Fingerprint, 0, batchEnd-batchStart)
		if findError := query.WithContext(ctx).Where("("+strings.Join(matches, " OR ")+")", values...).Find(&fingerprints).Error; findError != nil {
			return findError
		}

		applyDatabaseQueryNames(summaries[batchStart:batchEnd], fingerprints)
	}

	return nil
}

func applyDatabaseQueryNames(summaries []DatabaseQuerySummary, fingerprints []models.Fingerprint) {
	byIdentity := make(map[databaseQueryIdentity]models.Fingerprint, len(fingerprints))
	for _, fingerprint := range fingerprints {
		identity := databaseQueryIdentity{databaseID: fingerprint.DatabaseID, databaseName: fingerprint.DatabaseName, digest: fingerprint.Digest}
		if _, seen := byIdentity[identity]; !seen {
			byIdentity[identity] = fingerprint
		}
	}
	for index := range summaries {
		if fingerprint, seen := byIdentity[summaries[index].identity()]; seen {
			summaries[index].Normalised = fingerprint.Normalised
			summaries[index].Example = fingerprint.Example
		}
	}
}
