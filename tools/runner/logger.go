package runner

import (
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
)

// PodLogger provides functionality to save pod logs to files.
type PodLogger struct {
}

// NewPodLogger creates a new PodLogger object.
func NewPodLogger() *PodLogger {
	return &PodLogger{}
}

// Save pod logs to file with same name as pod
func (pl *PodLogger) savePodLogs(ctx context.Context, loadTest *grpcv1.LoadTest, podLogDir string) error {
	// Get pods for this test
	pods, err := pl.getTestPods(ctx, loadTest)
	if err != nil {
		return err
	}

	// Attempt to create directory. Will not error if directory already exists
	err = os.MkdirAll(podLogDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Failed to create pod log output directory %s: %v", podLogDir, err)
	}

	// Write logs to files
	errorOccured := false
	for _, pod := range pods {
		logFilePath := filepath.Join(podLogDir, pod.Name+".log")
		err = pl.writePodLogToFile(ctx, pod, logFilePath)
		if err != nil {
			errorOccured = true
		}
	}
	if errorOccured {
		return errors.New("One or more log files could not be written")
	}
	return nil
}

// Get the pods associated with the given LoadTest
func (pl *PodLogger) getTestPods(ctx context.Context, loadTest *grpcv1.LoadTest) ([]*corev1.Pod, error) {
	clientset := getGenericClientset()
	podLister := clientset.CoreV1().Pods(metav1.NamespaceAll)

	// Get a list of all pods
	podList, err := podLister.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.New("Failed to fetch list of pods")
	}

	// Get pods just for this specific test
	testPods := status.PodsForLoadTest(loadTest, podList.Items)
	return testPods, nil
}

// Writes a single pod's logs to a file.
func (pl *PodLogger) writePodLogToFile(ctx context.Context, pod *corev1.Pod, logFilePath string) error {
	clientset := getGenericClientset()

	// Open log stream
	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{})
	driverLogs, err := req.Stream(ctx)
	defer driverLogs.Close()
	if err != nil {
		return fmt.Errorf("Could not open log stream for pod: %s", pod.Name)
	}

	// Open output file
	logFile, err := os.Create(logFilePath)
	defer logFile.Close()
	if err != nil {
		return fmt.Errorf("Could not open %s for writing", logFilePath)
	}

	// Write log to output file
	_, err = io.Copy(logFile, driverLogs)
	logFile.Sync()
	if err != nil {
		return fmt.Errorf("Error writing to %s: %v", logFilePath, err)
	}

	return nil
}
