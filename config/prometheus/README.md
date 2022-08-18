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
