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

// SaveAllLogs saves all container logs to files under a given directory.
// This function goes through every container in every pods and writes
// its log to a file, if it is not empty. Information about each saved log
// is returned as a pointer to a LogInfo object.
func SaveAllLogs(ctx context.Context, loadTest *grpcv1.LoadTest, podsGetter corev1types.PodsGetter, pods []*corev1.Pod, podLogDir string) ([]*LogInfo, error) {
	var logInfos []*LogInfo

	// Attempts to create directory. Will not error if directory already exists.
	err := os.MkdirAll(podLogDir, os.ModePerm)
	if err != nil {
		return logInfos, fmt.Errorf("failed to create pod log output directory %s: %v", podLogDir, err)
	}

	// Write logs to files.
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {

			logInfo, err := SaveLog(ctx, loadTest, podsGetter, pod, container.Name, podLogDir)
			if err != nil {
				return logInfos, fmt.Errorf("could not get log from container: %v", err)
			}

			if logInfo != nil {
				logInfos = append(logInfos, logInfo)
			}
		}
	}
	return logInfos, nil
}

// SaveLog retrieves and saves logs for a specific container.
// This function retrieves the log for a single container within a given
// pod, and writes it to a file, if it is not empty. Information about
// the saved log is returned as a pointer to a LogInfo object.
func SaveLog(ctx context.Context, loadTest *grpcv1.LoadTest, podsGetter corev1types.PodsGetter, pod *corev1.Pod, containerName string, podLogDir string) (*LogInfo, error) {
	req := podsGetter.Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{Container: containerName})
	containerLogs, err := req.Stream(ctx)
	if err != nil {
		return nil, err
	}
	defer containerLogs.Close()

	logBuffer := new(bytes.Buffer)
	logBuffer.ReadFrom(containerLogs)

	// Don't write empty buffers,
	if logBuffer.Len() == 0 {
		return nil, nil
	}

	// Open output file,
	filePath := filepath.Join(podLogDir, LogFileName(pod.Name, containerName))
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open %s for writing", filePath)
	}
	defer file.Close()

	// Write log to output file,
	_, err = io.Copy(file, logBuffer)
	file.Sync()
	if err != nil {
		return nil, fmt.Errorf("error writing to %s: %v", filePath, err)
	}

	logInfo := &LogInfo{
		PodNameElem:   PodNameElem(pod.Name, loadTest.Name),
		ContainerName: containerName,
		LogPath:       filePath,
	}

	return logInfo, nil
}

// LogFileName constructs a log file name from pod and container names.
func LogFileName(podName string, containerName string) string {
	return fmt.Sprintf("%s-%s.log", podName, containerName)
}
