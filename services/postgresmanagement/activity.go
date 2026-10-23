package postgresmanagement

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/pgcluster/control"
	"sqldash/services/access"
	"sqldash/services/databases"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func authorize(ctx context.Context, name string, permission string) (context.Context, *models.Database, *fiber.Error) {
	fresh, record, err := databases.Authorize(ctx, name, permission, true)
	if err != nil {
		return fresh, nil, err
	}
	if record.Kind() != models.EnginePostgres {
		return fresh, nil, shortcuts.ServiceError(http.StatusBadRequest, PostgresRequired)
	}
	return fresh, record, nil
}

func GetActivity(ctx context.Context, name string) (*ActivityContext, *fiber.Error) {
	fresh, record, err := authorize(ctx, name, permissions.PostgresActivityView)
	if err != nil {
		return nil, err
	}
	bounded, cancel := context.WithTimeout(fresh, 10*time.Second)
	defer cancel()
	activity, readError := control.Activity(bounded, record)
	if readError != nil {
		return nil, shortcuts.ServiceError(http.StatusServiceUnavailable, ActivityUnavailable)
	}
	var connections []models.PostgresConnection
	if err := database.DB.WithContext(fresh).Select("username", "account_id", "label").Where("database_id = ?", record.ID).Find(&connections).Error; err != nil {
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, ActivityUnavailable)
	}
	byUsername := make(map[string]models.PostgresConnection, len(connections))
	accountIDs := make([]uint, 0, len(connections))
	for _, connection := range connections {
		byUsername[connection.Username] = connection
		accountIDs = append(accountIDs, connection.AccountID)
	}
	var accounts []models.Account
	if len(accountIDs) > 0 {
		if err := database.DB.WithContext(fresh).Select("id", "username", "display_name").Where("id IN ?", accountIDs).Find(&accounts).Error; err != nil {
			return nil, shortcuts.ServiceError(http.StatusInternalServerError, ActivityUnavailable)
		}
	}
	accountNames := make(map[uint]string, len(accounts))
	for _, account := range accounts {
		accountNames[account.ID] = account.DisplayName
		if accountNames[account.ID] == "" {
			accountNames[account.ID] = account.Username
		}
	}
	shown := &ActivityContext{Title: "Connections", Name: record.Name, CollectedAt: time.Now().Local().Format(time.DateTime), CanManage: access.FromContext(fresh).Can(permissions.PostgresActivityManage)}
	for _, entry := range activity {
		connection := byUsername[entry.Username]
		view := ActivityView{PID: entry.PID, Username: entry.Username, Account: accountNames[connection.AccountID], Label: connection.Label,
			Application: entry.Application, StartedAt: entry.StartedAt.Local().Format(time.DateTime), BackendStart: entry.StartedAt.UTC().Format(time.RFC3339Nano),
			State: entry.State, Wait: strings.Trim(strings.Join([]string{entry.WaitType, entry.WaitEvent}, " · "), " ·"), Query: entry.Query,
			QueryAge: age(entry.QueryStartedAt), TransactionAge: age(entry.TransactionStartedAt), Manageable: entry.Manageable}
		if entry.ClientAddress != nil {
			view.ClientAddress = *entry.ClientAddress
		}
		if view.Account == "" {
			view.Account = "Browser session"
		}
		if entry.State == "active" {
			shown.ActiveQueries++
		} else {
			view.QueryAge = "—"
		}
		shown.Rows = append(shown.Rows, view)
	}
	shown.Connections = len(shown.Rows)
	return shown, nil
}

func ChangeActivity(ctx context.Context, name string, target SessionTarget, terminate bool) (*models.Database, *fiber.Error) {
	fresh, record, err := authorize(ctx, name, permissions.PostgresActivityManage)
	if err != nil {
		return nil, err
	}
	if !access.FromContext(fresh).Can(permissions.PostgresActivityView) || target.PID == 0 || target.BackendStart.IsZero() {
		return nil, shortcuts.ServiceError(http.StatusForbidden, ActivityRefused)
	}
	bounded, cancel := context.WithTimeout(fresh, 10*time.Second)
	defer cancel()
	nativeTarget := control.ActivityTarget{PID: target.PID, StartedAt: target.BackendStart}
	var changeError error
	if terminate {
		changeError = control.TerminateActivity(bounded, record, nativeTarget)
	} else {
		changeError = control.CancelActivity(bounded, record, nativeTarget)
	}
	if changeError != nil {
		return nil, shortcuts.ServiceError(http.StatusConflict, ActivityFailed)
	}
	return record, nil
}

func age(started *time.Time) string {
	if started == nil {
		return "—"
	}
	elapsed := time.Since(*started)
	if elapsed < 0 {
		elapsed = 0
	}
	return fmt.Sprintf("%.1f s", elapsed.Seconds())
}
