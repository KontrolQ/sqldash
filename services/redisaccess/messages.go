package redisaccess

import "errors"

var Refused = errors.New("the Redis connection is not authorized")
