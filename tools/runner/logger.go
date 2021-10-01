package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodLogger provides functionality to save pod logs to files
type PodLogger struct {
	outputDir string
}

func NewPodLogger(oFlag string) *PodLogger {
	outputDir := createPodLogOutputDir(oFlag)
	return &PodLogger{
		outputDir: outputDir,
	}
}

// Save pod logs to file with same name as pod
func (pl *PodLogger) savePodLogs(ctx context.Context, loadTest *grpcv1.LoadTest) error {
	// Get pods for this test
	pods, err := pl.getTestPods(ctx, loadTest)
	if err != nil {
		return err
	}

	// Try to write pod's logs to files, collecting possible errors
	collectedErrors := "One or more errors have occured:\n"
	errorPresent := false
	for _, pod := range pods {
		err = pl.writePodLogToFile(ctx, pod)
		if err != nil {
			collectedErrors += fmt.Errorf("\t%w\n", err).Error()
			errorPresent = true
		}
	}

	if errorPresent {
		return errors.New(collectedErrors)
	}
	return nil
}

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
// The file will be named whatever the pod's name is
func (pl *PodLogger) writePodLogToFile(ctx context.Context, pod *corev1.Pod) error {
	clientset := getGenericClientset()

	// Open log stream
	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{})
	driverLogs, err := req.Stream(ctx)
	defer driverLogs.Close()
	if err != nil {
		return fmt.Errorf("Could not open log stream for pod: %s", pod.Name)
	}

	// Open output file
	logFileName := pod.Name + ".log"
	logFilePath := filepath.Join(pl.outputDir, logFileName)
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

// Attempt to create containing directory for log files
// Return path of created or existing directory
func createPodLogOutputDir(oFlag string) string {
	subDir := "pod_logs"
	pathDir := filepath.Dir(oFlag)
	pathDir = filepath.Join(pathDir, subDir)

	// Attempt to create directory. Will not error if directory already exists
	err := os.MkdirAll(pathDir, os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create output directory %q: %v", pathDir, err)
	}

	return pathDir
}
