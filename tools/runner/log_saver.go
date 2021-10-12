package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	clientset "github.com/grpc/test-infra/clientset"
	"github.com/grpc/test-infra/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LogSaver provides functionality to save pod logs to files.
type LogSaver struct {
	clientset clientset.GRPCTestClientset
}

// NewLogSaver creates a new LogSaver object.
func NewLogSaver(clientset clientset.GRPCTestClientset) *LogSaver {
	return &LogSaver{
		clientset: clientset,
	}
}

// SavePodLogs saves pod logs to files with same name as pod.
func (ls *LogSaver) SavePodLogs(ctx context.Context, loadTest *grpcv1.LoadTest, podLogDir string) error {
	// Get pods for this test
	pods, err := ls.getTestPods(ctx, loadTest)
	if err != nil {
		return err
	}

	// Attempt to create directory. Will not error if directory already exists
	err = os.MkdirAll(podLogDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Failed to create pod log output directory %s: %v", podLogDir, err)
	}

	// Write logs to files
	errorOccurred := false
	for _, pod := range pods {
		logFilePath := filepath.Join(podLogDir, pod.Name+".log")
		err = ls.writePodLogToFile(ctx, pod, logFilePath)
		if err != nil {
			errorOccurred = true
		}
	}
	if errorOccurred {
		return errors.New("One or more log files could not be written")
	}
	return nil
}

// getTestPods retrieves the pods associated with a LoadTest.
func (ls *LogSaver) getTestPods(ctx context.Context, loadTest *grpcv1.LoadTest) ([]*corev1.Pod, error) {
	podLister := ls.clientset.CoreV1().Pods(metav1.NamespaceAll)

	// Get a list of all pods
	podList, err := podLister.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.New("Failed to fetch list of pods")
	}

	// Get pods just for this specific test
	testPods := status.PodsForLoadTest(loadTest, podList.Items)
	return testPods, nil
}

// writePodLogToFile writes a single pod's logs to a file.
func (ls *LogSaver) writePodLogToFile(ctx context.Context, pod *corev1.Pod, logFilePath string) error {
	// Open log stream
	req := ls.clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{})
	driverLogs, err := req.Stream(ctx)
	if err != nil {
		return fmt.Errorf("Could not open log stream for pod: %s", pod.Name)
	}
	defer driverLogs.Close()

	// Open output file
	logFile, err := os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("Could not open %s for writing", logFilePath)
	}
	defer logFile.Close()

	// Write log to output file
	_, err = io.Copy(logFile, driverLogs)
	logFile.Sync()
	if err != nil {
		return fmt.Errorf("Error writing to %s: %v", logFilePath, err)
	}

	return nil
}
