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

// Configure is an executable that generates a defaults file for the manager.
// It accepts a template file and replaces placeholders with data that may
// change based on where the manager and container images will live and run.
//
// This tool uses Go's text/template package for templating, see
// https://pkg.go.dev/text/template for a description of the syntax.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Tests contains the values for fields that are accessible by
// flags.
type Tests struct {
	version             string
	preBuiltImagePrefix string
	langData            string
	languages           map[string]string
	cxxGitRef           string
	goGitRef            string
	pythonGitRef        string
	javaGitRef          string
	csharpGitRef        string
	rubyGitRef          string
	phpGitRef           string
	nodeGitRef          string
}

func generateTag() string {
	user := os.Getenv("KOKORO_BUILD_INITIATOR")
	testTime := time.Now().String()
	tag := user + "-" + testTime
	return tag
}

func init() {
	//TODO: modify this usage message, this is WIP
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s <template-file> <output-file>\n", os.Args[0])

		fmt.Fprintf(flag.CommandLine.Output(), `explaination
`)

		fmt.Fprintf(flag.CommandLine.Output(), "\nFlags:\n")
		flag.PrintDefaults()
	}
}

func main() {
	testTag := generateTag()

	file, err := os.Create(fmt.Sprintf("Image_location_for_build:%s", testTag))
	if err != nil {
		log.Fatalf("failed to create file to records image repositor: %s", err)
		os.Exit(126)
	}
	defer file.Close()

	if _, err := file.WriteString(fmt.Sprintf("current tag: %s", testTag)); err != nil {
		log.Fatal(fmt.Sprintf("failed to record tag for the current test: %s", testTag))
		os.Exit(126)
	}

	var data Tests
	data.languages = map[string]string{}

	// specify the languages wish to run tests in
	flag.StringVar(&data.langData, "l", "", "languages wish to test")

	// specify pre-built worker image's prefix
	flag.StringVar(&data.preBuiltImagePrefix, "p", "gcr.io/grpc-testing/e2etesting/pre_built_workers/", "pre-built worker image's prefix")

	// specify the gitRef for languages wish to run tests in
	flag.StringVar(&data.goGitRef, "go-gitRef", "", "specify the go gitRef wish to test")
	flag.StringVar(&data.cxxGitRef, "cxx-gitRef", "", "specify the cxx gitRef wish to test")
	flag.StringVar(&data.goGitRef, "ruby-gitRef", "", "specify the ruby gitRef wish to test")
	flag.StringVar(&data.cxxGitRef, "php-gitRef", "", "specify the php gitRef wish to test")
	flag.StringVar(&data.goGitRef, "node-gitRef", "", "specify the node gitRef wish to test")
	flag.StringVar(&data.cxxGitRef, "python-gitRef", "", "specify python cxx gitRef wish to test")
	flag.StringVar(&data.goGitRef, "java-gitRef", "", "specify the java gitRef wish to test")

	flag.Parse()

	for _, lang := range flag.Args() {
		lowerCaseLang := strings.ToLower(lang)
		data.languages[lowerCaseLang] = ""
	}

	if len(data.languages) == 0 {
		log.Fatalf("please select languages of test you wish to run with pre-built images")
		os.Exit(1)
	}

	if _, ok := data.languages["cxx"]; ok {
		data.languages["cxx"] = data.cxxGitRef
	}
	if _, ok := data.languages["go"]; ok {
		data.languages["go"] = data.goGitRef
	}
	if _, ok := data.languages["java"]; ok {
		data.languages["java"] = data.javaGitRef
	}
	if _, ok := data.languages["python"]; ok {
		data.languages["python"] = data.pythonGitRef
	}
	if _, ok := data.languages["php"]; ok {
		data.languages["php"] = data.phpGitRef
	}
	if _, ok := data.languages["node"]; ok {
		data.languages["node"] = data.nodeGitRef
	}
	if _, ok := data.languages["ruby"]; ok {
		data.languages["ruby"] = data.rubyGitRef
	}
	// This information could also be output as a file if needed.
	log.Println(data.languages)

	for lang, gitRef := range data.languages {
		var buildDockerImage *exec.Cmd

		var curRepo = fmt.Sprintf("%s%s:%s", data.preBuiltImagePrefix, lang, testTag)
		var dockerfileLocation = fmt.Sprintf("../../containers/pre_built_workers/%s/", lang)

		if gitRef != "" {
			buildDockerImage = exec.Command(
				"docker",
				"build",
				dockerfileLocation,
				"-t",
				curRepo,
				"-build-arg",
				fmt.Sprintf("GITREF=%s", gitRef),
			)
		} else {
			buildDockerImage = exec.Command(
				"docker",
				"build",
				dockerfileLocation,
				"-t",
				curRepo,
			)
		}

		log.Println(fmt.Sprintf("build the %s imags, command excecuted: %s", lang, buildDockerImage))

		if err := buildDockerImage.Run(); err != nil {
			log.Fatal(fmt.Sprintf("failed to build docker imaged for %s worker: %s", lang, err))
			os.Exit(126)
		}

		log.Println(fmt.Sprintf("successfully build %s worker: %s%s:%s", lang, data.preBuiltImagePrefix, lang, testTag))

		pushDockerImage := exec.Command(
			"docker",
			"push",
			fmt.Sprintf("%s%s:%s", data.preBuiltImagePrefix, lang, testTag),
		)

		if _, err := file.WriteString(fmt.Sprintf("%s: %s", lang, curRepo)); err != nil {
			log.Fatal(fmt.Sprintf("failed to record pre-built %s image with its destination: %s, the image has not been pushed yet", lang, curRepo))
			os.Exit(126)
		}

		log.Println(fmt.Sprintf("push to container registry, command excecuted: %s", pushDockerImage))

		if err := pushDockerImage.Run(); err != nil {
			log.Fatal(fmt.Sprintf("failed to push pre-built %s image to: %s", lang, curRepo))
			os.Exit(126)
		}

		log.Println(fmt.Sprintf("successfully pushed %s worker: %s%s:%s", lang, data.preBuiltImagePrefix, lang, testTag))

	}
}

// go run prepare_for_prebuilt_workers.go -l 1 go has to run from the directory the script was in
