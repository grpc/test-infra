package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1types "k8s.io/client-go/kubernetes/typed/core/v1"
)

// LogSaver provides functionality to save pod logs to files.
type LogSaver struct {
	podsGetter corev1types.PodsGetter
}

// NewLogSaver creates a new LogSaver object.
func NewLogSaver(podsGetter corev1types.PodsGetter) *LogSaver {
	return &LogSaver{
		podsGetter: podsGetter,
	}
}

// SavePodLogs saves pod logs to files with same name as pod. It returns a map
// where pods are keys and values are the filepath of the saved log.
func (ls *LogSaver) SavePodLogs(ctx context.Context, loadTest *grpcv1.LoadTest, podLogDir string) (map[*corev1.Pod]string, error) {
	podToLogPath := make(map[*corev1.Pod]string)

	// Get pods for this test
	pods, err := ls.getTestPods(ctx, loadTest)
	if err != nil {
		return podToLogPath, err
	}

	// Attempt to create directory. Will not error if directory already exists
	err = os.MkdirAll(podLogDir, os.ModePerm)
	if err != nil {
		return podToLogPath, fmt.Errorf("Failed to create pod log output directory %s: %v", podLogDir, err)
	}

	// Write logs to files
	for _, pod := range pods {
		logFilePath := filepath.Join(podLogDir, pod.Name+".log")
		buffer, err := ls.getPodLogBuffer(ctx, pod)
		if err != nil {
			return podToLogPath, fmt.Errorf("could not get log from pod: %s", err)
		}
		err = ls.writeBufferToFile(buffer, logFilePath)
		if err != nil {
			return podToLogPath, fmt.Errorf("could not write pod log buffer to file: %s", err)
		}
		podToLogPath[pod] = logFilePath
	}
	return podToLogPath, nil
}

// getTestPods retrieves the pods associated with a LoadTest.
func (ls *LogSaver) getTestPods(ctx context.Context, loadTest *grpcv1.LoadTest) ([]*corev1.Pod, error) {
	podLister := ls.podsGetter.Pods(metav1.NamespaceAll)

	// Get a list of all pods
	podList, err := podLister.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.New("Failed to fetch list of pods")
	}

	// Get pods just for this specific test
	testPods := status.PodsForLoadTest(loadTest, podList.Items)
	return testPods, nil
}

func (ls *LogSaver) getPodLogBuffer(ctx context.Context, pod *corev1.Pod) (*bytes.Buffer, error) {
	req := ls.podsGetter.Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{})
	driverLogs, err := req.Stream(ctx)
	if err != nil {
		return nil, err
	}
	defer driverLogs.Close()

	logBuffer := new(bytes.Buffer)
	logBuffer.ReadFrom(driverLogs)

	return logBuffer, nil
}

func (ls *LogSaver) writeBufferToFile(buffer *bytes.Buffer, filePath string) error {
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
