package redis

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/redishost"

	"github.com/gofiber/fiber/v3"
)

type Sample struct {
	RecordedAt           time.Time         `json:"recordedAt"`
	RunID                string            `json:"runId"`
	Fields               map[string]string `json:"fields"`
	CommandsPerSecond    *float64          `json:"commandsPerSecond"`
	HitRate              *float64          `json:"hitRate"`
	InputBytesPerSecond  *float64          `json:"inputBytesPerSecond"`
	OutputBytesPerSecond *float64          `json:"outputBytesPerSecond"`
	CounterReset         bool              `json:"counterReset"`
}

type InsightResult struct {
	Current        Sample   `json:"current"`
	History        []Sample `json:"history"`
	HistoryProblem string   `json:"historyProblem,omitempty"`
}

type CommandStat struct {
	Name                string `json:"name"`
	Calls               string `json:"calls"`
	Microseconds        string `json:"microseconds"`
	MicrosecondsPerCall string `json:"microsecondsPerCall"`
	RejectedCalls       string `json:"rejectedCalls"`
	FailedCalls         string `json:"failedCalls"`
}

type CommandResult struct {
	RecordedAt time.Time     `json:"recordedAt"`
	RunID      string        `json:"runId"`
	Commands   []CommandStat `json:"commands"`
	Truncated  bool          `json:"truncated"`
}

var historyLock sync.Mutex

var insightFields = strings.Fields("redis_version uptime_in_seconds connected_clients blocked_clients used_memory used_memory_rss maxmemory mem_fragmentation_ratio total_commands_processed instantaneous_ops_per_sec keyspace_hits keyspace_misses evicted_keys expired_keys total_net_input_bytes total_net_output_bytes rejected_connections rdb_last_save_time rdb_last_bgsave_status rdb_bgsave_in_progress aof_enabled aof_last_write_status aof_last_bgrewrite_status aof_rewrite_in_progress loading role master_link_status db0")

func parseInformation(text string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(text, "\n") {
		key, value, valid := strings.Cut(strings.TrimSpace(line), ":")
		if valid && len(key) < 256 {
			result[key] = limitedText(value, 4096)
		}
	}
	return result
}

func Insights(ctx context.Context, name string) (*InsightResult, *fiber.Error) {
	fresh, _, record, err := authorize(ctx, name, permissions.RedisInsightsView, true)
	if err != nil {
		return nil, err
	}
	client, clientError := redishost.Client(fresh, record, redishost.Inspect)
	if clientError != nil {
		return nil, operationError(clientError)
	}
	defer client.Close()
	information, informationError := client.Info(fresh, "server", "clients", "memory", "stats", "persistence", "replication", "keyspace").Result()
	if informationError != nil {
		return nil, operationError(informationError)
	}
	if len(information) > maximumReplyBytes {
		return nil, fiber.NewError(413, "Redis INFO exceeds the bounded statistics view.")
	}
	fields := parseInformation(information)
	current := Sample{RecordedAt: time.Now().UTC(), RunID: fields["run_id"], Fields: make(map[string]string)}
	for _, field := range insightFields {
		if value, present := fields[field]; present {
			current.Fields[field] = value
		}
	}
	if current.RunID == "" {
		return nil, fiber.NewError(503, "Redis did not return its server identity.")
	}
	result := &InsightResult{Current: current, History: make([]Sample, 0)}
	historyLock.Lock()
	defer historyLock.Unlock()
	var rows []models.RedisStatistics
	if database.DB.WithContext(fresh).Where("database_id = ? AND identity = ? AND recorded_at >= ?", record.ID, record.ClusterID, current.RecordedAt.Add(-time.Hour)).Order("recorded_at desc").Limit(maximumHistorySamples).Find(&rows).Error != nil {
		result.HistoryProblem = "Saved aggregate history is unavailable. Current measurements are still shown."
		return result, nil
	}
	for index := len(rows) - 1; index >= 0; index-- {
		var sample Sample
		if json.Unmarshal([]byte(rows[index].Payload), &sample) == nil && sample.RunID == rows[index].RunID {
			result.History = append(result.History, sample)
		} else {
			result.HistoryProblem = "Some saved aggregate measurements could not be read and are omitted."
		}
	}
	if len(result.History) > 0 {
		deriveSample(&result.Current, result.History[len(result.History)-1])
	}
	if len(rows) == 0 || current.RecordedAt.Sub(rows[0].RecordedAt) >= historyInterval {
		encoded, encodeError := json.Marshal(result.Current)
		if encodeError != nil || database.DB.WithContext(fresh).Create(&models.RedisStatistics{DatabaseID: record.ID, RecordedAt: current.RecordedAt, Identity: record.ClusterID, RunID: current.RunID, Payload: string(encoded)}).Error != nil {
			result.HistoryProblem = "This measurement could not be saved to aggregate history."
		} else if database.DB.WithContext(fresh).Where("database_id = ? AND (recorded_at < ? OR identity <> ?)", record.ID, current.RecordedAt.Add(-time.Hour), record.ClusterID).Delete(&models.RedisStatistics{}).Error != nil {
			result.HistoryProblem = "Aggregate history retention could not be updated."
		}
	}
	return result, nil
}

func deriveSample(current *Sample, previous Sample) {
	seconds := current.RecordedAt.Sub(previous.RecordedAt).Seconds()
	if current.RunID != previous.RunID || seconds <= 0 {
		current.CounterReset = true
		return
	}
	delta := func(field string) (uint64, bool) {
		present, presentError := strconv.ParseUint(current.Fields[field], 10, 64)
		before, beforeError := strconv.ParseUint(previous.Fields[field], 10, 64)
		return present - before, presentError == nil && beforeError == nil && present >= before
	}
	for field, target := range map[string]**float64{"total_commands_processed": &current.CommandsPerSecond, "total_net_input_bytes": &current.InputBytesPerSecond, "total_net_output_bytes": &current.OutputBytesPerSecond} {
		change, valid := delta(field)
		if !valid {
			current.CounterReset = true
			continue
		}
		rate := float64(change) / seconds
		*target = &rate
	}
	hits, hitValid := delta("keyspace_hits")
	misses, missValid := delta("keyspace_misses")
	if hitValid && missValid && (hits > 0 || misses > 0) {
		ratio := float64(hits) / (float64(hits) + float64(misses)) * 100
		current.HitRate = &ratio
	}
	if !hitValid || !missValid {
		current.CounterReset = true
	}
	if current.CounterReset {
		current.CommandsPerSecond, current.HitRate, current.InputBytesPerSecond, current.OutputBytesPerSecond = nil, nil, nil, nil
	}
}

func Commands(ctx context.Context, name string) (*CommandResult, *fiber.Error) {
	fresh, _, record, err := authorize(ctx, name, permissions.RedisInsightsView, true)
	if err != nil {
		return nil, err
	}
	client, clientError := redishost.Client(fresh, record, redishost.Inspect)
	if clientError != nil {
		return nil, operationError(clientError)
	}
	defer client.Close()
	information, informationError := client.Info(fresh, "server", "commandstats").Result()
	if informationError != nil {
		return nil, operationError(informationError)
	}
	if len(information) > maximumReplyBytes {
		return nil, fiber.NewError(413, "Redis INFO exceeds the bounded statistics view.")
	}
	fields := parseInformation(information)
	result := &CommandResult{RecordedAt: time.Now().UTC(), RunID: fields["run_id"], Commands: make([]CommandStat, 0)}
	for key, value := range fields {
		if !strings.HasPrefix(key, "cmdstat_") {
			continue
		}
		properties := make(map[string]string)
		for _, pair := range strings.Split(value, ",") {
			label, number, valid := strings.Cut(pair, "=")
			if valid {
				properties[label] = number
			}
		}
		result.Commands = append(result.Commands, CommandStat{Name: strings.TrimPrefix(key, "cmdstat_"), Calls: properties["calls"], Microseconds: properties["usec"], MicrosecondsPerCall: properties["usec_per_call"], RejectedCalls: properties["rejected_calls"], FailedCalls: properties["failed_calls"]})
	}
	sort.Slice(result.Commands, func(left, right int) bool {
		first, _ := strconv.ParseUint(result.Commands[left].Calls, 10, 64)
		second, _ := strconv.ParseUint(result.Commands[right].Calls, 10, 64)
		if first == second {
			return result.Commands[left].Name < result.Commands[right].Name
		}
		return first > second
	})
	if len(result.Commands) > 2048 {
		result.Commands = result.Commands[:2048]
		result.Truncated = true
	}
	return result, nil
}
