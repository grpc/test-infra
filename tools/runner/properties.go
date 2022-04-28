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

// LogInfo contains infomation for a log entry.
type LogInfo struct {
	podNameElement string
	containerName  string
	logPath        string
}

// NewLogInfo creates a pointer of new LogInfo object.
func NewLogInfo(podNameElement string, containerName string, logPath string) *LogInfo {
	return &LogInfo{
		podNameElement: podNameElement,
		containerName:  containerName,
		logPath:        logPath,
	}
}

// PodLogProperties creates container log property name to
// container log link map.
func PodLogProperties(logInfos []*LogInfo, logURLPrefix string) map[string]string {
	properties := make(map[string]string)
	for _, logInfo := range logInfos {
		prefix := []string{logInfo.podNameElement, "name", "log", logInfo.containerName}
		podLogPropertyKey := strings.Join(prefix, ".")
		url := logURLPrefix + logInfo.logPath
		properties[podLogPropertyKey] = url
	}
	return properties
}

// PodNameElement trim off the loadtest name from the pod name,
// and return only the element of pod name such as client-0,
// driver-0 and server-0.
func PodNameElement(podName, loadTestName string) string {
	prefix := fmt.Sprintf("%s-", loadTestName)
	podNameElement := strings.TrimPrefix(podName, prefix)
	return podNameElement
}

// PodNameProperties creates pod property name to pod name map.
func PodNameProperties(pods []*corev1.Pod, loadTestName string) map[string]string {
	properties := make(map[string]string)
	for _, pod := range pods {
		prefix := []string{PodNameElement(pod.Name, loadTestName), "name"}
		podNamePropertyKey := strings.Join(prefix, ".")
		properties[podNamePropertyKey] = pod.Name
	}

	return properties
}
