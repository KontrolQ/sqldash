package insights

import (
	"context"
	"net/url"
	"time"

	"sqldash/models"
	redisservice "sqldash/services/redis"
)

type RedisDatabaseView struct {
	Name         string
	Path         string
	CommandsPath string
	Memory       string
	Clients      string
	Keys         string
	Operations   string
	CollectedAt  string
	Problem      string
}

type RedisCommandView struct {
	Database         string
	Path             string
	Name             string
	Calls            string
	Microseconds     string
	MeanMicroseconds string
	Rejected         string
	Failed           string
}

type RedisContext struct {
	Databases    []RedisDatabaseView
	Commands     []RedisCommandView
	WithCommands bool
	Truncated    bool
}

func redisInsights(ctx context.Context, records []models.Database, withCommands bool) *RedisContext {
	shown := &RedisContext{WithCommands: withCommands}
	bounded, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	for _, record := range records {
		if record.Kind() != models.EngineRedis {
			continue
		}
		if len(shown.Databases) >= 100 {
			shown.Truncated = true
			break
		}
		base := "/databases/" + url.PathEscape(record.Name)
		row := RedisDatabaseView{Name: record.Name, Path: base + "/insights", CommandsPath: base + "/queries", Memory: UnavailableValue, Clients: UnavailableValue, Keys: UnavailableValue, Operations: UnavailableValue, CollectedAt: UnavailableValue}
		if !record.Available() {
			row.Problem = "Database is not ready."
			shown.Databases = append(shown.Databases, row)
			continue
		}
		current, stop := context.WithTimeout(bounded, 2*time.Second)
		if withCommands {
			result, err := redisservice.Commands(current, record.Name)
			if err != nil {
				row.Problem = "Command statistics are unavailable."
			} else {
				row.CollectedAt = result.RecordedAt.Local().Format(time.DateTime)
				shown.Truncated = shown.Truncated || result.Truncated
				for _, command := range result.Commands {
					if len(shown.Commands) >= 200 {
						shown.Truncated = true
						break
					}
					shown.Commands = append(shown.Commands, RedisCommandView{Database: record.Name, Path: row.CommandsPath, Name: command.Name, Calls: command.Calls, Microseconds: command.Microseconds, MeanMicroseconds: command.MicrosecondsPerCall, Rejected: command.RejectedCalls, Failed: command.FailedCalls})
				}
			}
		} else {
			result, err := redisservice.Insights(current, record.Name)
			if err != nil {
				row.Problem = "Current measurements are unavailable."
			} else {
				fields := result.Current.Fields
				row.Memory = redisField(fields, "used_memory")
				row.Clients = redisField(fields, "connected_clients")
				row.Keys = redisField(fields, "db0")
				row.Operations = redisField(fields, "instantaneous_ops_per_sec")
				row.CollectedAt = result.Current.RecordedAt.Local().Format(time.DateTime)
				row.Problem = result.HistoryProblem
			}
		}
		stop()
		shown.Databases = append(shown.Databases, row)
	}
	return shown
}

func redisField(fields map[string]string, key string) string {
	if value, present := fields[key]; present {
		return value
	}
	return UnavailableValue
}
