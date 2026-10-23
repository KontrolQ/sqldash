package pgmetrics

import (
	"context"
	"math"
	"sort"
	"time"

	"sqldash/database"
	"sqldash/models"
)

const SeriesMaximumGap = 3 * SampleInterval

type SeriesPoint struct {
	At             time.Time
	Calls          *int64
	ExecutionMs    *float64
	StorageBytes   *int64
	Connections    *int64
	BreakBefore    bool
	CounterPartial bool
}

type Series struct {
	Points                []SeriesPoint
	DatabaseCount         int
	ObservedDatabases     int
	CounterBuckets        int
	PartialCounterBuckets int
	GaugeBuckets          int
	RecordedCalls         *int64
	RecordedExecution     *float64
	LatestStorageBytes    *int64
	LatestConnections     *int64
}

type seriesSampleKey struct {
	DatabaseID uint
	Collected  int64
}

type seriesBucket struct {
	Calls          int64
	ExecutionMs    float64
	CounterPresent bool
	CounterInvalid bool
	CounterPartial bool
	StorageBytes   *int64
	Connections    *int64
	BreakBefore    bool
}

func ReadSeries(ctx context.Context, records []models.Database, since time.Time, until time.Time, slots int) (*Series, error) {
	var snapshots []models.PostgresStatistics
	var statements []models.PostgresStatementStatistics
	for _, record := range records {
		if record.Kind() != models.EnginePostgres || !record.Available() || record.ID == 0 || record.ClusterID == "" || record.BackendOID == 0 {
			continue
		}
		var collected []models.PostgresStatistics
		err := database.DB.WithContext(ctx).
			Where("database_id = ? AND cluster_id = ? AND database_oid = ? AND version = ? AND collected_at >= ? AND collected_at <= ?", record.ID, record.ClusterID, record.BackendOID, SampleVersion, since.Add(-SeriesMaximumGap), until).
			Order("collected_at ASC").Find(&collected).Error
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, collected...)
		var measured []models.PostgresStatementStatistics
		err = database.DB.WithContext(ctx).Model(&models.PostgresStatementStatistics{}).
			Where("database_id = ? AND cluster_id = ? AND database_oid = ? AND version = ? AND top_level = ? AND collected_at >= ? AND collected_at <= ?", record.ID, record.ClusterID, record.BackendOID, SampleVersion, true, since, until).
			Select("database_id, cluster_id, database_oid, version, top_level, collected_at, interval_started_at, CASE WHEN continuous AND calls_delta >= 0 AND execution_delta_ms >= 0 THEN 1 ELSE 0 END AS continuous, SUM(calls_delta) AS calls_delta, SUM(execution_delta_ms) AS execution_delta_ms").
			Group("database_id, cluster_id, database_oid, version, top_level, collected_at, interval_started_at, CASE WHEN continuous AND calls_delta >= 0 AND execution_delta_ms >= 0 THEN 1 ELSE 0 END").Scan(&measured).Error
		if err != nil {
			return nil, err
		}
		statements = append(statements, measured...)
	}
	return ReduceSeries(records, snapshots, statements, since, until, slots), nil
}

func ReduceSeries(records []models.Database, snapshots []models.PostgresStatistics, statements []models.PostgresStatementStatistics, since time.Time, until time.Time, slots int) *Series {
	held := &Series{}
	if slots < 1 || !until.After(since) || until.Sub(since) < time.Duration(slots) {
		return held
	}
	width := until.Sub(since) / time.Duration(slots)
	held.Points = make([]SeriesPoint, slots)
	for index := range held.Points {
		held.Points[index].At = since.Add(time.Duration(index) * width)
	}
	registrations := make(map[uint]models.Database)
	for _, record := range records {
		if record.Kind() == models.EnginePostgres {
			registrations[record.ID] = record
		}
	}
	held.DatabaseCount = len(registrations)
	if held.DatabaseCount == 0 {
		return held
	}
	byDatabase := make(map[uint][]models.PostgresStatistics)
	for _, sample := range snapshots {
		record, exists := registrations[sample.DatabaseID]
		if exists && matchingSeriesIdentity(record, sample.ClusterID, sample.DatabaseOID, sample.Version) && !sample.CollectedAt.After(until) {
			byDatabase[record.ID] = append(byDatabase[record.ID], sample)
		}
	}
	bySample := make(map[seriesSampleKey][]models.PostgresStatementStatistics)
	for _, statement := range statements {
		record, exists := registrations[statement.DatabaseID]
		if exists && statement.TopLevel && matchingSeriesIdentity(record, statement.ClusterID, statement.DatabaseOID, statement.Version) {
			key := seriesSampleKey{DatabaseID: record.ID, Collected: statement.CollectedAt.UnixNano()}
			bySample[key] = append(bySample[key], statement)
		}
	}
	bucketsByDatabase := make([][]seriesBucket, 0, held.DatabaseCount)
	latestStorage, latestConnections := int64(0), int64(0)
	latestComplete := true
	for identifier := range registrations {
		collected := byDatabase[identifier]
		sort.Slice(collected, func(first int, second int) bool {
			return collected[first].CollectedAt.Before(collected[second].CollectedAt)
		})
		buckets := make([]seriesBucket, slots)
		for index, sample := range collected {
			if sample.CollectedAt.Before(since) {
				continue
			}
			position := seriesPosition(sample.CollectedAt, since, width, slots)
			bucket := &buckets[position]
			if sample.StorageBytes >= 0 && sample.Connections >= 0 {
				bucket.StorageBytes = seriesInteger(sample.StorageBytes)
				bucket.Connections = seriesInteger(sample.Connections)
			}
			var previous *models.PostgresStatistics
			if index > 0 {
				previous = &collected[index-1]
			}
			continuous := continuousSeriesSample(sample, previous)
			if !continuous {
				bucket.BreakBefore = true
			}
			if previous != nil && sample.CollectedAt.Sub(previous.CollectedAt) > SeriesMaximumGap {
				invalidateSeriesBuckets(buckets, previous.CollectedAt, sample.CollectedAt, since, width)
			}
			if sample.IntervalStartedAt != nil && sample.IntervalStartedAt.Before(since) {
				continue
			}
			if !continuous || !sample.StatementsAvailable || !sample.StatementsContinuous || sample.IntervalStartedAt == nil {
				if previous != nil || sample.IntervalStartedAt != nil {
					bucket.CounterInvalid = true
				}
				continue
			}
			key := seriesSampleKey{DatabaseID: identifier, Collected: sample.CollectedAt.UnixNano()}
			measured := bySample[key]
			hasComparableStatements := len(measured) == 0
			for _, statement := range measured {
				if !statement.Continuous || statement.IntervalStartedAt == nil || !statement.IntervalStartedAt.Equal(*sample.IntervalStartedAt) || statement.CallsDelta < 0 || statement.ExecutionDeltaMs < 0 || math.IsNaN(statement.ExecutionDeltaMs) || math.IsInf(statement.ExecutionDeltaMs, 0) {
					bucket.CounterPartial = true
					continue
				}
				hasComparableStatements = true
				bucket.Calls += statement.CallsDelta
				bucket.ExecutionMs += statement.ExecutionDeltaMs
			}
			bucket.CounterPresent = bucket.CounterPresent || hasComparableStatements
		}
		bucketsByDatabase = append(bucketsByDatabase, buckets)
		if len(collected) == 0 {
			latestComplete = false
			continue
		}
		latest := collected[len(collected)-1]
		held.ObservedDatabases++
		if until.Sub(latest.CollectedAt) > SeriesMaximumGap || latest.StorageBytes < 0 || latest.Connections < 0 {
			latestComplete = false
			continue
		}
		latestStorage += latest.StorageBytes
		latestConnections += latest.Connections
	}
	if latestComplete {
		held.LatestStorageBytes = seriesInteger(latestStorage)
		held.LatestConnections = seriesInteger(latestConnections)
	}
	calls, execution := int64(0), 0.0
	for index := range held.Points {
		counterComplete, gaugeComplete := true, true
		bucketCalls, bucketExecution, bucketStorage, bucketConnections := int64(0), 0.0, int64(0), int64(0)
		for _, buckets := range bucketsByDatabase {
			bucket := buckets[index]
			counterComplete = counterComplete && bucket.CounterPresent && !bucket.CounterInvalid
			gaugeComplete = gaugeComplete && bucket.StorageBytes != nil && bucket.Connections != nil
			held.Points[index].BreakBefore = held.Points[index].BreakBefore || bucket.BreakBefore
			held.Points[index].CounterPartial = held.Points[index].CounterPartial || bucket.CounterPartial
			bucketCalls += bucket.Calls
			bucketExecution += bucket.ExecutionMs
			if bucket.StorageBytes != nil && bucket.Connections != nil {
				bucketStorage += *bucket.StorageBytes
				bucketConnections += *bucket.Connections
			}
		}
		if counterComplete && !math.IsNaN(bucketExecution) && !math.IsInf(bucketExecution, 0) {
			held.Points[index].Calls = seriesInteger(bucketCalls)
			held.Points[index].ExecutionMs = seriesNumber(bucketExecution)
			held.CounterBuckets++
			if held.Points[index].CounterPartial {
				held.PartialCounterBuckets++
			}
			calls += bucketCalls
			execution += bucketExecution
		}
		if gaugeComplete {
			held.Points[index].StorageBytes = seriesInteger(bucketStorage)
			held.Points[index].Connections = seriesInteger(bucketConnections)
			held.GaugeBuckets++
		}
	}
	if held.CounterBuckets > 0 {
		held.RecordedCalls = seriesInteger(calls)
		held.RecordedExecution = seriesNumber(execution)
	}
	return held
}

func matchingSeriesIdentity(record models.Database, clusterID string, databaseOID uint32, version uint) bool {
	return record.Available() && record.ID != 0 && record.ClusterID != "" && record.BackendOID != 0 && record.ClusterID == clusterID && record.BackendOID == databaseOID && version == SampleVersion
}

func continuousSeriesSample(sample models.PostgresStatistics, previous *models.PostgresStatistics) bool {
	return previous != nil && sample.Continuous && sample.IntervalStartedAt != nil && sample.IntervalStartedAt.Equal(previous.CollectedAt) &&
		sample.CollectedAt.After(previous.CollectedAt) && sample.CollectedAt.Sub(previous.CollectedAt) <= SeriesMaximumGap &&
		sample.PostmasterStartedAt.Equal(previous.PostmasterStartedAt) && sameTime(sample.ResetAt, previous.ResetAt) &&
		sameTime(sample.StatementResetAt, previous.StatementResetAt) && sample.StatementsDeallocated == previous.StatementsDeallocated && monotonicDatabase(sample, *previous)
}

func seriesPosition(moment time.Time, since time.Time, width time.Duration, slots int) int {
	return min(max(int(moment.Sub(since)/width), 0), slots-1)
}

func invalidateSeriesBuckets(buckets []seriesBucket, from time.Time, to time.Time, since time.Time, width time.Duration) {
	first := seriesPosition(from, since, width, len(buckets))
	last := seriesPosition(to, since, width, len(buckets))
	for index := first; index <= last; index++ {
		buckets[index].CounterInvalid = true
	}
}

func seriesInteger(value int64) *int64    { return &value }
func seriesNumber(value float64) *float64 { return &value }
