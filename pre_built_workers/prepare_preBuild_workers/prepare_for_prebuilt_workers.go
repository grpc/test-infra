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
	"bufio"
	"encoding/json"
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

// generateTag takes KOKORO_BUILD_INITIATOR, which is the user who start the
// current Kokoro job, and current time to generate unique tag string for
// images.
func generateTag() string {
	user := os.Getenv("KOKORO_BUILD_INITIATOR")
	testTime := time.Now().Format("2006-01-02-150405")
	if user == "" {
		user = "anonymous_user"
		log.Println("could not find user")
	}
	tag := user + "-" + testTime
	return tag
}

func main() {
	testTag := generateTag()
	outputFilePath := os.Getenv("PREBUILD_WORKER_OUTPUT_PATH")

	if outputFilePath == "" {
		outputFilePath = "$HOME/grpc-test-infra/container_registry/container_registry_path.yaml"
		log.Println(fmt.Sprintf("failed to get path for output file, create file at: %s", outputFilePath))
	}

	file, err := os.Create(outputFilePath)
	if err != nil {
		log.Fatalf("failed to create file to records container registry: %s", err)
	}

	// write current tag into the file
	writer := bufio.NewWriter(file)
	if _, err := writer.WriteString(fmt.Sprintf("current tag: %s\n", testTag)); err != nil {
		log.Fatal(fmt.Sprintf("failed to record tag for the current test: %s", testTag))
	}

	if flushError := writer.Flush(); flushError != nil {
		log.Fatal(fmt.Sprintf("failed to write message (current tag: %s\n) from buffer)", testTag))
	}

	var data Tests
	data.languages = map[string]string{}

	// specify the languages wish to run tests with
	flag.StringVar(&data.langData, "l", "", "languages wish to test")

	// specify pre-built worker image's prefix
	flag.StringVar(&data.preBuiltImagePrefix, "p", "gcr.io/grpc-testing/e2etesting/pre_built_workers/", "pre-built worker image's prefix")

	// specify the gitRef for languages wish to run tests in
	flag.StringVar(&data.goGitRef, "go-gitRef", "master", "specify the go gitRef wish to test")
	flag.StringVar(&data.cxxGitRef, "cxx-gitRef", "master", "specify the cxx gitRef wish to test")
	flag.StringVar(&data.goGitRef, "ruby-gitRef", "master", "specify the ruby gitRef wish to test")
	flag.StringVar(&data.cxxGitRef, "php-gitRef", "master", "specify the php gitRef wish to test")
	flag.StringVar(&data.goGitRef, "node-gitRef", "master", "specify the node gitRef wish to test")
	flag.StringVar(&data.cxxGitRef, "python-gitRef", "master", "specify python cxx gitRef wish to test")
	flag.StringVar(&data.goGitRef, "java-gitRef", "master", "specify the java gitRef wish to test")

	flag.Parse()

	for _, lang := range flag.Args() {
		lowerCaseLang := strings.ToLower(lang)
		data.languages[lowerCaseLang] = ""
	}

	if len(data.languages) == 0 {
		log.Fatalf("please select languages of test you wish to run with pre-built images")
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

	log.Print("Selected language : gitref")
	formattedMap, _ := json.MarshalIndent(data.languages, "", "  ")
	log.Print(string(formattedMap))

	for lang, gitRef := range data.languages {
		var containerRegistry = fmt.Sprintf("%s%s:%s", data.preBuiltImagePrefix, lang, testTag)

		// Build images
		var dockerfileLocation = fmt.Sprintf("../../containers/pre_built_workers/%s/", lang)

		var buildDockerImage *exec.Cmd

		buildDockerImage = exec.Command(
			"docker",
			"build", dockerfileLocation, "-t", containerRegistry, "-build-arg", fmt.Sprintf("GITREF=%s", gitRef),
		)

		log.Println(fmt.Sprintf("building %s imags", lang))

		buildStdout, err := buildDockerImage.StdoutPipe()
		if err != nil {
			log.Fatal(fmt.Sprintf("fail to present logs when building %s image", lang))
		}

		if err := buildDockerImage.Start(); err != nil {
			log.Fatal(fmt.Sprintf("failed to build %s worker: %s", lang, err))
		}

		buildScanner := bufio.NewScanner(buildStdout)
		buildScanner.Split(bufio.ScanLines)
		for buildScanner.Scan() {
			i := buildScanner.Text()
			fmt.Println(i)
		}
		if err := buildDockerImage.Wait(); err != nil {
			log.Fatal(fmt.Sprintf("exit during buildinig %s image, error: %s", lang, err))
		}

		log.Println(fmt.Sprintf("successfully build %s worker: %s", lang, containerRegistry))

		// Record current image's registry to the file
		message := fmt.Sprintf("%s: %s\n", lang, containerRegistry)

		if _, err := writer.WriteString(message); err != nil {
			log.Fatal(fmt.Sprintf("failed to record %s image with its container registry: %s", lang, containerRegistry))
		}

		if err := writer.Flush(); err != nil {
			log.Fatal(fmt.Sprintf("failed to write message (%s) from buffer)", message))
		}

		// Push image
		pushDockerImage := exec.Command(
			"docker",
			"push",
			containerRegistry,
		)

		log.Println(fmt.Sprintf("pushing %s image", lang))

		pushStdout, err := pushDockerImage.StdoutPipe()
		if err != nil {
			log.Fatal(fmt.Sprintf("failed to present logs when pushing %s images", lang))
		}

		if err := pushDockerImage.Start(); err != nil {
			log.Fatal(fmt.Sprintf("failed to push %s image to: %s", lang, containerRegistry))
		}

		pushScanner := bufio.NewScanner(pushStdout)
		pushScanner.Split(bufio.ScanLines)
		for pushScanner.Scan() {
			j := pushScanner.Text()
			fmt.Println(j)
		}
		if err := pushDockerImage.Wait(); err != nil {
			log.Fatal(fmt.Sprintf("exit during pushing %s image, error: %s", lang, err))
		}

		log.Println(fmt.Sprintf("successfully pushed %s worker to %s", lang, containerRegistry))
	}

	if err := file.Close(); err != nil {
		log.Fatalf("failed to close file")
	}
}

// go run prepare_for_prebuilt_workers.go -l 1 go
// has to run from the directory the script was in
