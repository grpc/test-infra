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

package config

import (
	"fmt"
)

// imageMap is a structure with a map that allows internal code to efficiently
// find the default build and runtime container images for a language. It is
// not intended to be a public API.
type imageMap struct {
	m map[string]*LanguageDefault
}

// newImageMap constructs an imageMap object.
func newImageMap(lds []LanguageDefault) *imageMap {
	m := make(map[string]*LanguageDefault)

	for i := range lds {
		ld := &lds[i]
		m[ld.Language] = ld
	}

	return &imageMap{m}
}

// buildImage returns the default build container image for a language. If the
// language has no default, an error is returned.
func (im *imageMap) buildImage(language string) (string, error) {
	ld, ok := im.m[language]
	if !ok {
		return "", fmt.Errorf("cannot find image for language %q", language)
	}

	return ld.BuildImage, nil
}

// runImage returns the default runtime container image for a language. If the
// language has no default, an error is returned.
func (im *imageMap) runImage(language string) (string, error) {
	ld, ok := im.m[language]
	if !ok {
		return "", fmt.Errorf("cannot find image for language %q", language)
	}

	return ld.RunImage, nil
}
