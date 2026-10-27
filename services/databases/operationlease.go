package databases

import (
	"context"
	"errors"
	"time"

	"sqldash/database"
	"sqldash/models"
)

const (
	operationLeaseDuration     = 2 * time.Minute
	operationHeartbeatInterval = 15 * time.Second
	operationHeartbeatTimeout  = 2 * time.Second
)

func renewOperation(ctx context.Context, operation *models.DatabaseOperation) (bool, error) {
	until := time.Now().Add(operationLeaseDuration)
	result := database.DB.WithContext(ctx).Model(&models.DatabaseOperation{}).Where("id = ? AND state = ? AND lease_until = ?", operation.ID, models.DatabaseApplying, operation.LeaseUntil).Update("heartbeat_until", until)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 1 {
		return true, nil
	}
	var current models.DatabaseOperation
	if err := database.DB.WithContext(ctx).First(&current, operation.ID).Error; err != nil {
		return false, err
	}
	if current.State == models.DatabaseReady && current.LeaseUntil == nil {
		return false, nil
	}
	return false, errors.New(DatabaseNotReady)
}

func heartbeatOperation(parent context.Context, operation *models.DatabaseOperation) (context.Context, func()) {
	work, cancel := context.WithCancelCause(parent)
	heartbeat, stopHeartbeat := context.WithCancel(context.WithoutCancel(parent))
	finished := make(chan struct{})
	if operation.LeaseUntil == nil {
		cancel(errors.New(DatabaseNotReady))
		stopHeartbeat()
		close(finished)
		return work, func() {}
	}
	lease := *operation.LeaseUntil
	claimed := models.DatabaseOperation{LeaseUntil: &lease}
	claimed.ID = operation.ID
	go func() {
		defer close(finished)
		ticker := time.NewTicker(operationHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeat.Done():
				return
			case <-ticker.C:
				checked, stop := context.WithTimeout(heartbeat, operationHeartbeatTimeout)
				active, err := renewOperation(checked, &claimed)
				stop()
				if err != nil {
					cancel(err)
					return
				}
				if !active {
					return
				}
			}
		}
	}()
	return work, func() {
		stopHeartbeat()
		cancel(nil)
		<-finished
	}
}
