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

	getRepository := exec.Command("gcloud", "container", "images", "list", fmt.Sprintf("--repository=%s", imagePrefix))
	getRepositoryOutput, err := getRepository.CombinedOutput()
	if err != nil {
		log.Printf("failed to get repositories within %s: %s\n", imagePrefix, string(getRepositoryOutput))
	}

	log.Printf("below are all images within specified registry: %s\n", imagePrefix)
	log.Println(string(getRepositoryOutput))

	allRepositories := strings.Split(string(getRepositoryOutput), "\n")
	for i, curRepository := range allRepositories {
		if i == 0 || curRepository == "" {
			continue
		}
		log.Printf("current processing image repository: %s\n", curRepository)

		curImageToProcess := fmt.Sprintf("%s:%s", curRepository, tagOfImagesToDelete)

		getImageHaveTheTag := exec.Command("gcloud", "container", "images", "list-tags", curRepository, fmt.Sprintf("--filter=%s", tagOfImagesToDelete))
		getImageHaveTheTagOutput, err := getImageHaveTheTag.CombinedOutput()
		if err != nil {
			log.Printf("failed to get the image %s with tag %s: %s\n", curRepository, tagOfImagesToDelete, string(getImageHaveTheTagOutput))
		}

		imageFullLine := strings.Split(string(getImageHaveTheTagOutput), "\n")
		if len(imageFullLine) <= 2 {
			log.Printf("tag: %s is not presented.\n", tagOfImagesToDelete)
			continue
		}

		numbersOfTagsOfCurrentImage := len(strings.Split(strings.Fields(imageFullLine[1])[1], ","))

		if numbersOfTagsOfCurrentImage > 1 {
			log.Printf("image have multiple tags, including %s, untag the image with tag %s instead of deleting image\n", tagOfImagesToDelete, tagOfImagesToDelete)
			untagImages := exec.Command("gcloud", "-q", "container", "images", "untag", curImageToProcess)
			unTagImageOutput, err := untagImages.CombinedOutput()
			if err != nil {
				log.Printf("failed to untag %s: %s\n", curImageToProcess, string(unTagImageOutput))
			}
			log.Printf("successfully untag %s:%s\n", curRepository, tagOfImagesToDelete)
		} else {
			deleteImage := exec.Command("gcloud", "-q", "container", "images", "delete", curImageToProcess)
			deleteImageOutput, err := deleteImage.CombinedOutput()
			if err != nil {
				log.Printf("failed to delete image %s : %s\n", curImageToProcess, string(deleteImageOutput))
			}
			log.Printf("successfully delete %s\n", curImageToProcess)
		}
	}
	log.Printf("all images with tag: %s within container registry: %s are processed.\n", tagOfImagesToDelete, imagePrefix)
}
