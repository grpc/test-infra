# Fake control plane

The directory containing a fake control plane used in running PSM test. The fake
control plane is running in a container along the side of the test client.

## xDS server

The xDS server is the fake control plane used in running PSM performance test.
The xDS server first creates a piece of resource snapshot from the configuration
files. Note that the configuration files here only contains placeholders for the
address and port for backend services. After validation of the configuration
files supplied, a gRPC server for endpoint update is started. The endpoint
update server listens on test backends' the podIP and test port. On receiving
the backend servers' podIP and test port, the resource of the xDS server will be
updated accordingly.

After filling in the actual backend services, the xDS server starts listening on
request and server the configuration created through above steps.

## Options for starting the xDS server

The following example start a xDS server with the initial endpoint update server
listening on port 18005, serving resource snapshot created based on
`config/default_config.json`. The server target string passed to gRPC proxyless
client needs to match the name of the listener resource that handling the
requests, the listener name is primarily supplied through the
`default_config.json` file, here the flag is used to validate that at lease one
of the listener has the name of the server target string.

```shell
go run main.go
   -default-config-path config/default_config.json \
   -endpoint-update-port 18005    \
   -psm-target-string default_testGrpcListenerName

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
  [Custom configuration of xDS server](#Custom-configuration-of-xDS-server)

- -psm-target-string

  This field in only for validation. The listener name that serves non-proxied
  client (gRPC proxyless). The listener name has to match the
  server-target-string in xds:///server-target-string, which is passed as target
  for test client. The source of the truth for name of the listener's names are
  from the configuration files, the flag here is to make sure that at lease one
  of the listener resource has the required name.

- -sidecar-listener-port This field in only for validation. The listener port
  that sidecar proxy is listeneing on. The traffic wish to go through envoy
  should be send to this port. The default value of this field is `10000`.

- -validation-only This flag allows user to validate the custome resource
  configuration. The default value of this filed is `false`

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
   -validation-only true
```

Currently, the `sidecar-listener-port` and `psm-target-string` fields are for
validation only, will not change the actual config.
