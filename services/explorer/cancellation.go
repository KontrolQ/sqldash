package explorer

import (
	"context"
	"net/http"
	"regexp"
	"sync"
	"time"

	"sqldash/models"
	"sqldash/permissions"
	"sqldash/services/access"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

type consoleJob struct {
	AccountID  uint
	DatabaseID uint
	Cancel     context.CancelFunc
}

var consoleJobs = struct {
	sync.Mutex
	jobs map[string]consoleJob
}{jobs: make(map[string]consoleJob)}

var consoleIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{16,80}$`)

func registerConsole(parent context.Context, record *models.Database, runID string) (context.Context, func(), *fiber.Error) {
	requestContext, cancel := context.WithTimeout(parent, 2*time.Minute)
	if runID == "" {
		return requestContext, cancel, nil
	}
	if !consoleIDPattern.MatchString(runID) {
		cancel()
		return nil, nil, consoleRequestError("Invalid query identifier.")
	}
	accountID := access.AccountID(parent)
	consoleJobs.Lock()
	defer consoleJobs.Unlock()
	if _, exists := consoleJobs.jobs[runID]; exists {
		cancel()
		return nil, nil, consoleRequestError("This query identifier is already running.")
	}
	if len(consoleJobs.jobs) >= 256 {
		cancel()
		return nil, nil, shortcuts.ServiceError(http.StatusTooManyRequests, "Too many console submissions are running.")
	}
	consoleJobs.jobs[runID] = consoleJob{AccountID: accountID, DatabaseID: record.ID, Cancel: cancel}
	return requestContext, func() {
		cancel()
		consoleJobs.Lock()
		delete(consoleJobs.jobs, runID)
		consoleJobs.Unlock()
	}, nil
}

func CancelConsole(requestContext context.Context, databaseName string, runID string) *fiber.Error {
	record, permissionError := authorizedDatabase(requestContext, databaseName, permissions.SQLExecute)
	if permissionError != nil {
		return permissionError
	}
	consoleJobs.Lock()
	defer consoleJobs.Unlock()
	job, found := consoleJobs.jobs[runID]
	if !found {
		return shortcuts.ServiceError(http.StatusNotFound, "This submission is not running. It may still be preparing or have already finished.")
	}
	if job.AccountID != access.AccountID(requestContext) || job.DatabaseID != record.ID {
		return shortcuts.ServiceError(http.StatusForbidden, "This submission belongs to another session.")
	}
	job.Cancel()
	return nil
}
