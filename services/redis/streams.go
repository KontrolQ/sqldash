package redis

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"sqldash/config"
	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/redisdata"
	"sqldash/redishost"
	"sqldash/services/access"
	"sqldash/sessions"

	"github.com/gofiber/fiber/v3"
)

type LiveEvent struct {
	Kind       string           `json:"kind"`
	Line       string           `json:"line,omitempty"`
	Channel    *redisdata.Bytes `json:"channel,omitempty"`
	Pattern    *redisdata.Bytes `json:"pattern,omitempty"`
	Payload    *redisdata.Bytes `json:"payload,omitempty"`
	RecordedAt time.Time        `json:"recordedAt"`
	Truncated  bool             `json:"truncated"`
	Dropped    string           `json:"dropped,omitempty"`
}

type StreamTicket struct {
	accountID  uint
	databaseID uint
	name       string
	identity   string
	generation uint64
	permission string
	session    sessions.Reference
}

type SubscriptionRequest struct {
	Channel redisdata.Bytes `json:"channel"`
	Pattern bool            `json:"pattern"`
}

type PublishRequest struct {
	Channel redisdata.Bytes `json:"channel"`
	Payload redisdata.Bytes `json:"payload"`
}

type PublishResult struct {
	Receivers string `json:"receivers"`
}

func SubscriptionChannel(asked SubscriptionRequest) ([]byte, *fiber.Error) {
	channel, err := redisdata.DecodeBytes(asked.Channel)
	if err != nil || len(channel) > 4096 {
		return nil, fiber.NewError(400, InvalidRequest)
	}
	return channel, nil
}

func PrepareStream(ctx context.Context, name string, monitor bool) (*StreamTicket, *fiber.Error) {
	permission := permissions.RedisSubscribe
	if monitor {
		permission = permissions.RedisMonitorView
	}
	fresh, actor, record, err := authorize(ctx, name, permission, true)
	if err != nil {
		return nil, err
	}
	var instance models.RedisInstance
	if database.DB.WithContext(fresh).Where("database_id = ? AND identity = ?", record.ID, record.ClusterID).First(&instance).Error != nil || instance.State != models.DatabaseReady {
		return nil, fiber.NewError(409, DatabaseNotReady)
	}
	return &StreamTicket{accountID: actor.Account.ID, databaseID: record.ID, name: strings.Clone(name), identity: instance.Identity, generation: instance.Generation, permission: permission, session: sessions.ContextReference(fresh)}, nil
}

func (self *StreamTicket) authority(ctx context.Context) (context.Context, *models.Database, *fiber.Error) {
	actor := &access.AccountAccess{}
	actor.Account.ID = self.accountID
	fresh, currentActor, record, err := authorize(access.WithContext(ctx, actor), self.name, self.permission, true)
	if err != nil {
		return fresh, nil, err
	}
	if currentActor.Account.SessionVersion != self.session.Version || currentActor.Account.ID != self.session.AccountID || !sessions.ValidReference(fresh, self.session) {
		return fresh, nil, fiber.NewError(403, AuthorityRefused)
	}
	if record.ID != self.databaseID || record.ClusterID != self.identity {
		return fresh, nil, fiber.NewError(409, DatabaseNotReady)
	}
	var instance models.RedisInstance
	if database.DB.WithContext(fresh).Where("database_id = ? AND identity = ? AND generation = ? AND state = ?", self.databaseID, self.identity, self.generation, models.DatabaseReady).First(&instance).Error != nil {
		return fresh, nil, fiber.NewError(409, DatabaseNotReady)
	}
	return fresh, record, nil
}

func (self *StreamTicket) Run(parent context.Context, subscription *SubscriptionRequest, deliver func(LiveEvent) error) error {
	bounded, cancel := context.WithCancel(parent)
	defer cancel()
	fresh, record, err := self.authority(bounded)
	if err != nil {
		return err
	}
	interval := config.Redis.RevalidationInterval
	if interval <= 0 || interval > 2*time.Second {
		interval = 2 * time.Second
	}
	finished := make(chan struct{})
	defer close(finished)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-bounded.Done():
				return
			case <-finished:
				return
			case <-ticker.C:
				if _, _, authorityError := self.authority(bounded); authorityError != nil {
					cancel()
					return
				}
			}
		}
	}()
	if subscription == nil {
		var opening sync.Once
		var openingError *fiber.Error
		return redishost.MonitorCommands(fresh, record, func(event redishost.MonitorEvent) error {
			opening.Do(func() { _, _, openingError = self.authority(bounded) })
			if openingError != nil {
				return openingError
			}
			select {
			case <-bounded.Done():
				return bounded.Err()
			default:
			}
			return deliver(LiveEvent{Kind: "command", Line: event.Line, RecordedAt: time.Now().UTC(), Truncated: event.Truncated, Dropped: event.Dropped})
		})
	}
	channel, decodeError := SubscriptionChannel(*subscription)
	if decodeError != nil {
		return decodeError
	}
	client, clientError := redishost.Client(fresh, record, redishost.Subscribe)
	if clientError != nil {
		return operationError(clientError)
	}
	defer client.Close()
	if _, _, authorityError := self.authority(bounded); authorityError != nil {
		return authorityError
	}
	pubsub := client.Subscribe(fresh)
	defer pubsub.Close()
	if subscription.Pattern {
		clientError = pubsub.PSubscribe(fresh, string(channel))
	} else {
		clientError = pubsub.Subscribe(fresh, string(channel))
	}
	if clientError != nil {
		return operationError(clientError)
	}
	if _, receiveError := pubsub.Receive(fresh); receiveError != nil {
		return operationError(receiveError)
	}
	messages := make(chan LiveEvent, min(64, max(1, config.Redis.MaximumStreamBuffer)))
	received := make(chan error, 1)
	var dropped atomic.Uint64
	go func() {
		for {
			message, receiveError := pubsub.ReceiveMessage(fresh)
			if receiveError != nil {
				select {
				case received <- receiveError:
				default:
				}
				return
			}
			payload := []byte(message.Payload)
			truncated := len(payload) > maximumLiveBytes
			if truncated {
				payload = payload[:maximumLiveBytes]
			}
			channelBytes, patternBytes := []byte(message.Channel), []byte(message.Pattern)
			if len(channelBytes) > 4096 {
				channelBytes = channelBytes[:4096]
				truncated = true
			}
			if len(patternBytes) > 4096 {
				patternBytes = patternBytes[:4096]
				truncated = true
			}
			encodedChannel, encodedPayload := redisdata.EncodeBytes(channelBytes), redisdata.EncodeBytes(payload)
			encodedPattern := redisdata.EncodeBytes(patternBytes)
			event := LiveEvent{Kind: "message", Channel: &encodedChannel, Pattern: &encodedPattern, Payload: &encodedPayload, RecordedAt: time.Now().UTC(), Truncated: truncated}
			select {
			case messages <- event:
			default:
				dropped.Add(1)
			}
			select {
			case <-bounded.Done():
				return
			default:
			}
		}
	}()
	for {
		select {
		case <-bounded.Done():
			return bounded.Err()
		case receiveError := <-received:
			return operationError(receiveError)
		case event := <-messages:
			event.Dropped = strconv.FormatUint(dropped.Swap(0), 10)
			if deliveryError := deliver(event); deliveryError != nil {
				return deliveryError
			}
		}
	}
}

func Publish(ctx context.Context, name string, asked PublishRequest) (*PublishResult, *fiber.Error) {
	channel, channelError := redisdata.DecodeBytes(asked.Channel)
	payload, payloadError := redisdata.DecodeBytes(asked.Payload)
	if channelError != nil || payloadError != nil || len(channel) > 4096 || len(payload) > maximumLiveBytes {
		return nil, fiber.NewError(400, InvalidRequest)
	}
	fresh, _, record, err := authorize(ctx, name, permissions.RedisPublish, true)
	if err != nil {
		return nil, err
	}
	client, clientError := redishost.Client(fresh, record, redishost.Publish)
	if clientError != nil {
		return nil, operationError(clientError)
	}
	defer client.Close()
	count, publishError := client.Publish(fresh, string(channel), payload).Result()
	if publishError != nil {
		problem := operationError(publishError)
		return nil, fiber.NewError(problem.Code, "Message delivery could not be confirmed. It may have reached subscribers; retrying can deliver a duplicate.")
	}
	return &PublishResult{Receivers: strconv.FormatInt(count, 10)}, nil
}
