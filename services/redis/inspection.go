package redis

import (
	"context"
	"strconv"
	"strings"
	"time"

	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/redisdata"
	"sqldash/redishost"

	"github.com/gofiber/fiber/v3"
)

type SlowEntry struct {
	ID           string            `json:"id"`
	Time         time.Time         `json:"time"`
	Microseconds string            `json:"microseconds"`
	Arguments    []redisdata.Bytes `json:"arguments"`
	Client       string            `json:"client"`
	Name         string            `json:"name"`
	Truncated    bool              `json:"truncated"`
}

type SlowResult struct {
	Entries   []SlowEntry `json:"entries"`
	Truncated bool        `json:"truncated"`
}

type ClientView struct {
	ID       string `json:"id"`
	Address  string `json:"address"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Age      string `json:"age"`
	Idle     string `json:"idle"`
	Command  string `json:"command"`
	Flags    string `json:"flags"`
}

type TerminateClientRequest struct {
	ID    string `json:"id"`
	RunID string `json:"runId"`
}

type ClientResult struct {
	RunID     string       `json:"runId"`
	Clients   []ClientView `json:"clients"`
	Truncated bool         `json:"truncated"`
}

func SlowLog(ctx context.Context, name string) (*SlowResult, *fiber.Error) {
	fresh, _, record, err := authorize(ctx, name, permissions.RedisSlowLogView, true)
	if err != nil {
		return nil, err
	}
	client, clientError := redishost.Client(fresh, record, redishost.Inspect)
	if clientError != nil {
		return nil, operationError(clientError)
	}
	defer client.Close()
	entries, readError := client.SlowLogGet(fresh, 100).Result()
	if readError != nil {
		return nil, operationError(readError)
	}
	result := &SlowResult{Entries: make([]SlowEntry, 0, len(entries)), Truncated: len(entries) >= 100}
	budget := maximumReplyBytes / 2
	for _, entry := range entries {
		if len(entry.Args) > 0 && sensitiveCommand(entry.Args[0]) {
			continue
		}
		shown := SlowEntry{ID: strconv.FormatInt(entry.ID, 10), Time: entry.Time, Microseconds: strconv.FormatInt(entry.Duration.Microseconds(), 10), Client: limitedText(entry.ClientAddr, 512), Name: limitedText(entry.ClientName, 512), Arguments: make([]redisdata.Bytes, 0, len(entry.Args))}
		for _, argument := range entry.Args {
			limit := min(8192, max(0, budget))
			value := []byte(argument)
			if len(value) > limit {
				value = value[:limit]
				shown.Truncated = true
			}
			budget -= len(value)
			shown.Arguments = append(shown.Arguments, redisdata.EncodeBytes(value))
			if budget <= 0 {
				shown.Truncated = true
				break
			}
		}
		result.Entries = append(result.Entries, shown)
		if budget <= 0 {
			result.Truncated = true
			break
		}
	}
	return result, nil
}

func ResetSlowLog(ctx context.Context, name string) *fiber.Error {
	fresh, actor, record, err := authorize(ctx, name, permissions.RedisSlowLogManage, true)
	if err != nil {
		return err
	}
	if !actor.CanDatabase(record, permissions.RedisSlowLogView) {
		return fiber.NewError(403, AuthorityRefused)
	}
	return operationError(redishost.ResetSlowLog(fresh, record))
}

func Clients(ctx context.Context, name string) (*ClientResult, *fiber.Error) {
	fresh, _, record, err := authorize(ctx, name, permissions.RedisActivityView, true)
	if err != nil {
		return nil, err
	}
	client, clientError := redishost.Client(fresh, record, redishost.Inspect)
	if clientError != nil {
		return nil, operationError(clientError)
	}
	defer client.Close()
	text, listError := client.ClientList(fresh).Result()
	if listError != nil {
		return nil, operationError(listError)
	}
	instance, instanceError := redishost.Instance(fresh, record)
	if instanceError != nil {
		return nil, operationError(instanceError)
	}
	information, informationError := client.Info(fresh, "server").Result()
	if informationError != nil || parseInformation(information)["run_id"] != instance.RunID {
		return nil, fiber.NewError(409, DatabaseNotReady)
	}
	result := &ClientResult{RunID: instance.RunID, Clients: make([]ClientView, 0)}
	for _, line := range strings.Split(text, "\n") {
		fields := make(map[string]string)
		for _, pair := range strings.Fields(line) {
			key, value, valid := strings.Cut(pair, "=")
			if valid {
				fields[key] = limitedText(value, 512)
			}
		}
		if fields["id"] == "" || strings.Contains(fields["user"], "_system_") {
			continue
		}
		if len(result.Clients) >= 1000 {
			result.Truncated = true
			break
		}
		result.Clients = append(result.Clients, ClientView{ID: fields["id"], Address: fields["addr"], Name: fields["name"], Username: fields["user"], Age: fields["age"], Idle: fields["idle"], Command: fields["cmd"], Flags: fields["flags"]})
	}
	return result, nil
}

func TerminateClient(ctx context.Context, name string, asked TerminateClientRequest) *fiber.Error {
	identifier, parseError := strconv.ParseInt(asked.ID, 10, 64)
	if parseError != nil || identifier < 1 || strconv.FormatInt(identifier, 10) != asked.ID || len(asked.RunID) != 40 {
		return fiber.NewError(400, InvalidRequest)
	}
	fresh, _, record, err := authorize(ctx, name, permissions.RedisActivityManage, true)
	if err != nil {
		return err
	}
	listed, err := Clients(fresh, name)
	if err != nil {
		return err
	}
	if listed.RunID != asked.RunID {
		return fiber.NewError(409, DatabaseNotReady)
	}
	var username string
	for _, client := range listed.Clients {
		if client.ID == asked.ID {
			username = client.Username
			break
		}
	}
	if !models.PublicRedisUsername(username) {
		return fiber.NewError(404, "Public Redis client not found.")
	}
	var count int64
	if database.DB.WithContext(fresh).Model(&models.RedisConnection{}).Where("database_id = ? AND instance_identity = ? AND username = ?", record.ID, record.ClusterID, username).Count(&count).Error != nil {
		return fiber.NewError(503, RedisUnavailable)
	}
	if count != 1 {
		return fiber.NewError(404, "Public Redis client not found.")
	}
	return operationError(redishost.KillClientForRun(fresh, record, identifier, asked.RunID))
}

func sensitiveCommand(command string) bool {
	switch strings.ToUpper(command) {
	case "AUTH", "HELLO", "ACL", "CONFIG", "MIGRATE", "REPLICAOF", "SLAVEOF", "SYNC", "PSYNC", "MONITOR":
		return true
	}
	return false
}
