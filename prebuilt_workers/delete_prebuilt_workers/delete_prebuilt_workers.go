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

func main() {
	var imagePrefix string
	var tagOfImagesToDelete string

	flag.StringVar(&imagePrefix, "p", "", "set the repository for search, in the form of HOSTNAME/PROJECT-ID, do not include image name")
	flag.StringVar(&tagOfImagesToDelete, "t", "", "images with this tag will be deleted")

	flag.Parse()

	getRepository := exec.Command(
		"gcloud",
		"container",
		"images",
		"list",
		fmt.Sprintf("--repository=%s", imagePrefix),
	)

	allRepositories, err := getRepository.CombinedOutput()
	if err != nil {
		log.Println(fmt.Sprintf("failed to get repositories within %s", imagePrefix))
		log.Fatalf("error message: %s\n", allRepositories)
	}

	log.Println(fmt.Sprintf("below are all images within specified registry: %s", imagePrefix))
	log.Println(string(allRepositories))

	for i, repo := range strings.Split(string(allRepositories), "\n") {
		if i == 0 || repo == "" {
			continue
		}

		log.Println(fmt.Sprintf("current processing image: %s", repo))

		getImage := exec.Command(
			"gcloud",
			"container",
			"images",
			"list-tags",
			repo,
			fmt.Sprintf("--filter=%s", tagOfImagesToDelete),
		)

		imageByte, err := getImage.CombinedOutput()
		if err != nil {
			log.Println(fmt.Sprintf("failed to get the image %s: %s", repo, err))
			log.Fatalf("error message: %s\n", imageByte)
		}

		imageLine := strings.Split(string(imageByte), "\n")
		if len(imageLine) <= 2 {
			log.Printf("tag: %s is not presented.\n", tagOfImagesToDelete)
			continue
		}

		image := imageLine[1]
		fields := strings.Fields(image)
		tags := strings.Split(fields[1], ",")

		if len(tags) > 1 {
			log.Println(fmt.Sprintf("image have multiple tags, including %s, untag the image with tag %s instead of deleting image", tagOfImagesToDelete, tagOfImagesToDelete))
			untagImages := exec.Command(
				"gcloud",
				"-q",
				"container",
				"images",
				"untag",
				fmt.Sprintf("%s:%s", repo, tagOfImagesToDelete),
			)
			unTagImageStdout, err := untagImages.CombinedOutput()
			if err != nil {
				log.Println(fmt.Sprintf("failed to untag %s:%s", repo, tagOfImagesToDelete))
				log.Fatalf(fmt.Sprintf("error message: %s\n", unTagImageStdout))
			}
			log.Printf("successfully untag %s:%s", repo, tagOfImagesToDelete)
		} else {
			deleteImages := exec.Command(
				"gcloud",
				"-q",
				"container",
				"images",
				"delete",
				fmt.Sprintf("%s:%s", repo, tagOfImagesToDelete),
			)
			deleteImagesStdout, err := deleteImages.CombinedOutput()
			if err != nil {
				log.Println(fmt.Sprintf("failed to delete %s:%s", repo, tags[0]))
				log.Fatalf("err message: %s\n", deleteImagesStdout)
			}
			log.Printf("successfully delete %s:%s\n", repo, tagOfImagesToDelete)
		}
	}
}

// go run delete_prebuilt_workers.go -p gcr.io/grpc-testing/wanlin/pre_built_workers -t wanlindu-2021-04-15-134444
