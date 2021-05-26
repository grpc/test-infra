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

package inspect

import (
	"sort"

	grpcv1 "github.com/grpc/test-infra/api/v1"
)

// GetWorkerLanguages returns a sorted slice with the languages of all servers
// and drivers in a LoadTest.
func GetWorkerLanguages(config *grpcv1.LoadTest) []string {
	var languages []string
	languageSet := make(map[string]bool)
	for _, server := range config.Spec.Servers {
		language := server.Language
		if _, ok := languageSet[language]; !ok {
			languageSet[language] = true
		}
	}
	for _, client := range config.Spec.Clients {
		language := client.Language
		if _, ok := languageSet[language]; !ok {
			languageSet[language] = true
		}
	}
	for language := range languageSet {
		languages = append(languages, language)
	}
	sort.Strings(languages)
	return languages
}
