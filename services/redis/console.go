package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"sqldash/permissions"
	"sqldash/redisdata"
	"sqldash/redishost"
	"sqldash/rediswire"
	"sqldash/services/access"

	"github.com/gofiber/fiber/v3"
	redisclient "github.com/redis/go-redis/v9"
)

type ConsoleRequest struct {
	Commands string `json:"commands"`
}
type Reply struct {
	Type      string           `json:"type"`
	Text      string           `json:"text,omitempty"`
	Bytes     *redisdata.Bytes `json:"bytes,omitempty"`
	Items     []Reply          `json:"items,omitempty"`
	Truncated bool             `json:"truncated,omitempty"`
}
type ConsoleResult struct {
	Replies      []Reply `json:"replies"`
	Milliseconds string  `json:"milliseconds"`
	Truncated    bool    `json:"truncated"`
	Processed    string  `json:"processed"`
	Requested    string  `json:"requested"`
	Discarded    bool    `json:"discarded"`
}

var consoleReadCommands = commandSet("PING ECHO DBSIZE TYPE EXISTS GET MGET GETRANGE STRLEN TTL PTTL EXPIRETIME PEXPIRETIME SCAN HSCAN HGET HGETALL HMGET HEXISTS HLEN HKEYS HVALS HSTRLEN LRANGE LINDEX LLEN SSCAN SISMEMBER SMISMEMBER SCARD SMEMBERS SRANDMEMBER ZSCAN ZCARD ZCOUNT ZLEXCOUNT ZSCORE ZMSCORE ZRANK ZREVRANK ZRANGE ZREVRANGE ZRANGEBYSCORE ZREVRANGEBYSCORE ZRANGEBYLEX XRANGE XREVRANGE XLEN DUMP JSON.GET JSON.TYPE JSON.OBJKEYS JSON.OBJLEN JSON.ARRLEN JSON.STRLEN")
var consoleWriteCommands = commandSet("SET SETNX SETEX PSETEX MSET MSETNX GETSET GETDEL GETEX APPEND SETRANGE INCR INCRBY DECR DECRBY INCRBYFLOAT DEL UNLINK EXPIRE PEXPIRE EXPIREAT PEXPIREAT PERSIST RENAME RENAMENX COPY HSET HSETNX HMSET HDEL HINCRBY HINCRBYFLOAT LPUSH RPUSH LPUSHX RPUSHX LSET LTRIM LREM LPOP RPOP LMOVE SADD SREM SMOVE SPOP ZADD ZREM ZINCRBY ZREMRANGEBYRANK ZREMRANGEBYSCORE ZREMRANGEBYLEX ZPOPMIN ZPOPMAX XADD XDEL XTRIM JSON.SET JSON.DEL JSON.FORGET JSON.NUMINCRBY JSON.STRAPPEND JSON.ARRAPPEND JSON.ARRINSERT JSON.ARRPOP JSON.ARRTRIM JSON.CLEAR JSON.TOGGLE")

func commandSet(names string) map[string]bool {
	result := make(map[string]bool)
	for _, name := range strings.Fields(names) {
		result[name] = true
	}
	return result
}

func Execute(ctx context.Context, name string, asked ConsoleRequest) (*ConsoleResult, *fiber.Error) {
	commands, parseError := rediswire.ParseCommands(asked.Commands)
	if parseError != nil || len(commands) == 0 || len(commands) > 100 {
		return nil, fiber.NewError(400, "Enter at most 100 valid Redis commands. Binary arguments may use quoted \\xNN escapes.")
	}
	fresh, actor, record, err := authorize(ctx, name, permissions.RedisConsoleExecute, true)
	if err != nil {
		return nil, err
	}
	if !actor.CanDatabase(record, permissions.RedisDataRead) {
		return nil, fiber.NewError(403, AuthorityRefused)
	}
	transaction := strings.EqualFold(string(commands[0][0]), "MULTI")
	write := false
	result := &ConsoleResult{Replies: make([]Reply, 0, len(commands))}
	for index, command := range commands {
		verb := strings.ToUpper(string(command[0]))
		if transaction && ((index == 0 && verb == "MULTI") || (index == len(commands)-1 && (verb == "EXEC" || verb == "DISCARD"))) {
			if len(command) != 1 {
				return nil, fiber.NewError(400, InvalidRequest)
			}
			continue
		}
		if !consoleReadCommands[verb] && !consoleWriteCommands[verb] {
			return nil, fiber.NewError(400, "Command "+limitedText(verb, 128)+" is not available in this console. Use the dedicated database controls for administration, monitoring and Pub/Sub.")
		}
		write = write || consoleWriteCommands[verb]
	}
	if transaction && (len(commands) < 2 || !strings.EqualFold(string(commands[len(commands)-1][0]), "EXEC") && !strings.EqualFold(string(commands[len(commands)-1][0]), "DISCARD")) {
		return nil, fiber.NewError(400, "A MULTI batch must end with EXEC or DISCARD in the same request.")
	}
	if write && !actor.CanDatabase(record, permissions.RedisDataWrite) {
		return nil, fiber.NewError(403, ReadOnly)
	}
	profile := redishost.ReadOnly
	if write {
		profile = redishost.ReadWrite
	}
	client, clientError := redishost.Client(fresh, record, profile)
	if clientError != nil {
		return nil, operationError(clientError)
	}
	defer client.Close()
	connection := client.Conn()
	defer connection.Close()
	instance, instanceError := redishost.Instance(fresh, record)
	if instanceError != nil {
		return nil, operationError(instanceError)
	}
	clientID, identityError := connection.Do(fresh, "CLIENT", "ID").Int64()
	if identityError != nil || clientID < 1 {
		return nil, fiber.NewError(503, RedisUnavailable)
	}
	checkAuthority := func(step context.Context) error {
		_, currentActor, current, err := authorize(step, name, permissions.RedisConsoleExecute, true)
		if err != nil {
			return err
		}
		if current.ID != record.ID || current.ClusterID != record.ClusterID || !currentActor.CanDatabase(current, permissions.RedisDataRead) || write && !currentActor.CanDatabase(current, permissions.RedisDataWrite) {
			return fiber.NewError(403, AuthorityRefused)
		}
		currentInstance, currentError := redishost.Instance(step, current)
		if currentError != nil || currentInstance.Generation != instance.Generation || currentInstance.RunID != instance.RunID {
			return fiber.NewError(409, DatabaseNotReady)
		}
		return nil
	}
	deadline, cancel := context.WithTimeout(fresh, 20*time.Second)
	defer cancel()
	guarded, stop, guardError := access.Guard(deadline, checkAuthority, func(cleanup context.Context) error {
		if err := redishost.KillClientForRun(cleanup, record, clientID, instance.RunID); err != nil {
			return redishost.Quarantine(cleanup, record, instance.Generation)
		}
		return nil
	})
	if guardError != nil {
		return nil, fiber.NewError(403, AuthorityRefused)
	}
	defer stop()
	bounded := guarded
	defer func() {
		if transaction {
			cleanup, stop := context.WithTimeout(context.Background(), time.Second)
			defer stop()
			connection.Do(cleanup, "DISCARD")
		}
	}()
	start := time.Now()
	budget, elements := maximumReplyBytes, maximumReplyElements
	for _, command := range commands {
		if err := checkAuthority(bounded); err != nil {
			return nil, fiber.NewError(403, "Redis console authority changed. Completed commands are not rolled back; inspect affected keys before retrying.")
		}
		arguments := make([]any, len(command))
		for index, argument := range command {
			arguments[index] = argument
		}
		value, commandError := connection.Do(bounded, arguments...).Result()
		var reply Reply
		if commandError != nil && !errors.Is(commandError, redisclient.Nil) {
			var redisError redisclient.Error
			if !errors.As(commandError, &redisError) {
				problem := operationError(commandError)
				return nil, fiber.NewError(problem.Code, fmt.Sprintf("%s Stopped after %d command responses; the last command may have completed. Inspect affected keys before retrying.", problem.Message, len(result.Replies)))
			}
			reply = Reply{Type: "error", Text: limitedText(commandError.Error(), 4096)}
		} else {
			reply = typedReply(value, &budget, &elements, 0)
		}
		result.Replies = append(result.Replies, reply)
		if budget <= 0 || elements <= 0 {
			result.Truncated = true
			break
		}
	}
	result.Milliseconds = strconv.FormatInt(time.Since(start).Milliseconds(), 10)
	result.Processed = strconv.Itoa(len(result.Replies))
	result.Requested = strconv.Itoa(len(commands))
	result.Discarded = transaction && len(result.Replies) < len(commands)
	return result, nil
}

func typedReply(value any, budget, elements *int, depth int) Reply {
	if depth > 32 || *budget <= 0 || *elements <= 0 {
		return Reply{Type: "truncated", Truncated: true}
	}
	*elements--
	switch held := value.(type) {
	case nil:
		return Reply{Type: "null"}
	case int64:
		text := strconv.FormatInt(held, 10)
		*budget -= len(text)
		return Reply{Type: "integer", Text: text}
	case uint64:
		text := strconv.FormatUint(held, 10)
		*budget -= len(text)
		return Reply{Type: "integer", Text: text}
	case float64:
		text := strconv.FormatFloat(held, 'g', -1, 64)
		*budget -= len(text)
		return Reply{Type: "double", Text: text}
	case bool:
		return Reply{Type: "boolean", Text: strconv.FormatBool(held)}
	case string:
		return byteReply([]byte(held), budget)
	case []byte:
		return byteReply(held, budget)
	case error:
		text := limitedText(held.Error(), 4096)
		*budget -= len(text)
		return Reply{Type: "error", Text: text}
	case []any:
		result := Reply{Type: "array", Items: make([]Reply, 0, min(len(held), *elements))}
		for _, item := range held {
			if *budget <= 0 || *elements <= 0 {
				result.Truncated = true
				break
			}
			result.Items = append(result.Items, typedReply(item, budget, elements, depth+1))
		}
		return result
	default:
		text := fmt.Sprintf("Unsupported reply type %T", value)
		return Reply{Type: "unsupported", Text: text}
	}
}

func byteReply(value []byte, budget *int) Reply {
	limit := max(0, *budget/2)
	truncated := len(value) > limit
	if truncated {
		value = value[:limit]
	}
	encoded := redisdata.EncodeBytes(value)
	*budget -= len(encoded.Data) + len(value)
	return Reply{Type: "bytes", Bytes: &encoded, Truncated: truncated}
}

func limitedText(text string, limit int) string {
	if len(text) > limit {
		return text[:limit] + "…"
	}
	return text
}
