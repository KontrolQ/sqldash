package redisdata

import (
	"context"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/redis/go-redis/v9"
)

func (self *snapshot) loadNativeMetadata(ctx context.Context, transaction *redis.Tx, key []byte, options ReadOptions) error {
	command := ""
	switch self.kind {
	case "MBbloom--":
		command = "BF.INFO"
	case "MBbloomCF":
		command = "CF.INFO"
	case "CMSk-TYPE":
		command = "CMS.INFO"
	case "TopK-TYPE":
		command = "TOPK.INFO"
	case "TDIS-TYPE":
		command = "TDIGEST.INFO"
	case "vectorset":
		command = "VINFO"
	case "TSDB-TYPE":
		// TS.INFO invalidates WATCH even when it does not change the value.
		return nil
	}
	if command == "" {
		return nil
	}
	response, err := transaction.Do(ctx, command, key).Slice()
	if err != nil {
		message := err.Error()
		if strings.HasPrefix(message, "NOPERM") || strings.HasPrefix(message, "ERR unknown command") {
			return nil
		}
		return commandError(err)
	}
	budget := options.MaxBytes
	if !boundedReply(response, &budget) {
		return LimitExceeded
	}
	if len(response)%2 != 0 {
		return Unsupported
	}
	for index := 0; index < len(response); index += 2 {
		name, present := response[index].(string)
		if !present || !utf8.ValidString(name) {
			continue
		}
		content := ""
		switch value := response[index+1].(type) {
		case string:
			if !utf8.ValidString(value) {
				continue
			}
			content = value
		case int64:
			content = strconv.FormatInt(value, 10)
		default:
			continue
		}
		self.nativeProperties = append(self.nativeProperties, NativeProperty{Name: name, Value: content})
	}
	return nil
}
