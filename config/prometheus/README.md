# Prometheus installation

[Prometheus](https://prometheus.io) is used to monitor CPU and memory
utilization in [PSM benchmarks](../../README.md#psm-benchmarks).

[Prometheus Operator](https://github.com/prometheus-operator/prometheus-operator)
is used to install Prometheus. A prometheus operator is a custom controller that
manages the Prometheus instance.

The usual way to install Prometheus Operator is through the Makefile. See the
section on [Deploying Prometheus](../../doc/deployment.md#deploying-prometheus)
in the deployment guide.

## Configuration details

### Prometheus Operator

There are two configuration files related to Prometheus Operator,
[crds.yaml](crds/bases/crds.yaml) and
[install-prometheus-operator.yaml](install-prometheus-operator.yaml). These
files are taken from
[bundle.yaml](https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.55.0/bundle.yaml),
in the Prometheus Operator repository, with the fields `namespace` and
`nodeSelector` replaced where applicable.

### Prometheus instance and ServiceMonitor instance

The files [install-prometheus.yaml](install-prometheus.yaml) and
[servicemonitors.yaml](servicemonitors.yaml) are taken from
[istio/tools](https://github.com/istio/tools/), with configuration related to
Istio left out.

### Prometheus scrape interval

The scrape interval (`scrape_interval`) determines how often Prometheus scrapes
targets. The default value for this interval is
[1m](https://prometheus.io/docs/prometheus/latest/configuration/configuration/).
We change this interval in our configuration to 1s, since our tests typically
last only 30s. See <https://github.com/grpc/test-infra/pull/325> for details.

### Delay to query

We add 20s delay before collecting data from Prometheus, in the tests that use
Prometheus. This delay allows data to become available
([cAdvisor housekeeping interval](https://github.com/google/cadvisor/blob/master/docs/runtime_options.md#housekeeping))
and pulled by Prometheus
([Prometheus scrape interval](https://github.com/grpc/test-infra/pull/325)). See
details in <https://github.com/grpc/test-infra/pull/330>.
