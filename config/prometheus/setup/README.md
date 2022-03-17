# Prometheus installation

[Prometheus Operator](https://github.com/prometheus-operator/prometheus-operator)
is used to install Prometheus. A prometheus operator is a custom controller
helps managing the Prometheus instance.

## Deploy a Prometheus Operator

The crds.yaml and install-prometheus-operator.yaml is taken from
[bundle.yaml](https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.55.0/bundle.yaml)
from
[Prometheus-operator](https://github.com/prometheus-operator/prometheus-operator)
with replacement on `namespace` and `nodeselector` fields where applies.

To deploy a prometheus operator, use the following commands.

Create `prometheus` namespace:

```shell
kubectl create namespace prometheus
```

Install all required
[customer resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
on the cluster:

```shell
kubectl create -f crds.yaml
```

Deploy a prometheus operator on the cluster:

```shell
kubectl apply -f install-prometheus-operator.yaml
```

## Create a Prometheus instance

After prometheus operator is ready, the next step is to apply the configuration
of Prometheus and ServiceMonitors. The prometheus operator will create the
configure resources automatically.

The install-prometheus.yaml and servicemonitor.yaml configurations are taken
from [istio/tools](https://github.com/istio/tools/) with configration related to
istio left out.

Install prometheus:

```shell
kubectl apply -f install-prometheus.yaml
```

Install ServiceMonitor:

```shell
kubectl apply -f servicemonitor.yaml
```
