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
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
)

// Tests contains the values for fields that are accessible by
// flags.
type Tests struct {
	preBuiltImagePrefix string
	testTag             string
	dockerfileRoot      string
	buildOnly           bool
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

	flag.StringVar(&test.preBuiltImagePrefix, "p", "", "image registry to push images")

	flag.BoolVar(&test.buildOnly, "build-only", false, "use build-only=true if the images are not intended to be pushed to a container registry")

	flag.StringVar(&test.testTag, "t", "", "tag for pre-built images, this unique tag to identify the images build and pushed in current test")

	flag.StringVar(&test.dockerfileRoot, "r", "", "root directory of Dockerfiles to build prebuilt images")

	flag.Var(&languagesSelected, "l", "languages and its GITREF wish to run tests, example: cxx:master")

	flag.Parse()

	if test.preBuiltImagePrefix == "" {
		log.Fatalf("No registry provided, please provide a container registry.If the images are not intended to be pushed to a registry, please provide a prefix for naming the built images")
	}

	if test.testTag == "" {
		log.Fatalf("Failed preparing prebuilt images: no image tag provided")
	} else if len(test.testTag) > 128 {
		log.Fatalf("Failed preparing prebuilt images: invalid tag name, a tag name may not start with a period or a dash and may contain a maximum of 128 characters.")
	}

	if test.dockerfileRoot == "" {
		log.Fatalf("Fail preparing prebuilt images: no root directory for Dockerfiles provided")
	}

	if len(languagesSelected) == 0 {
		log.Fatalf("Failed preparing prebuilt images: no language and its gitref pair specified, please provide languages and the GITREF as cxx:master")
	}

	test.languagesToGitref = map[string]string{}
	converterToImageLanguage := map[string]string{
		"c++":             "cxx",
		"node_purejs":     "node",
		"php7":            "php",
		"php7_protobuf_c": "php",
		"python_asyncio":  "python",
	}

	for _, pair := range languagesSelected {
		split := strings.Split(pair, ":")

		if len(split) != 2 || split[len(split)-1] == "" {
			log.Fatalf("Input error in language and gitref selection, please follow the format as language:gitref, for example: cxx:master")
		}

		lang := split[0]
		gitref := split[1]

		if convertedLang, ok := converterToImageLanguage[lang]; ok {
			test.languagesToGitref[convertedLang] = gitref
		} else {
			test.languagesToGitref[lang] = gitref
		}
	}

	log.Println("Selected language : GITREF")
	formattedMap, _ := json.MarshalIndent(test.languagesToGitref, "", "  ")
	log.Print(string(formattedMap))

	var wg sync.WaitGroup
	wg.Add(len(test.languagesToGitref))

	for lang, gitRef := range test.languagesToGitref {
		go func(lang string, gitRef string) {
			defer wg.Done()

			image := fmt.Sprintf("%s/%s:%s", test.preBuiltImagePrefix, lang, test.testTag)
			dockerfileLocation := fmt.Sprintf("%s/%s/", test.dockerfileRoot, lang)

			// build image
			log.Println(fmt.Sprintf("building %s image", lang))
			buildDockerImage := exec.Command("docker", "build", dockerfileLocation, "-t", image, "--build-arg", fmt.Sprintf("GITREF=%s", gitRef), "--build-arg", fmt.Sprintf("BREAK_CACHE=%s", test.testTag))
			buildOutput, err := buildDockerImage.CombinedOutput()
			if err != nil {
				log.Println(err)
				log.Fatalf("Failed building %s image: %s", lang, string(buildOutput))
			}
			//log.Println(string(buildOutput))
			log.Printf("Succeeded building %s worker: %s\n", lang, image)

			if !test.buildOnly {
				// push image
				log.Println(fmt.Sprintf("pushing %s image", lang))
				pushDockerImage := exec.Command("docker", "push", image)
				pushOutput, err := pushDockerImage.CombinedOutput()
				if err != nil {
					log.Println(err)
					log.Fatalf("Failed pushing %s image: %s", lang, string(pushOutput))
				}
				//log.Println(string(pushOutput))
				log.Printf("Succeeded pushing %s worker to %s\n", lang, image)
			}
		}(lang, gitRef)
	}

	wg.Wait()

	log.Printf("All images are processed")
}
