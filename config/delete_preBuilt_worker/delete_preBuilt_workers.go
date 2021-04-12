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
	"time"
)

// Images contains the values for fields that are accessible by
// flags.
type Images struct {
	imageName string
	tag       string
	time      time.Time
	digest    string
}

func main() {
	var imageName string
	//var TTL time.Time

	flag.StringVar(&imageName, "image-name", "", "HOSTNAME/PROJECT-ID/IMAGE")

	//flag.StringVar(&TTL, "cut time", "", "the time we run the cleanup job")
	flag.Parse()

	getImages := exec.Command(
		"gcloud",
		"container",
		"images",
		"list-tags",
		imageName,
	)

	log.Printf("command run is : %s", getImages)

	outPut, _ := getImages.Output()
	images := make(map[time.Time]Images) //tag -> Images
	for i, line := range strings.Split(string(outPut), "\n") {
		curFields := strings.Fields(line)
		if len(curFields) == 0 || i == 0 {
			continue
		}
		//TODO: find out what is the timestamp stands for (creation or pushed)
		var curImage Images
		curImage.imageName = imageName
		curImage.digest = curFields[0]
		curImage.time, _ = time.Parse("2006-01-02T15:04:05", curFields[len(curFields)-1])

		if len(curFields) >= 3 {
			curImage.tag = curFields[1]
		}
		fmt.Println(curImage.time)
		images[curImage.time] = curImage
	}
	fmt.Println(images)
}

// has to run from the directory the script was in
