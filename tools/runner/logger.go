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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grpc/test-infra/config"
	"github.com/grpc/test-infra/status"
)

type PodLogger struct {
	outputDir string
}

func NewPodLogger(oFlag string) *PodLogger {
	outputDir := createPodLogOutputDir(oFlag)
	return &PodLogger{
		outputDir: outputDir,
	}
}

func (pl *PodLogger) saveDriverLogs(ctx context.Context, loadTest *grpcv1.LoadTest) error {
	clientset := getGenericClientset()
	podLister := clientset.CoreV1().Pods(metav1.NamespaceAll)

	// Get a list of all pods
	podList, err := podLister.List(ctx, metav1.ListOptions{})
	if err != nil {
		return errors.New("Failed to fetch list of pods")
	}

	// Get pods just for this specific test
	testPods := status.PodsForLoadTest(loadTest, podList.Items)

	// Find driver pod
	foundDriverPod := false
	var driverPod *corev1.Pod
	for _, pod := range testPods {
		if pod.Labels[config.RoleLabel] == config.DriverRole {
			foundDriverPod = true
			driverPod = pod
		}
	}

	// Attempt to write driver logs to file
	if foundDriverPod {
		// Open log stream
		req := clientset.CoreV1().Pods(driverPod.Namespace).GetLogs(driverPod.Name, &corev1.PodLogOptions{})
		driverLogs, err := req.Stream(ctx)
		defer driverLogs.Close()
		if err != nil {
			return errors.New("Could not open driver log stream")
		}

		// Open output file
		logFileName := driverPod.Name + ".log"
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

	} else {
		return errors.New("Could not find driver pod")
	}

	return nil
}

// Attempt to create containing directory for log files
// Return path of created or existing directory
func createPodLogOutputDir(oFlag string) string {
	subDir := "pod-logs"
	pathDir := filepath.Dir(oFlag)
	pathDir = filepath.Join(pathDir, subDir)

	// Attempt to create directory. Will not error if directory already exists
	err := os.MkdirAll(pathDir, os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create output directory %q: %v", pathDir, err)
	}

	return pathDir
}
