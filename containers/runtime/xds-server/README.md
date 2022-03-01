# Fake control plane

This directory contains a fake control plane used to run
[PSM benchmarks](../../../README.md#psm-benchmarks). The fake control plane runs
in a container as part of the test client pod.

## xDS server

The xDS server first reads and validates a resource fragment from a
configuration file. This fragment is a template, requiring the test's backend
server details to be filled in.

The xDS server then starts an endpoint update server that listens for a
[message](../../../proto/endpointupdater/endpoint.proto) from the test driver's
ready container. The message contains the test's backend server IP and port, and
whether the test should be proxied or proxyless. The update server responds to
this message with the correct target string to be passed to the driver's run
container. The update server shuts down after responding to the message.

For a proxied test, the xDS server will remove all api_listeners from its
configuration, and only serve the socket listener to the Envoy sidecar.

After filling in the actual backend service addresses, the xDS server starts
listening for requests and serves the configuration created through the above
steps.
