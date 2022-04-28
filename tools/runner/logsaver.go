/*
Copyright 2021 gRPC authors.

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
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	corev1 "k8s.io/api/core/v1"
	corev1types "k8s.io/client-go/kubernetes/typed/core/v1"
)

// SaveLogs saves logs to files, the name of the files are in format
// pod-name-container-name.log.
// This function returns a list of pointers of LogInfo.
func SaveLogs(ctx context.Context, loadTest *grpcv1.LoadTest, pods []*corev1.Pod, podsGetter corev1types.PodsGetter, podLogDir string) ([]*LogInfo, error) {
	logInfos := []*LogInfo{}

	// Attempt to create directory. Will not error if directory already exists
	err := os.MkdirAll(podLogDir, os.ModePerm)
	if err != nil {
		return logInfos, fmt.Errorf("Failed to create pod log output directory %s: %v", podLogDir, err)
	}

	// Write logs to files
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			logBuffer, err := GetLogBuffer(ctx, pod, podsGetter, container.Name)

			if err != nil {
				return logInfos, fmt.Errorf("could not get log from pod: %s", err)
			}

			if logBuffer.Len() == 0 {
				continue
			}

			logFilePath := filepath.Join(podLogDir, fmt.Sprintf("%s-%s.log", pod.Name, container.Name))
			logInfo := NewLogInfo(PodNameElement(pod.Name, loadTest.Name), container.Name, logFilePath)

			err = writeBufferToFile(logBuffer, logFilePath)
			if err != nil {
				return logInfos, fmt.Errorf("could not write %s container in %s pod log buffer to file: %s", logInfo.containerName, pod.Name, err)
			}

			logInfos = append(logInfos, logInfo)
		}
	}
	return logInfos, nil
}

// GetLogBuffer retrieves logs from a specific container
// in the given pod and return the log buffer.
func GetLogBuffer(ctx context.Context, pod *corev1.Pod, podsGetter corev1types.PodsGetter, containerName string) (*bytes.Buffer, error) {
	req := podsGetter.Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{Container: containerName})
	containerLogs, err := req.Stream(ctx)
	if err != nil {
		return nil, err
	}
	defer containerLogs.Close()
	logBuffer := new(bytes.Buffer)
	logBuffer.ReadFrom(containerLogs)
	return logBuffer, nil
}

func writeBufferToFile(buffer *bytes.Buffer, filePath string) error {
	// Don't write empty buffers
	if buffer.Len() == 0 {
		return nil
	}

	// Open output file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("could not open %s for writing", filePath)
	}
	defer file.Close()

	// Write log to output file
	_, err = io.Copy(file, buffer)
	file.Sync()
	if err != nil {
		return fmt.Errorf("error writing to %s: %v", filePath, err)
	}

	return nil
}
