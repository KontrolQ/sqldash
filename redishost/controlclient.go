package redishost

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"

	"sqldash/config"
	"sqldash/models"
)

func requestControl(ctx context.Context, request controlRequest) (*controlResponse, error) {
	if !config.Redis.Enabled || !managedSupported() {
		return nil, Unavailable
	}
	token, err := controlToken()
	if err != nil {
		return nil, err
	}
	defer clear(token)
	encoded, err := json.Marshal(request)
	if err != nil {
		return nil, Unavailable
	}
	transport := &http.Transport{DialContext: func(parent context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{Timeout: controlTimeout}).DialContext(parent, "unix", controlSocketPath())
	}, DisableKeepAlives: true}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport}
	message, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://redis.internal/control", bytes.NewReader(encoded))
	if err != nil {
		return nil, Unavailable
	}
	message.Header.Set("X-Sqldash-Redis", string(token))
	message.Header.Set("Content-Type", "application/json")
	response, err := client.Do(message)
	if err != nil {
		return nil, Unavailable
	}
	defer response.Body.Close()
	var result controlResponse
	decoder := json.NewDecoder(io.LimitReader(response.Body, maximumControlBytes))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&result) != nil {
		return nil, Unavailable
	}
	if response.StatusCode != http.StatusOK || result.Failure != "" {
		switch result.Failure {
		case "identity":
			return nil, IdentityMismatch
		case "capacity":
			return nil, CapacityExceeded
		case "storage":
			return nil, StorageUnowned
		case "policy":
			return nil, PolicyUnconfirmed
		default:
			return nil, Unavailable
		}
	}
	return &result, nil
}

func Ready(ctx context.Context) error {
	if !config.Redis.Enabled {
		return nil
	}
	_, err := requestControl(ctx, controlRequest{Action: "ready"})
	return err
}

func Create(ctx context.Context, record *models.Database) error {
	if record == nil || record.Kind() != models.EngineRedis || record.ID == 0 {
		return IdentityMismatch
	}
	response, err := requestControl(ctx, controlRequest{Action: "create", DatabaseID: record.ID, Identity: record.ClusterID})
	if err != nil {
		return err
	}
	if response.Identity == "" || record.ClusterID != "" && record.ClusterID != response.Identity {
		return IdentityMismatch
	}
	record.ClusterID = response.Identity
	return nil
}

func Delete(ctx context.Context, record *models.Database) error {
	if record == nil || record.Kind() != models.EngineRedis || record.ID == 0 || record.ClusterID == "" {
		return IdentityMismatch
	}
	_, err := requestControl(ctx, controlRequest{Action: "delete", DatabaseID: record.ID, Identity: record.ClusterID})
	return err
}

func Stop(ctx context.Context, record *models.Database) error {
	if record == nil || record.Kind() != models.EngineRedis || record.ID == 0 || record.ClusterID == "" {
		return IdentityMismatch
	}
	_, err := requestControl(ctx, controlRequest{Action: "stop", DatabaseID: record.ID, Identity: record.ClusterID})
	return err
}
