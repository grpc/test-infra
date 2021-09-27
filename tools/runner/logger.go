package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/grpc/test-infra/config"
	"github.com/grpc/test-infra/status"
)

func saveDriverLogs(ctx context.Context, loadTest *grpcv1.LoadTest) error {
	clientset := GetGenericClientset()
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
		// open log stream
		req := clientset.CoreV1().Pods(driverPod.Namespace).GetLogs(driverPod.Name, &corev1.PodLogOptions{})
		driverLogs, err := req.Stream(ctx)
		defer driverLogs.Close()
		if err != nil {
			return errors.New("Could not open driver log stream")
		}

		// open output file
		logFileName := driverPod.Name + ".log"
		// logFileName := "pod_logs/" + driverPod.Name + ".log" // TODO: save into "pod_logs" directory
		f, err := os.Create(logFileName)
		defer f.Close()
		if err != nil {
			return fmt.Errorf("Could not open %s for writing", logFileName)
		}

		io.Copy(f, driverLogs)
		f.Sync()

	} else {
		return errors.New("Could not find driver pod")
	}

	return nil
}
