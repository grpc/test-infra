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
	testTag             string
	languagesToGitref   map[string]string
	useCurrentTime      bool
}

type langFlags []string

func (l *langFlags) String() string {
	var result string
	for _, lang := range *l {
		result = result + " " + lang
	}
	return result
}

func (l *langFlags) Set(value string) error {
	*l = append(*l, value)
	return nil
}

var languagesSelected langFlags

func main() {
	var test Tests

	// specify the PREBUILD_IMAGE_PREFIX
	flag.StringVar(&test.preBuiltImagePrefix, "p", "", "image registry to push the image")

	// specify the tag of pre built image
	flag.StringVar(&test.testTag, "t", "", "tags for pre-built images, this unique tag to identify the images build and pushed in current test, if a tag is not provided, will user default tag as: build-initiator-date-time")

	// specify the languages wish to run tests with
	flag.Var(&languagesSelected, "l", "languages and its GITREF wish to run tests, example: cxx:master")

	flag.Parse()

	if test.preBuiltImagePrefix == "" {
		log.Println("no registry provided, using default image registry: gcr.io/grpc-testing/e2etesting/pre_built_workers")
		test.preBuiltImagePrefix = "gcr.io/grpc-testing/e2etesting/pre_built_workers"
	}

	if test.testTag == "" {
		user := os.Getenv("KOKORO_BUILD_INITIATOR")
		testTime := time.Now().Format("2006-01-02-15-04-05")
		if user == "" {
			user = "anonymous-user"
			log.Println("could not find kokoro build initiator, use anonymous_user instead")
		}
		test.testTag = user + "-" + testTime
		log.Println(fmt.Sprintf("no pre-built image gat provided, using default PREBUILD_IMAGE_TAG: %s", test.testTag))
	} else if len(test.testTag) > 128 {
		log.Fatalf("invalid tag name: A tag name may not start with a period or a dash and may contain a maximum of 128 characters.")
	}

	if len(languagesSelected) == 0 {
		log.Fatalf("please select languages and the GITREF for tests you wish to run with pre-built images, for example: cxx:master")
	}

	test.languagesToGitref = map[string]string{}
	for _, pair := range languagesSelected {
		split := strings.Split(pair, ":")
		if len(split) != 2 {
			log.Fatalf("input error, please follow the format as language:gitref, for example: cxx:master")
		}
		test.languagesToGitref[split[0]] = split[1]
	}

	log.Print("selected language : GITREF")
	fmt.Println(test.languagesToGitref)
	formattedMap, _ := json.MarshalIndent(test.languagesToGitref, "", "  ")
	log.Print(string(formattedMap))

	for lang, gitRef := range test.languagesToGitref {
		var containerRegistry = fmt.Sprintf("%s/%s:%s", test.preBuiltImagePrefix, lang, test.testTag)
		log.Print(fmt.Sprintf("image registry: %s", containerRegistry))

		// Build images
		var dockerfileLocation = fmt.Sprintf("containers/pre_built_workers/%s/", lang)

		var buildDockerImage *exec.Cmd

		buildDockerImage = exec.Command(
			"docker",
			"build", dockerfileLocation, "-t", containerRegistry, "--build-arg", fmt.Sprintf("GITREF=%s", gitRef),
		)
		log.Print(buildDockerImage)
		log.Println(fmt.Sprintf("building %s images", lang))

		buildStdout, err := buildDockerImage.StdoutPipe()
		if err != nil {
			log.Fatalf("fail to present logs when building %s image", lang)
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

}
