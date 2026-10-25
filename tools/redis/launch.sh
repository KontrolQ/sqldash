#!/bin/sh
set -eu

exec /opt/redis-runtime/loader --inhibit-cache \
    --library-path /opt/redis-runtime/lib \
    /opt/redis-runtime/bin/redis-server "$@"
