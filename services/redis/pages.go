package redis

import (
	"context"
	"net"
	"net/url"
	"strconv"
	"time"

	"sqldash/config"
	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/services/access"
	"sqldash/services/databases"
	"sqldash/utils/collections"

	"github.com/gofiber/fiber/v3"
)

type CredentialView struct {
	Identifier uint
	Label      string
	Username   string
	Scope      string
	Primary    bool
	Subscribe  bool
	Publish    bool
	State      string
	ExpiresAt  string
	Expired    bool
	CanRotate  bool
	CanRevoke  bool
	CanReveal  bool
}

type credentialMetadata struct {
	models.RedisConnection
	PendingAvailable bool
}

type RuleView struct {
	Identifier uint
	Network    string
	Note       string
}

type PageData struct {
	Title           string
	Name            string
	Engine          string
	IsRedis         bool
	BasePath        string
	APIPath         string
	Address         string
	Host            string
	Port            int
	Section         string
	Actions         map[string]bool
	Unavailable     string
	MetadataProblem string
	Protected       bool
	MemoryLimit     string
	EvictionPolicy  string
	Persistence     string
	Fsync           string
	Credentials     []CredentialView
	Rules           []RuleView
	ViewOptions     []collections.Option
	ByteEncodings   []collections.Option
	KeyTypes        []collections.Option
	FilterKeyTypes  []collections.Option
	ExpiryModes     []collections.Option
	ExpiryUnits     []collections.Option
}

func Page(ctx context.Context, name, section string) (*PageData, *fiber.Error) {
	permission := map[string]string{"overview": permissions.RedisDatabaseView, "data": permissions.RedisDataRead, "console": permissions.RedisConsoleExecute, "insights": permissions.RedisInsightsView, "queries": permissions.RedisInsightsView, "slowlog": permissions.RedisSlowLogView, "monitor": permissions.RedisMonitorView, "pubsub": permissions.RedisSubscribe, "connections": permissions.RedisCredentialsManage}[section]
	fresh, actor, err := databases.FreshActor(ctx)
	if err != nil {
		return nil, err
	}
	record, recordError := access.Database(actor, name)
	if recordError != nil || record.Kind() != models.EngineRedis {
		return nil, fiber.NewError(404, DatabaseMissing)
	}
	allowed := actor.CanDatabase(record, permission)
	if section == "pubsub" {
		allowed = allowed || actor.CanDatabase(record, permissions.RedisPublish)
	}
	if section == "connections" {
		allowed = allowed || actor.CanDatabase(record, permissions.RedisActivityView)
	}
	if !allowed {
		return nil, fiber.NewError(403, AuthorityRefused)
	}
	basePath := "/databases/" + url.PathEscape(name)
	data := &PageData{Title: map[string]string{"overview": "Overview", "data": "Data", "console": "Console", "insights": "Insights", "queries": "Top commands", "slowlog": "Slow log", "monitor": "Monitor", "pubsub": "Pub/Sub", "connections": "Connections"}[section], Name: name, Engine: models.EngineRedis, IsRedis: true, BasePath: basePath, APIPath: basePath + "/redis", Address: "rediss://" + net.JoinHostPort(config.RedisHost(), strconv.Itoa(config.Redis.PublicPort)) + "/0", Host: config.RedisHost(), Port: config.Redis.PublicPort, Section: section, Actions: redisActions(actor, record), Protected: record.Protected}
	var primary models.RedisConnection
	if database.DB.WithContext(fresh).Select("username").Where("database_id = ? AND instance_identity = ? AND primary_connection = ?", record.ID, record.ClusterID, true).First(&primary).Error == nil {
		address := url.URL{Scheme: "rediss", Host: net.JoinHostPort(data.Host, strconv.Itoa(data.Port)), Path: "/0", User: url.User(primary.Username)}
		data.Address = address.String()
	}
	if section == "data" {
		dataOptions(data)
	}
	if !record.Available() {
		data.Unavailable = DatabaseNotReady
	}
	if section == "overview" {
		var instance models.RedisInstance
		if database.DB.WithContext(fresh).Where("database_id = ? AND identity = ?", record.ID, record.ClusterID).First(&instance).Error != nil {
			data.Unavailable = "Redis configuration is currently unavailable."
		} else {
			data.MemoryLimit = strconv.FormatInt(instance.MemoryLimit, 10)
			data.EvictionPolicy = instance.EvictionPolicy
			data.Persistence = instance.Persistence
			data.Fsync = instance.Fsync
		}
		if data.Actions["CanManageNetwork"] {
			var rules []models.AccessRule
			if database.DB.WithContext(fresh).Where("database_name = ?", name).Order("id").Limit(1000).Find(&rules).Error != nil {
				data.MetadataProblem = "Network rules could not be loaded."
			} else {
				for _, rule := range rules {
					data.Rules = append(data.Rules, RuleView{Identifier: rule.ID, Network: rule.Network, Note: rule.Note})
				}
			}
		}
	}
	if section == "connections" && data.Actions["CanManageConnections"] {
		var credentials []credentialMetadata
		if database.DB.WithContext(fresh).Model(&models.RedisConnection{}).Select("id, account_id, instance_identity, generation, label, username, scope, primary_connection, subscribe, publish, state, expires_at, revoked_at, CASE WHEN pending_secret <> '' THEN 1 ELSE 0 END AS pending_available").Where("database_id = ? AND instance_identity = ?", record.ID, record.ClusterID).Order("primary_connection desc, id desc").Limit(1000).Find(&credentials).Error != nil {
			data.MetadataProblem = "Credential metadata is unavailable."
		} else {
			if len(credentials) == 1000 {
				data.MetadataProblem = "Showing up to 1,000 recent credential records, with primary credentials first."
			}
			for _, credential := range credentials {
				profileAllowed := (credential.Scope != models.ScopeReadWrite || data.Actions["CanWriteData"]) && (!credential.Subscribe || data.Actions["CanSubscribeRedis"]) && (!credential.Publish || data.Actions["CanPublishRedis"])
				shown := CredentialView{Identifier: credential.ID, Label: credential.Label, Username: credential.Username, Scope: string(credential.Scope), Primary: credential.PrimaryConnection, Subscribe: credential.Subscribe, Publish: credential.Publish, State: credential.State, ExpiresAt: "No expiry", CanRotate: profileAllowed, CanRevoke: profileAllowed && credential.RevokedAt == nil, CanReveal: profileAllowed && credential.AccountID == actor.Account.ID && credential.PendingAvailable && credential.Usable(time.Now())}
				if credential.ExpiresAt != nil {
					shown.ExpiresAt = credential.ExpiresAt.UTC().Format(time.RFC3339)
					shown.Expired = !time.Now().Before(*credential.ExpiresAt)
				}
				if credential.RevokedAt != nil {
					shown.State = "revoked"
				}
				data.Credentials = append(data.Credentials, shown)
			}
		}
	}
	return data, nil
}

func redisActions(actor *access.AccountAccess, record *models.Database) map[string]bool {
	result := access.Actions(actor, record)
	for flag, permission := range map[string]string{"CanViewDatabases": permissions.RedisDatabaseView, "CanReadData": permissions.RedisDataRead, "CanWriteData": permissions.RedisDataWrite, "CanRunSQL": permissions.RedisConsoleExecute, "CanViewInsights": permissions.RedisInsightsView, "CanEditDatabaseSettings": permissions.RedisDatabaseSettings, "CanExportDatabases": permissions.RedisDatabaseExport, "CanForkDatabases": permissions.RedisDatabaseFork, "CanDeleteDatabases": permissions.RedisDatabaseDelete, "CanManageNetwork": permissions.RedisNetworkManage, "CanManageConnections": permissions.RedisCredentialsManage, "CanViewActivity": permissions.RedisActivityView, "CanManageActivity": permissions.RedisActivityManage, "CanViewSlowLog": permissions.RedisSlowLogView, "CanResetSlowLog": permissions.RedisSlowLogManage, "CanMonitorRedis": permissions.RedisMonitorView, "CanSubscribeRedis": permissions.RedisSubscribe, "CanPublishRedis": permissions.RedisPublish} {
		result[flag] = actor.CanDatabase(record, permission)
	}
	result["CanShareDatabase"] = actor.CanShare(record)
	return result
}
