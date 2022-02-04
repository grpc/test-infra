#!/bin/bash

# Catch SIGTERM signal when test finish to gracefuly terminate
# Envoy.
term() {
  echo "Test complete, caught SIGTERM signal, terminating Envoy!"
  kill -TERM "$PROXY" 2>/dev/null
  exit 0
}
trap term TERM

envoy -c /etc/envoy/envoy.yaml &

PROXY=$!
wait "$PROXY"
