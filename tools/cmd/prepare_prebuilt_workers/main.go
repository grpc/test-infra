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
	"time"
)

// Tests contains the values for fields that are accessible by
// flags.
type Tests struct {
	preBuiltImagePrefix     string
	testTag                 string
	dockerfileRoot          string
	buildOnly               bool
	languagesToLanguageSpec map[string]LanguageSpec
}

// LanguageSpec containers the specs of each tested language.
type LanguageSpec struct {
	Name   string `json:"name"`
	Repo   string `json:"repo"`
	Gitref string `json:"gitref"`
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

	flag.Var(&languagesSelected, "l", "languages, its repository and GITREF wish to run tests, example: cxx:<commit-sha> or cxx:grpc/grpc:<commit-sha>")

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

	test.languagesToLanguageSpec = map[string]LanguageSpec{}
	converterToImageLanguage := map[string]string{
		"c++":             "cxx",
		"node_purejs":     "node",
		"php7_protobuf_c": "php7",
		"python_asyncio":  "python",
	}

	for _, pair := range languagesSelected {
		split := strings.SplitN(pair, ":", 3)

		// C++:master will be split to 2 items, c++:grpc/grpc:master will be
		// split to 3 items.
		if (len(split) == 0 || split[len(split)-1] == "" ){
			log.Fatalf("Input error in language and gitref selection. Please follow the format language:gitref or language:repository:gitref, for example c++:master or c++:grpc/grpc:master")
		}
		var spec LanguageSpec
		lang := split[0]
		if convertedLang, ok := converterToImageLanguage[lang]; ok {
			spec.Name = convertedLang
		} else {
			spec.Name = lang
		}
		if len(split) == 3 {
			spec.Repo = split[1]
			spec.Gitref = split[2]
		} else {
			spec.Repo = ""
			spec.Gitref = split[1]
		}
		test.languagesToLanguageSpec[spec.Name] = spec
	}

	log.Println("Selected language : REPOSITORY: GITREF")
	formattedMap, _ := json.MarshalIndent(test.languagesToLanguageSpec, "", "  ")
	log.Print(string(formattedMap))

	var wg sync.WaitGroup
	wg.Add(len(test.languagesToLanguageSpec))

	uniqueCacheBreaker := time.Now().String()

	for lang, spec := range test.languagesToLanguageSpec {
		go func(lang string, spec LanguageSpec) {
			defer wg.Done()

			image := fmt.Sprintf("%s/%s:%s", test.preBuiltImagePrefix, lang, test.testTag)
			dockerfileLocation := fmt.Sprintf("%s/%s/", test.dockerfileRoot, lang)

			// Build image
			log.Printf("building %s image\n", lang)
			buildCommandTimeoutSeconds := 30 * 60 // 30 mins should be enough for all languages
			buildDockerImage := exec.Command("timeout", fmt.Sprintf("%ds", buildCommandTimeoutSeconds), "docker", "build", dockerfileLocation, "-t", image, "--build-arg", fmt.Sprintf("GITREF=%s", spec.Gitref), "--build-arg", fmt.Sprintf("BREAK_CACHE=%s", uniqueCacheBreaker))
			if spec.Repo != "" {
				buildDockerImage.Args = append(buildDockerImage.Args, "--build-arg", fmt.Sprintf("REPOSITORY=%s", spec.Repo))
			}
			log.Printf("Running command: %s", strings.Join(buildDockerImage.Args, " "))
			buildOutput, err := buildDockerImage.CombinedOutput()
			if err != nil {
				log.Printf("Failed building %s image. Dump of command's output will follow:\n", lang)
				log.Println(string(buildOutput))
				log.Fatalf("Failed building %s image: %s", lang, err.Error())
			}
			log.Printf("Succeeded building %s image. Dump of command's output will follow:\n", lang)
			log.Println(string(buildOutput))
			log.Printf("Succeeded building %s image: %s\n", lang, image)

			if !test.buildOnly {
				// Push image
				log.Printf("pushing %s image\n", lang)
				pushDockerImage := exec.Command("docker", "push", image)
				pushOutput, err := pushDockerImage.CombinedOutput()
				if err != nil {
					log.Printf("Failed pushing %s image. Dump of command's output will follow:\n", lang)
					log.Println(string(pushOutput))
					log.Fatalf("Failed pushing %s image: %s", lang, err.Error())
				}
				log.Printf("Succeeded pushing %s image. Dump of command's output will follow:\n", lang)
				log.Println(string(pushOutput))
				log.Printf("Succeeded pushing %s image to %s\n", lang, image)
			}
		}(lang, spec)
	}

	wg.Wait()

	log.Printf("All images are processed")
}
