/*
Copyright 2020 gRPC authors.

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

package main

import (
	"context"
	"errors"
	"flag"
	"io/ioutil"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/yaml"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/config"
	"github.com/grpc/test-infra/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	scheme             = runtime.NewScheme()
	errMissingDefaults = errors.New("missing flag -defaults-file")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(grpcv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var defaultsFile string
	var metricsAddr string
	var probeAddr string
	var enableLeaderElection bool
	var namespace string

	flag.StringVar(&defaultsFile, "defaults-file", "config/defaults.yaml", "Path to a YAML file with a default configuration.")
	flag.StringVar(&namespace, "namespace", "", "Limits resources considered to a specific namespace.")
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	logger := log.FromContext(ctx).WithValues("controller", "LoadTest")
	logger.Info("starting manager")

	if defaultsFile == "" {
		logger.Error(errMissingDefaults, "cannot start without defaults")
		os.Exit(1)
	}

	defaultsBytes, err := ioutil.ReadFile(defaultsFile)
	if err != nil {
		logger.Error(err, "could not read defaults file")
		os.Exit(1)
	}

	defaultOptions := config.Defaults{}
	if err := yaml.Unmarshal(defaultsBytes, &defaultOptions); err != nil {
		logger.Error(err, "could not parse the defaults file contents")
		os.Exit(1)
	}

	if err := defaultOptions.Validate(); err != nil {
		logger.Error(err, "failed to start due to invalid defaults")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "284e7070.e2etest.grpc.io",
		Namespace:              namespace,
	})
	if err != nil {
		logger.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.LoadTestReconciler{
		Defaults: &defaultOptions,
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to create controller", "controller", "LoadTest")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Error(err, "problem running manager")
		os.Exit(1)
	}
}
