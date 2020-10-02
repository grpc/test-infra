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
	"flag"
	"fmt"
	"os"
	"text/template"
)

// config contains data that is accessible within the configuration template.
type config struct {
	Version         string
	DriverVersion   string
	DriverPool      string
	WorkerPool      string
	InitImagePrefix string
	ImagePrefix     string
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s <template-file> <output-file>\n\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Flags:\n")
		flag.PrintDefaults()
	}
}

func main() {
	var cfg config

	flag.StringVar(&cfg.Version, "version", "latest", "version of all docker images to use")

	flag.StringVar(&cfg.DriverVersion, "driver-version", "master", "version of gRPC that is included in the driver image")

	flag.StringVar(&cfg.DriverPool, "driver-pool", "", `pool where drivers are scheduled by default (required)

Drivers will be scheduled by default on nodes with a "pool" label that matches
this -driver-pool flag.`)

	flag.StringVar(&cfg.WorkerPool, "worker-pool", "", `pool where workers are scheduled by default (required)

Workers will be scheduled by default on nodes with a "pool" label that matches
this -worker-pool flag.`)

	flag.StringVar(&cfg.InitImagePrefix, "init-image-prefix", "", `prefix to append to init container images (optional)

This -init-image-prefix flag allows a specific prefix to apply to all init
container images.`)

	flag.StringVar(&cfg.InitImagePrefix, "image-prefix", "", `prefix to append to container images (optional)

This -image-prefix flag allows a specific prefix to apply to all container
images that are not used as init containers.`)

	flag.Parse()

	if flag.NArg() != 2 {
		exitWithErrorf(1, true, "missing required arguments")
	}

	if cfg.DriverPool == "" {
		exitWithErrorf(1, true, "missing required -driver-pool flag")
	}

	if cfg.WorkerPool == "" {
		exitWithErrorf(1, true, "missing required -worker-pool flag")
	}

	templ, err := template.ParseFiles(flag.Arg(0))
	if err != nil {
		exitWithErrorf(1, true, "could not open and parse <template-file>: %v", err)
	}

	outputFile, err := os.Create(flag.Arg(1))
	if err != nil {
		exitWithErrorf(1, true, "could not create <output-file>: %v", err)
	}

	if err := templ.Execute(outputFile, cfg); err != nil {
		exitWithErrorf(1, false, "could not write config to output file: %v", err)
	}
}

// exitWithErrorf aborts the process, logging a message to the command line and,
// optionally, printing the usage documentation for the configuration program.
func exitWithErrorf(code int, showUsage bool, messageFmt string, args ...interface{}) {
	if showUsage {
		flag.Usage()
	}

	fmt.Fprintf(flag.CommandLine.Output(), messageFmt+"\n", args...)
	os.Exit(code)
}
