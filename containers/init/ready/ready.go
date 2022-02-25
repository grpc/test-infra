/*
Copyright 2020 gRPC authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"time"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	grpcclientset "github.com/grpc/test-infra/clientset"
	testconfig "github.com/grpc/test-infra/config"
	"github.com/grpc/test-infra/kubehelpers"
	pb "github.com/grpc/test-infra/proto/endpointupdater"
	"github.com/grpc/test-infra/status"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	grpcstatus "google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// TimeoutEnv is the name of the environment variable that will contain the
// maximum amount of time to wait for pods to become ready.
const TimeoutEnv = "READY_TIMEOUT"

// DefaultTimeout specifies the amount of time to wait for ready pods if the
// environment variable specified by the TimeoutEnv constant is not set.
const DefaultTimeout = 25 * time.Minute

// OutputFileEnv is the optional name of the file where the executable should
// write a comma-separated list of IP addresses. If this environment variable is
// unset, DefaultOutputFile will be used as the default.
const OutputFileEnv = "READY_OUTPUT_FILE"

// OutputMetadataEnv is the optional name of the file where the executable should
// write all metadata.
const OutputMetadataEnv = "METADATA_OUTPUT_FILE"

// OutputNodeInfoEnv is the optional name of the file where the executable should
// write the node information.
const OutputNodeInfoEnv = "NODE_INFO_OUTPUT_FILE"

// DefaultOutputFile is the name of the default file where the executable should
// write the comma-separated list of IP addresses.
const DefaultOutputFile = "/tmp/loadtest_workers"

// DefaultMetadataOutputFile is the name of the default file where the executable should
// write the metadata.
const DefaultMetadataOutputFile = "/tmp/metadata.json"

// DefaultNodeInfoOutputFile is the name of the default file where the executable should
// write the node infomation.
const DefaultNodeInfoOutputFile = "/tmp/node_info.json"

// DefaultServerTargetOverrideOutputFile is the name of the defalt file where the
// executable should write the string to override the test target, this is only
// used for PSM tests.
const DefaultServerTargetOverrideOutputFile = testconfig.ReadyMountPath + "/server_target_override"

// DefaultDriverPort is the default port for communication between the driver
// and worker pods. When another port could not be found on a pod, this port is
// included in the addresses returned by the WaitForReadyPods function.
const DefaultDriverPort int32 = 10000

// defaultConnectionTimeout specifies the default maximum allowed duration of a RPC call .
const defaultConnectionTimeout = 20 * time.Second

// KubeConfigEnv is the name of the environment variable that may contain a
// path to a kubeconfig file. This environment variable does not need to be set
// when the container runs on a node in a Kubernetes cluster.
const KubeConfigEnv = "KUBE_CONFIG"

// pollInterval specifies the amount of time between subsequent requests to the
// Kubernetes API for a list of pods.
const pollInterval = 3 * time.Second

// PodLister lists pods known to a Kubernetes cluster.
type PodLister interface {
	List(context.Context, metav1.ListOptions) (*corev1.PodList, error)
}

// LoadTestGetter fetches a load test with a specific name.
type LoadTestGetter interface {
	Get(context.Context, string, metav1.GetOptions) (*grpcv1.LoadTest, error)
}

// NodeInfo contains pod name, pod IP and node name in which the pod reside for one worker or driver.
type NodeInfo struct {
	Name     string
	PodIP    string
	NodeName string
}

// NodesInfo contains NodeInfo for all pods included in a load test.
type NodesInfo struct {
	Driver  NodeInfo
	Servers []NodeInfo
	Clients []NodeInfo
}

// isPodReady returns true if the pod has been assigned an IP address and all of
// its containers are ready.
func isPodReady(pod *corev1.Pod) bool {
	spec := &pod.Spec
	status := &pod.Status

	if status.PodIP == "" {
		return false
	}

	if len(spec.Containers) != len(status.ContainerStatuses) {
		return false
	}

	for _, cstat := range status.ContainerStatuses {
		if !cstat.Ready {
			return false
		}
	}

	return true
}

// findDriverPort searches through a pod's list of containers and their ports to
// locate a port named "driver". If discovered, its number is returned. If not
// found, DefaultDriverPort is returned.
func findDriverPort(pod *corev1.Pod) int32 {
	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
			if port.Name == "driver" {
				return port.ContainerPort
			}
		}
	}

	return DefaultDriverPort
}

// WaitForReadyPods blocks until all worker pods within the load test are ready.
// It accepts a context, allowing a timeout or deadline to be specified. When
// all pods are ready, it returns a slice of strings with the IP address and
// driver port for each matching pod. server pod would come before client pod.
//
// The driver port is determined by searching the pod for a container with a TCP
// port named "driver". If there is no port named "driver" exposed on any of the
// matching pod's containers, the value of DefaultDriverPort will be used.
//
// If the timeout is exceeded or there is a problem communicating with the
// Kubernetes API, an error is returned.
func WaitForReadyPods(ctx context.Context, ltg LoadTestGetter, pl PodLister, testName string) ([]string, *NodesInfo, error) {
	var loadtest *grpcv1.LoadTest
	var clientPodAddresses []string
	var serverPodAddresses []string
	var nodesInfo NodesInfo
	clientMatchCount := 0
	serverMatchCount := 0
	driverMatched := false
	timeoutsEnabled := true
	matchingPods := make(map[string]bool)

	deadline, ok := ctx.Deadline()
	if !ok {
		timeoutsEnabled = false
		log.Printf("no timeout is set; this could block forever")
	}

	for {
		if timeoutsEnabled && time.Now().After(deadline) {
			return nil, nil, errors.Errorf("deadline exceeded (%v)", deadline)
		}
		if loadtest == nil {
			l, err := ltg.Get(ctx, testName, metav1.GetOptions{})
			if err != nil {
				log.Printf("failed to fetch loadtest: %v", err)
				time.Sleep(pollInterval)
				continue
			}
			loadtest = l
			for range loadtest.Spec.Clients {
				clientPodAddresses = append(clientPodAddresses, "")
			}
			for range loadtest.Spec.Servers {
				serverPodAddresses = append(serverPodAddresses, "")
			}
		}
		podList, err := pl.List(ctx, metav1.ListOptions{})
		if err != nil {
			log.Fatalf("failed to fetch list of pods: %v", err)
		}
		ownedPods := status.PodsForLoadTest(loadtest, podList.Items)
		for _, pod := range ownedPods {
			if pod.Labels[testconfig.RoleLabel] == testconfig.DriverRole {
				if !driverMatched && pod.Status.PodIP != "" {
					nodesInfo.Driver = NodeInfo{
						Name:     pod.Name,
						PodIP:    pod.Status.PodIP,
						NodeName: pod.Spec.NodeName,
					}
					driverMatched = true
				}
				continue
			}
			if !isPodReady(pod) {
				continue
			}
			if _, alreadyMatched := matchingPods[pod.Name]; alreadyMatched {
				continue
			}
			matchingPods[pod.Name] = true
			ip := pod.Status.PodIP
			driverPort := findDriverPort(pod)
			if pod.Labels[testconfig.RoleLabel] == testconfig.ServerRole {
				serverPodAddresses[serverMatchCount] = net.JoinHostPort(ip, fmt.Sprint(driverPort))
				nodesInfo.Servers = append(nodesInfo.Servers, NodeInfo{
					Name:     pod.Name,
					PodIP:    ip,
					NodeName: pod.Spec.NodeName,
				})
				serverMatchCount++
			} else {
				clientPodAddresses[clientMatchCount] = net.JoinHostPort(ip, fmt.Sprint(driverPort))
				nodesInfo.Clients = append(nodesInfo.Clients, NodeInfo{
					Name:     pod.Name,
					PodIP:    ip,
					NodeName: pod.Spec.NodeName,
				})
				clientMatchCount++
			}
		}

		if clientMatchCount == len(clientPodAddresses) && serverMatchCount == len(serverPodAddresses) && driverMatched {
			break
		}

		time.Sleep(pollInterval)
	}
	podAddresses := append(serverPodAddresses, clientPodAddresses...)
	return podAddresses, &nodesInfo, nil
}

// communicateWithEachClient takes a client IP, a list of server IP plus its
// test port and a boolean value indicates if the test is a proxied test. The
// function communicates with the given client's xds server through a RPC
// including information such as the full list of the server IP plus its test
// port and the boolean value indicates the PSM test type. In the response of
// the RPC, communicateWithEachClient gets back the target string will be used
// in the loadtest.After the communication, the function starts a separate
// goroutine to close the test update server on the xds server container. Then
// the function returns the target string as an return value.
func communicateWithEachClient(clientIP string, targets []*pb.Endpoint, isProxied bool) (string, error) {
	var psmServerTargetOverride string
	dialTarget := net.JoinHostPort(clientIP, fmt.Sprint(testconfig.ServerUpdatePort))
	conn, err := grpc.Dial(dialTarget, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewTestUpdaterClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), defaultConnectionTimeout)
	defer cancel()
	reply, err := c.UpdateTest(ctx, &pb.TestUpdateRequest{Endpoints: targets, IsProxied: isProxied})
	if err != nil {
		statusCode, _ := grpcstatus.FromError(err)
		log.Print(statusCode.Details()...)
		log.Fatalf("could not connect to test update server: %v", err)
	}
	psmServerTargetOverride = reply.PsmServerTargetOverride

	log.Printf("all backend targets has been communicated to client %v", clientIP)

	go func() {
		log.Printf("stopping test update server on client %v", clientIP)
		if _, err := c.QuitTestUpdateServer(ctx, &pb.Void{}); err != nil {
			statusCode, _ := grpcstatus.FromError(err)
			log.Print(statusCode.Details()...)
		}
	}()
	return psmServerTargetOverride, nil
}

func buildEndpoints(serverNodes []NodeInfo, psmTestServerPort uint32) []*pb.Endpoint {
	var targets []*pb.Endpoint
	for _, serverNode := range serverNodes {
		targets = append(targets, &pb.Endpoint{
			IpAddress: serverNode.PodIP,
			Port:      psmTestServerPort,
		})
	}
	return targets
}

func main() {
	var err error
	timeout := DefaultTimeout
	timeoutStr, ok := os.LookupEnv(TimeoutEnv)
	if ok {
		timeout, err = time.ParseDuration(timeoutStr)
		if err != nil {
			log.Fatalf("failed to parse $%s: %v", TimeoutEnv, err)
		}
	}

	schemebuilder := runtime.NewSchemeBuilder(func(scheme *runtime.Scheme) error {
		scheme.AddKnownTypes(grpcv1.GroupVersion,
			&grpcv1.LoadTest{},
			&grpcv1.LoadTestList{},
		)
		metav1.AddToGroupVersion(scheme, grpcv1.GroupVersion)
		return nil
	})

	config, err := rest.InClusterConfig()
	if err != nil {
		if err != rest.ErrNotInCluster {
			log.Fatalf("failed to connect within cluster: %v", err)
		}

		kubeConfigFile, ok := os.LookupEnv(KubeConfigEnv)
		if !ok {
			log.Fatalf("could not find kubenetes config file")
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeConfigFile)
		if err != nil {
			log.Fatalf("failed to construct config for path %q: %v", kubeConfigFile, err)
		}
	}

	schemebuilder.AddToScheme(clientgoscheme.Scheme)
	scheme := clientgoscheme.Scheme
	types := scheme.AllKnownTypes()
	_ = types

	grpcClientset, err := grpcclientset.NewForConfig(config)
	if err != nil {
		log.Fatalf("failed to create a grpc clientset: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("failed to connect with implicit kubeconfig: %v", err)
	}

	outputFile := DefaultOutputFile
	outputFileOverride, ok := os.LookupEnv(OutputFileEnv)
	if ok {
		outputFile = outputFileOverride
	}

	outputNodeInfoFile := DefaultNodeInfoOutputFile
	outputNodeInfoFileOverride, ok := os.LookupEnv(OutputNodeInfoEnv)
	if ok {
		outputNodeInfoFile = outputNodeInfoFileOverride
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	log.Printf("Waiting for ready pods")
	podIPs, nodesInfo, err := WaitForReadyPods(ctx, grpcClientset.LoadTestV1().LoadTests(corev1.NamespaceDefault), clientset.CoreV1().Pods(metav1.NamespaceAll), os.Args[1])
	if err != nil {
		log.Fatalf("failed to wait for ready pods: %v", err)
	}

	log.Printf("all pods ready")
	workerFileBody := strings.Join(podIPs, ",")
	ioutil.WriteFile(outputFile, []byte(workerFileBody), 0777)

	test, err := grpcClientset.LoadTestV1().LoadTests(corev1.NamespaceDefault).Get(ctx, os.Args[1], metav1.GetOptions{})
	if err != nil {
		log.Fatalf("failed to fetch loadtest: %v", err)
	}

	if clientSpecValid, err := kubehelpers.IsClientsSpecValid(&test.Spec.Clients); !clientSpecValid {
		log.Fatalf("validation failed in checking clients' spec: %v", err)
	}

	if kubehelpers.IsPSMTest(&test.Spec.Clients) {
		log.Printf("running PSM test, prepare to send backends information and test type to xds server")

		endpoints := buildEndpoints(nodesInfo.Servers, uint32(testconfig.ServerPort))

		isProxied := kubehelpers.IsProxiedTest(&test.Spec.Clients)
		if isProxied {
			log.Printf("running proxied test")
		} else {
			log.Printf("running proxyless test")
		}

		psmTargetOverride := ""
		for _, clientNode := range nodesInfo.Clients {

			currentPSMTargetOverride, err := communicateWithEachClient(clientNode.PodIP, endpoints, isProxied)
			if err != nil {
				log.Fatalf("failed to communicate backend endpoints to client %v: %v", clientNode.Name, err)
			}

			if psmTargetOverride == "" {
				psmTargetOverride = currentPSMTargetOverride
			} else {
				if psmTargetOverride != currentPSMTargetOverride {
					log.Fatalf("not all client have the same server target")
				}
			}
		}

		if psmTargetOverride != "" {
			if err := ioutil.WriteFile(DefaultServerTargetOverrideOutputFile, []byte(psmTargetOverride), 0777); err != nil {
				log.Fatalf("failed to write PSM server target: %v to %v", psmTargetOverride, DefaultServerTargetOverrideOutputFile)
			}
			log.Printf("write PSM server target: %v to %v\n", psmTargetOverride, DefaultServerTargetOverrideOutputFile)
		} else {
			log.Fatalf("failed to obtain PSM server target")
		}

		if isProxied {
			_, envoyListenerPort, err := net.SplitHostPort(psmTargetOverride)
			if err != nil {
				log.Fatalf("failed to obtain socket listener port")
			}
			readyEnvoy := make(map[string]bool)
			startTime := time.Now()
			for {
				if time.Now().Sub(startTime) >= DefaultTimeout {
					log.Fatalf("timeout exceed")
				}

				for _, clientNode := range nodesInfo.Clients {
					if readyEnvoy[clientNode.PodIP] {
						continue
					}

					curEnvoyListener := net.JoinHostPort(clientNode.PodIP, envoyListenerPort)
					conn, err := net.DialTimeout("tcp", curEnvoyListener, timeout)
					if err != nil {
						log.Printf("Envoy on %v is not ready: %v", clientNode.PodIP, err)
					}
					if conn != nil {
						defer conn.Close()
						readyEnvoy[clientNode.PodIP] = true
						log.Printf("Envoy on %v: %v is ready. \n", clientNode.Name, clientNode.PodIP)
					}

					time.Sleep(pollInterval)
				}

				if len(readyEnvoy) == len(nodesInfo.Clients) {
					log.Printf("all Envoy sidecars are fully functioning")
					break
				}
			}
		}
	}

	outputMetadataFile := DefaultMetadataOutputFile
	outputMetadataFileOverride, ok := os.LookupEnv(OutputMetadataEnv)
	if ok {
		outputMetadataFile = outputMetadataFileOverride
	}

	metaDataSet := test.ObjectMeta
	metaDataBody, err := json.Marshal(metaDataSet)
	if err != nil {
		log.Fatalf("failed to marshal metaData for loadtest %s: %v", test.Name, err)
	}
	ioutil.WriteFile(outputMetadataFile, metaDataBody, 0777)

	nodeInfoFileBody, err := json.Marshal(*nodesInfo)
	if err != nil {
		log.Fatalf("failed to marshal nodes information for loadtest %s: %v", test.Name, err)
	}
	ioutil.WriteFile(outputNodeInfoFile, nodeInfoFileBody, 0777)
}
