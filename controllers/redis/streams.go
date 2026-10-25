package redis

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"time"

	service "sqldash/services/redis"

	"github.com/gofiber/fiber/v3"
)

func Monitor(context fiber.Ctx) error {
	ticket, err := service.PrepareStream(context.Context(), context.Params("name"), true)
	if err != nil {
		return respond(context, nil, err)
	}
	return sendStream(context, ticket, nil)
}

func Subscribe(context fiber.Ctx) error {
	var asked service.SubscriptionRequest
	if err := decodeRequest(context, &asked); err != nil {
		return respond(context, nil, err)
	}
	if _, err := service.SubscriptionChannel(asked); err != nil {
		return respond(context, nil, err)
	}
	ticket, err := service.PrepareStream(context.Context(), context.Params("name"), false)
	if err != nil {
		return respond(context, nil, err)
	}
	return sendStream(context, ticket, &asked)
}

func sendStream(request fiber.Ctx, ticket *service.StreamTicket, subscription *service.SubscriptionRequest) error {
	request.Set("Content-Type", "text/event-stream")
	request.Set("Cache-Control", "no-store, no-transform")
	request.Set("X-Accel-Buffering", "no")
	request.Set("X-Content-Type-Options", "nosniff")
	connection := request.RequestCtx().Conn()
	return request.SendStreamWriter(func(writer *bufio.Writer) {
		defer connection.SetWriteDeadline(time.Time{})
		flush := func() error {
			if err := connection.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
				return err
			}
			return writer.Flush()
		}
		streamContext, cancel := context.WithCancel(context.Background())
		defer cancel()
		events := make(chan service.LiveEvent, 8)
		completed := make(chan error, 1)
		go func() {
			completed <- ticket.Run(streamContext, subscription, func(event service.LiveEvent) error {
				select {
				case events <- event:
					return nil
				case <-streamContext.Done():
					return streamContext.Err()
				}
			})
		}()
		heartbeat := time.NewTicker(time.Second)
		defer heartbeat.Stop()
		if _, err := writer.WriteString("event: ready\ndata: {}\n\n"); err != nil || flush() != nil {
			return
		}
		for {
			select {
			case event := <-events:
				encoded, err := json.Marshal(event)
				if err != nil {
					return
				}
				if _, err := fmt.Fprintf(writer, "event: message\ndata: %s\n\n", encoded); err != nil || flush() != nil {
					return
				}
			case <-heartbeat.C:
				if _, err := writer.WriteString(": heartbeat\n\n"); err != nil || flush() != nil {
					return
				}
			case err := <-completed:
				message := "Stream stopped. Start again to reconnect."
				if err != nil {
					message = "Stream stopped because the connection, database generation or account permissions changed. Start again after checking access."
				}
				encoded, _ := json.Marshal(problem{Error: message})
				fmt.Fprintf(writer, "event: stopped\ndata: %s\n\n", encoded)
				flush()
				return
			}
		}
	})
}
