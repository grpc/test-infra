The prometheus and servicemonitor configurations are taken from 
[istio/tools](https://github.com/istio/tools).

Steps to generate the configuration:

* The base configurations and script used came from 
[tools/perf/istio-install/](https://github.com/istio/tools/tree/master/perf/istio-install#istio-setup)
* Leave the part related to istio out, only take the portion related 
to prometheus operator and creating service accounts, clusterroles and clusterrolebinding.
* Change the namespaces.