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
)

// Tests contains the values for fields that are accessible by
// flags.
type Tests struct {
	version             string
	preBuiltImagePrefix string
	tag                 string
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

func main() {
	var data Tests
	data.languages = map[string]string{}
	//var languageKeys []string

	// specify the release version of gRPC OSS benchmark infrastructure
	flag.StringVar(&data.version, "version", "latest", "version of all docker images to use")

	// specify the languages wish to run tests in
	flag.StringVar(&data.langData, "l", "", "languages wish to test")

	// specify pre-built worker image's prefix and tag
	//TODO: modify this before checkin, make tag as a requirement field
	flag.StringVar(&data.preBuiltImagePrefix, "p", "gcr.io/grpc-testing/wanlin/pre_built_workers/", "pre-built worker image's prefix")
	flag.StringVar(&data.tag, "t", "", "pre-built worker image's tag, this is unique for each run")

	// specify the gitRef for languages wish to run tests in
	flag.StringVar(&data.goGitRef, "go gitRef wish to test", "", "specify the go gitRef wish to test")
	flag.StringVar(&data.cxxGitRef, "cxx gitRef wish to test", "", "specify the cxx gitRef wish to test")

	flag.Parse()

	for _, lang := range flag.Args() {
		data.languages[lang] = ""
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

	fmt.Println(data.languages)

	for lang, gitRef := range data.languages {
		var buildDockerImage *exec.Cmd

		if gitRef != "" {
			buildDockerImage = exec.Command(
				"docker",
				"build",
				fmt.Sprintf("../../containers/pre_built_workers/%s/", lang),
				"-t",
				fmt.Sprintf("%s%s:%s", data.preBuiltImagePrefix, lang, data.tag),
				fmt.Sprintf("GITREF=%s", gitRef),
				"-build-arg",
				fmt.Sprintf("GITREF=%s", gitRef),
			)

		} else {
			buildDockerImage = exec.Command(
				"docker",
				"build",
				fmt.Sprintf("../../containers/pre_built_workers/%s/", lang),
				"-t",
				fmt.Sprintf("%s%s:%s", data.preBuiltImagePrefix, lang, data.tag),
			)
		}

		fmt.Println(buildDockerImage)
		if err := buildDockerImage.Run(); err != nil {
			log.Fatal(err)
		}

		log.Println(fmt.Sprintf("Finished building %s worker: %s%s:%s", lang, data.preBuiltImagePrefix, lang, data.tag))

		pushDockerImage := exec.Command(
			"docker",
			"push",
			fmt.Sprintf("%s%s:%s", data.preBuiltImagePrefix, lang, data.tag),
		)
		if err := pushDockerImage.Run(); err != nil {
			log.Fatal(err)
		}
		log.Println(fmt.Sprintf("Finished pushing %s worker: %s%s:%s", lang, data.preBuiltImagePrefix, lang, data.tag))
	}
}

// go run prepare_for_prebuilt_workers.go -l 1 go has to run from the directory the script was in
