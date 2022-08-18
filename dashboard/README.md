# Dashboard

This folder contains the components necessary to build and deploy a dashboard to
visualize gRPC OSS benchmarking results.

gRPC OSS benchmarks save results to [BigQuery][bigquery]. The dashboard consists
of two components:

1. A [Postgres replicator][replicator], to transfer the results to a Postgres
   database.
1. A Grafana dashboard, to display the results from the Postgres database.

These components can be built and deployed manually using the
[Makefile](Makefile) (see [manual build](#manual-build)).

Notice that the dashboard build is independent from the top-level build.

[bigquery]: https://cloud.google.com/bigquery
[replicator]: cmd/postgres_replicator/README.md

## Configuration

The configuration of the Postgres replicator is defined in a YAML file. The
default configuration is defined here, in template form:

- [config/postgres_replicator/default/config.yaml][replicatorconfig]

For more information, see [Postgres replicator][replicator].

The configuration of the Grafana dashboard is defined in a set of dashboards
specified in JSON files. The default configuration is defined here:

- [config/grafana/dashboards/default/][grafanaconfig]

The continuous integration dashboard linked from the [gRPC performance
benchmarking][benchmarking] page uses the default configuration. The variables
`REPLICATOR_CONFIG_TEMPLATE` and `DASHBOARDS_CONFIG_DIR` can be set to build
dashboards with different configurations.

[benchmarking]: https://grpc.io/docs/guides/benchmarking/
[grafanaconfig]: config/grafana/dashboards/default/
[replicatorconfig]: config/postgres_replicator/default/config.yaml

## Manual build

Several environment variables must be set before building and deploying. The
table below shows the names and values of the variables in our main dashboard:

| Variable                    | Value                                   |
| --------------------------- | --------------------------------------- |
| `BQ_PROJECT_ID`             | `grpc-testing`                          |
| `CLOUD_SQL_INSTANCE`        | `grpc-testing:us-central1:grafana-data` |
| `GCP_DATA_TRANSFER_SERVICE` | `postgres-replicator`                   |
| `GCP_GRAFANA_SERVICE`       | `grafana`                               |
| `GCP_PROJECT_ID`            | `grpc-testing`                          |
| `GRAFANA_ADMIN_PASS`        | ...                                     |
| `PG_DATABASE`               | `datasource`                            |
| `PG_PASS`                   | ...                                     |
| `PG_USER`                   | `grafana-user`                          |
| `PSM`                       | `true`                                  |

The dashboard for PSM benchmarks is only deployed if the PSM variable is set to
a non-empty value, for instance PSM=true.

Docker files that can be used to build and deploy the Postgres replicator and
Grafana dashboard are then created with the following commands:

```shell
make configure-replicator
make configure-grafana
```

## Cloud build

The continuous integration dashboard is built and deployed with [Cloud
Build][cloudbuild], using the configuration specified in
[cloudbuild.yaml](cloudbuild.yaml).

The use of Cloud Build allows the dashboard to be redeployed automatically on
configuration changes. In addition, it allows passwords such as `PG_PASS` and
`GRAFANA_ADMIN_PASS` to be stored as secrets in the cloud project.

[cloudbuild]: https://cloud.google.com/build
