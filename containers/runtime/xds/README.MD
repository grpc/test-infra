# Fake control plane

## xDS server

The xDS server is the fake control plane used in running PSM load test. The xDS server implemented here first created a piece of resource snapshot from the configuration files, after validation of the supplied configuration, a gRPC server is started. This is a resource update server, it listens on the podIP and port of test backends. The server will be closed  

## Options for starting the xDS server

## Custom configuration of xDS server
