/*
Copyright 2022 gRPC authors.

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

package runner

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// LogInfo contains infomation for each log file.
type LogInfo struct {
	// PodNameElem is the element added to the LoadTest name to
	// construct the pod name. Examples of PodNameElem are client-0,
	// driver-0 and server-0.
	PodNameElem string
	// ContainerName is the container's name where the log comes from.
	ContainerName string
	// LogPath is the path pointing to the log file.
	LogPath string
}

// PodLogProperties creates a map of log property keys to log path urls.
func PodLogProperties(logInfos []*LogInfo, logURLPrefix string, prefix ...string) map[string]string {
	properties := make(map[string]string)
	for _, logInfo := range logInfos {
		podLogPropertyKey := PodLogPropertyKey(logInfo, prefix...)
		logURL := logURLPrefix + logInfo.LogPath
		properties[podLogPropertyKey] = logURL
	}
	return properties
}

// PodLogPropertyKey generates the key for a pod log property.
func PodLogPropertyKey(logInfo *LogInfo, prefix ...string) string {
	key := strings.Join(append(prefix, logInfo.PodNameElem, "log", logInfo.ContainerName), ".")
	return key
}

// PodNameProperties creates a map of pod name property keys to pod names.
func PodNameProperties(pods []*corev1.Pod, loadTestName string, prefix ...string) map[string]string {
	properties := make(map[string]string)
	for _, pod := range pods {
		podNamePropertyKey := PodNamePropertyKey(PodNameElem(pod.Name, loadTestName), prefix...)
		properties[podNamePropertyKey] = pod.Name
	}

	return properties
}

// PodNameElem returns the pod name element used to construct a pod name.
// Pods within a LoadTest are distinguished by elements attached to the
// LoadTest name, such as client-0, driver-0, server-0.
func PodNameElem(podName, loadTestName string) string {
	prefix := fmt.Sprintf("%s-", loadTestName)
	podNameElem := strings.TrimPrefix(podName, prefix)
	return podNameElem
}

// PodNamePropertyKey generates the key for a pod name property.
func PodNamePropertyKey(podNameElem string, prefix ...string) string {
	key := strings.Join(append(prefix, podNameElem, "name"), ".")
	return key
}
