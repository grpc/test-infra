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
	"errors"
	"flag"
	"io/ioutil"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/yaml"

	grpcv1 "github.com/grpc/test-infra/api/v1"
	"github.com/grpc/test-infra/config"
	"github.com/grpc/test-infra/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = grpcv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var defaultsFile string
	var metricsAddr string
	var enableLeaderElection bool
	var namespace string
	flag.StringVar(&defaultsFile, "defaults-file", "config/defaults.yaml", "The path to a YAML file with a default configuration")
	flag.StringVar(&metricsAddr, "metrics-addr", ":3777", "The address the metric endpoint binds to.")
	flag.StringVar(&namespace, "namespace", "", "Limits resources considered to a specific namespace")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	if defaultsFile == "" {
		setupLog.Error(errors.New("missing defaultsFile flag"), "cannot start without defaults")
		os.Exit(1)
	}

	defaultsBytes, err := ioutil.ReadFile(defaultsFile)
	if err != nil {
		setupLog.Error(err, "could not read defaults file")
		os.Exit(1)
	}

	defaultOptions := config.Defaults{}
	if err := yaml.Unmarshal(defaultsBytes, &defaultOptions); err != nil {
		setupLog.Error(err, "could not parse the defaults file contents")
		os.Exit(1)
	}

	if err := defaultOptions.Validate(); err != nil {
		setupLog.Error(err, "failed to start due to invalid defaults")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "284e7070.e2etest.grpc.io",
		Namespace:          namespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.LoadTestReconciler{
		Defaults: &defaultOptions,
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("LoadTest"),
		Scheme:   mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "LoadTest")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
