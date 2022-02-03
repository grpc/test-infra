#!/bin/bash

_term() {
  echo "Test complete, caught SIGTERM signal, terminating Envoy!"
  kill -TERM "$proxy" 2>/dev/null
  exit 0
}

trap _term TERM

envoy -c /etc/envoy/envoy.yaml &

proxy=$!
wait "$proxy"
