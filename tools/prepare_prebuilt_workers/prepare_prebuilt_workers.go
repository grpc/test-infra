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
	"os/exec"
	"strings"
)

// Tests contains the values for fields that are accessible by
// flags.
type Tests struct {
	preBuiltImagePrefix string
	testTag             string
	dockerfileRoot      string
	languagesToGitref   map[string]string
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

	// specify the the image registry to push images
	flag.StringVar(&test.preBuiltImagePrefix, "p", "", "image registry to push the image")

	// specify the tag of prebuilt image
	flag.StringVar(&test.testTag, "t", "", "tags for pre-built images, this unique tag to identify the images build and pushed in current test")

	// specify the root path of Dockerfiles for prebuilt images
	flag.StringVar(&test.dockerfileRoot, "r", "", "root directory of dockerfiles to build prebuilt images")

	// specify the languages wish to run tests with
	flag.Var(&languagesSelected, "l", "languages and its GITREF wish to run tests, example: cxx:master")

	flag.Parse()

	if test.preBuiltImagePrefix == "" {
		log.Fatalf("failed to prepare prebuilt images: no registry provided, please provide a containr registry to store the images")
	}

	if test.testTag == "" {
		log.Fatalf("failed to prepare prebuilt images: no image tag provided")
	} else if len(test.testTag) > 128 {
		log.Fatalf("failed to prepare prebuilt images: invalid tag name, a tag name may not start with a period or a dash and may contain a maximum of 128 characters.")
	}

	if test.dockerfileRoot == "" {
		log.Fatalf("fail to prepare prebuilt images: no root directory for Dockerfiles provided")
	}

	if len(languagesSelected) == 0 {
		log.Fatalf("failed to prepare prebuilt images: no language and its gitref pair specified, please provide languages and the GITREF as cxx:master")
	}

	test.languagesToGitref = map[string]string{}
	for _, pair := range languagesSelected {
		split := strings.Split(pair, ":")
		if len(split) != 2 {
			log.Fatalf("input error, please follow the format as language:gitref, for example: cxx:master")
		}
		test.languagesToGitref[split[0]] = split[1]
	}

	log.Println("selected language : GITREF")
	log.Println(test.languagesToGitref)
	//formattedMap, _ := json.MarshalIndent(test.languagesToGitref, "", "  ")
	//log.Print(string(formattedMap))

	for lang, gitRef := range test.languagesToGitref {
		var image = fmt.Sprintf("%s/%s:%s", test.preBuiltImagePrefix, lang, test.testTag)
		var dockerfileLocation = fmt.Sprintf("%s/%s/", test.dockerfileRoot, lang)

		// build image
		log.Println(fmt.Sprintf("building %s images", lang))
		buildDockerImage := exec.Command("docker", "build", dockerfileLocation, "-t", image, "--build-arg", fmt.Sprintf("GITREF=%s", gitRef))
		buildOutput, err := buildDockerImage.CombinedOutput()
		if err != nil {
			log.Fatalf("failed to build %s image: %s", lang, string(buildOutput))
		}
		log.Println(string(buildOutput))
		log.Printf("successfully build %s worker: %s\n", lang, image)

		// push image
		log.Println(fmt.Sprintf("pushing %s image", lang))
		pushDockerImage := exec.Command("docker", "push", image)
		pushOutput, err := pushDockerImage.CombinedOutput()
		if err != nil {
			log.Fatalf("failed to push %s image: %s", lang, string(pushOutput))
		}
		log.Println(string(pushOutput))
		log.Printf("successfully pushed %s worker to %s\n", lang, image)
	}
	log.Printf("all images are built and pushed to container registry: %s with tag: %s", test.preBuiltImagePrefix, test.testTag)
}
