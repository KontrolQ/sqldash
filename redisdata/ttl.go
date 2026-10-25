package redisdata

import (
	"strconv"
	"time"
)

func ttlArguments(change *TTLChange) (string, string, error) {
	if change == nil {
		return "preserve", "", nil
	}
	switch change.Mode {
	case "preserve", "persistent":
		if change.Milliseconds != "" || change.ExpiresAtMilliseconds != "" {
			return "", "", InvalidInput
		}
		return change.Mode, "", nil
	case "relative":
		value, err := strconv.ParseInt(change.Milliseconds, 10, 64)
		if err != nil || value < 0 || strconv.FormatInt(value, 10) != change.Milliseconds || change.ExpiresAtMilliseconds != "" || value > (1<<63-1)-time.Now().UnixMilli() {
			return "", "", InvalidInput
		}
		return change.Mode, change.Milliseconds, nil
	case "absolute":
		value, err := strconv.ParseInt(change.ExpiresAtMilliseconds, 10, 64)
		if err != nil || value < 0 || strconv.FormatInt(value, 10) != change.ExpiresAtMilliseconds || change.Milliseconds != "" {
			return "", "", InvalidInput
		}
		return change.Mode, change.ExpiresAtMilliseconds, nil
	default:
		return "", "", InvalidInput
	}
}
