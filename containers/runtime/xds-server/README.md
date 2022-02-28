# Fake control plane

The directory containing a fake control plane used in running PSM test. The fake
control plane is running in a container along the side of the test client.

## xDS server

The xDS server is the fake control plane used in running PSM performance test.
The xDS server first creates a piece of resource snapshot from the configuration
files. Note that the configuration files here only contains placeholders for the
address and port for backend services. After validation of the configuration
files supplied, a gRPC server for test update is started. The test update server
communicate with the Driver's ready container, it listens for test backends'
podIP, test port and if the test is a proxied test.

For proxied test, the xds server will remove all api_listneners and only serves
the socket listener to the Envoy. This is a work around to ensure that the
resources as a whole pass the validation of Envoy.

Based on whether the current test is a proxied test or not, the xDS server will
construct the corresponding target string and return to the driver's ready
container, the Driver's ready container will then keep the information for
Driver's run container to use. This target string is used to override the test
target that the driver sends to client for conducting the tests.

The test update server will be closed after the communication.

After filling in the actual backend services, the xDS server starts listening on
request and server the configuration created through above steps.

## Options for starting the xDS server

The following example start an xDS server with the initial endpoint update
server listening on port 18005, serving resource snapshot created based on
`config/default_config.json`. The server target string passed to gRPC proxyless
client needs to match the name of the listener resource that handling the
requests, the listener name is primarily supplied through the
`default_config.json` file, here the flag is used to validate that at lease one
of the listener has the name of the server target string.

```shell
go run main.go
   -default-config-path config/default_config.json \
   -endpoint-update-port 18005    \
   -psm-target-string defaultApiListener

```

The binary main.go takes the following options:

- -xds-server-port

  The server port of the xds server listening on. This port matches the server
  port in the field `server_uri` in `../gRPC_bootstrap/bootstrap.json` file and
  `port_value` in the `static_resources` section in `../envoy/envoy.yaml` file.
  This filed has the defalt value of `18000`.

- -endpoint-update-port

  The port that endpoint update server is listening on. The default value of the
  port value is `18005`.

- -node-ID

  The node ID that this snapshot of the resource can be served to. The nodeID
  has to match `id` field in `node` section in
  `../gRPC_bootstrap/bootstrap.json` file and `id` field in `node` section in
  `../envoy/envoy.yaml` file. The default value of this field is `test_id`.

- -default-config-path

  The path of default configuration JSON file that the resource snapshot based
  on. The default value of the field is `config/default_config.json`.

- -custom-config-path

  The path of the user supplied configuration JSON file. No default value for
  this field, for more information check section:
  [Custom configuration of xDS server](#custom-configuration-of-xds-server)

- -validate-only

  This flag allows user to validate the custom resource configuration,
  especially for case that a custom configuration is submitted. The default
  value of this filed is `false`, means the program only validate the resources
  configurations and will not start any servers.

- -path-to-bootstrap

  Non-proxy clients requires a bootstrap file to help the xds client understands
  where is the xds server. Since the bootstrap file is only needed when running
  PSM tests, the `bootstrap.json` is included in the xDS server image to avoid
  interference with regular benchmark. To be able to provide this file to the
  test clients, xDS server needs to move the `bootstrap.json` file to a shared
  volume. This flag allows user to provide the path of the `bootstrap.json`, if
  there is nothing passed through the flag, the xds server will skip this
  function.

## Custom configuration of xDS server

Note that the default configuration file: `config/default_config.json` here used
ADS for all resource update, the bootstrap files for Envoy proxy has matched
this configuration with using ADS in both `cds_config` and `lds_config`.

The user supplied configuration can be a subset of the required resource
collection, the resources not altered by users will default to
`config/default_config.json`. After create the custom configuration, user can
use the following command to check if the custome resource configuration if
ready be served:

```shell
go run main.go
   -default-config-path config/default_config.json \
   -validate-only true
```

Currently, the `sidecar-listener-port` and `psm-target-string` fields are for
validation only, will not change the actual config.
