package redishost

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"sqldash/config"
	"sqldash/models"
	"sqldash/rediswire"
)

type MonitorEvent struct {
	Line      string
	Truncated bool
	Dropped   string
}

const maximumMonitorLine = 16 * 1024

var monitorSlots = make(chan struct{}, 8)

func monitorLine(reader *bufio.Reader) ([]byte, bool, error) {
	var contents []byte
	truncated := false
	for {
		fragment, err := reader.ReadSlice('\n')
		remaining := maximumMonitorLine - len(contents)
		if len(fragment) > remaining {
			contents = append(contents, fragment[:remaining]...)
			truncated = true
		} else {
			contents = append(contents, fragment...)
		}
		if err == nil {
			break
		}
		if !errors.Is(err, bufio.ErrBufferFull) {
			return nil, truncated, err
		}
	}
	return bytes.TrimSuffix(bytes.TrimSuffix(contents, []byte("\n")), []byte("\r")), truncated, nil
}

func sensitiveMonitorLine(line string) bool {
	closing := strings.Index(line, "] ")
	if closing < 0 {
		return true
	}
	command := line[closing+2:]
	if len(command) < 3 || command[0] != '"' {
		return true
	}
	ending := strings.IndexByte(command[1:], '"')
	if ending < 0 {
		return true
	}
	name := strings.ToUpper(command[1 : ending+1])
	switch name {
	case "AUTH", "HELLO", "ACL", "CONFIG", "DEBUG", "MODULE", "REPLICAOF", "SLAVEOF", "MIGRATE", "PSYNC", "SYNC", "SHUTDOWN":
		return true
	}
	return false
}

func MonitorCommands(ctx context.Context, record *models.Database, deliver func(MonitorEvent) error) error {
	if deliver == nil {
		return InvalidSettings
	}
	select {
	case monitorSlots <- struct{}{}:
		defer func() { <-monitorSlots }()
	case <-ctx.Done():
		return ctx.Err()
	default:
		return CapacityExceeded
	}
	instance, err := Instance(ctx, record)
	if err != nil || instance.State != models.DatabaseReady {
		return Unavailable
	}
	password, err := systemPassword(instance, Monitor, false)
	if err != nil {
		return err
	}
	defer clear(password)
	connection, err := dialInstance(ctx, instance)
	if err != nil {
		return err
	}
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(controlTimeout))
	authentication := rediswire.EncodeCommand([]byte("AUTH"), []byte(systemUsername(record.ID, Monitor)), password)
	err = rediswire.WriteAll(connection, authentication)
	clear(authentication)
	if err != nil {
		return Unavailable
	}
	reader := bufio.NewReaderSize(connection, 4096)
	response, err := rediswire.ReadValue(reader, rediswire.Limits{MaximumBytes: 1024})
	if err != nil || response.Kind != '+' || !bytes.Equal(response.Bytes, []byte("OK")) {
		return Unavailable
	}
	if err := rediswire.WriteAll(connection, rediswire.EncodeCommand([]byte("MONITOR"))); err != nil {
		return Unavailable
	}
	response, err = rediswire.ReadValue(reader, rediswire.Limits{MaximumBytes: 1024})
	if err != nil || response.Kind != '+' || !bytes.Equal(response.Bytes, []byte("OK")) {
		return Unavailable
	}
	_ = connection.SetDeadline(time.Time{})
	stream, cancel := context.WithCancel(ctx)
	defer cancel()
	finished := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		select {
		case <-stream.Done():
			connection.Close()
		case <-stopped:
		}
	}()
	defer close(stopped)
	queue := make(chan MonitorEvent, min(config.Redis.MaximumStreamBuffer, 1024))
	failures := make(chan error, 1)
	var dropped atomic.Uint64
	go func() {
		defer close(finished)
		defer close(queue)
		for {
			line, truncated, err := monitorLine(reader)
			if err != nil {
				failures <- err
				return
			}
			text := strings.TrimPrefix(string(line), "+")
			if sensitiveMonitorLine(text) {
				continue
			}
			event := MonitorEvent{Line: text, Truncated: truncated}
			select {
			case queue <- event:
			default:
				dropped.Add(1)
			}
			select {
			case <-stream.Done():
				return
			default:
			}
		}
	}()
	defer func() { cancel(); connection.Close(); <-finished }()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, available := <-queue:
			if !available {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				select {
				case err := <-failures:
					if errors.Is(err, io.EOF) {
						return Unavailable
					}
				default:
				}
				return Unavailable
			}
			event.Dropped = strconv.FormatUint(dropped.Swap(0), 10)
			if err := deliver(event); err != nil {
				return err
			}
		}
	}
}
