#!/bin/sh
set -e

(
  cd "$(dirname "$0")" 
  go build -o /tmp/http-server-go app/*.go
)

exec /tmp/http-server-go "$@"
