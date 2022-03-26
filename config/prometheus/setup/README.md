# Prometheus installation

[Prometheus Operator](https://github.com/prometheus-operator/prometheus-operator)
is used to install Prometheus. A prometheus operator is a custom controller
helps managing the Prometheus instance.

## Quick Start

Create `prometheus` namespace:

```shell
kubectl create namespace prometheus
```

Use following command to install Prometheus, this command applied all yaml
configurations from config/prometheus/setup/ to the cluster:

```shell

kubectl create -f config/prometheus/setup/

```

### Prometheus Operator

There are two configurations related to creating a Prometheus Operator. The
crds.yaml and install-prometheus-operator.yaml are taken from
[bundle.yaml](https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.55.0/bundle.yaml)
from
[Prometheus-operator](https://github.com/prometheus-operator/prometheus-operator)
with replacement on `namespace` and `nodeselector` fields where applies.

### Prometheus instance and ServiceMonitor instance

The install-prometheus.yaml and servicemonitor.yaml configurations are taken
from [istio/tools](https://github.com/istio/tools/) with configuration related
to Istio left out.

## Verify installation

Following command shows all resources created in prometheus namespace:

```shell

kubectl get all -n prometheus

```

You should see the following resources created, most of them is running right
away, but it would take about 50s to get statefulset.apps/prometheus-prometheus
up and running:

```shell
NAME                                       READY   STATUS    RESTARTS   AGE
pod/prometheus-operator-677cf7f876-mqm6w   1/1     Running   0          3m29s
pod/prometheus-prometheus-0                2/2     Running   0          3m25s

NAME                          TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
service/prometheus            ClusterIP   10.108.15.152   <none>        9090/TCP   3m28s
service/prometheus-operated   ClusterIP   None            <none>        9090/TCP   3m25s
service/prometheus-operator   ClusterIP   None            <none>        8080/TCP   3m29s

NAME                                  READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/prometheus-operator   1/1     1            1           3m29s

NAME                                             DESIRED   CURRENT   READY   AGE
replicaset.apps/prometheus-operator-677cf7f876   1         1         1       3m30s

NAME                                     READY   AGE
statefulset.apps/prometheus-prometheus   1/1     3m26s
```

After the all the resources are up and running, we can forward the port by
following command:

```shell
kubectl port-forward service/prometheus 9090:9090 -n prometheus

```

Then we can verify that Prometheus is serving metrics by navigating to its
metrics endpoint: `http://localhost:9090/graph`.
