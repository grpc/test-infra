/*
Copyright 2021 gRPC authors.

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

package runner

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// FileNames defines an accumulator flag for file names.
type FileNames []string

// Set implements the flag.Value interface.
func (f *FileNames) Set(value string) error {
	if value == "" {
		return errors.New("value must not be empty")
	}
	*f = append(*f, value)
	return nil
}

// String implements the flag.Value interface.
func (f *FileNames) String() string {
	return fmt.Sprint(*f)
}

// ConcurrencyLevels defines an accumulator flag for concurrency levels.
// Concurrency levels are in the form [<queue name>:]<concurrency level>.
// These values are parsed and accumulated into a map.
type ConcurrencyLevels map[string]int

// Set implements the flag.Value interface.
func (c *ConcurrencyLevels) Set(value string) error {
	var key string
	var cLevelString string
	elems := strings.SplitN(value, ":", 2)
	if len(elems) < 2 {
		cLevelString = elems[0]
	} else {
		key = elems[0]
		cLevelString = elems[1]
	}
	cLevel, err := strconv.Atoi(cLevelString)
	if err != nil {
		if key == "" {
			return errors.New("value must be of the form [<queue name>:]<concurrency level>")
		}
		return fmt.Errorf("concurrency level must be an integer, got %s", cLevelString)
	}
	if cLevel <= 0 {
		return fmt.Errorf("concurrency level must be positive, got %d", cLevel)
	}
	if (*c) == nil {
		(*c) = make(map[string]int)
	}
	(*c)[key] = cLevel
	if (*c)[""] > 0 && len(*c) > 1 {
		return errors.New("global capacity and queue names are mutually exclusive")
	}
	return nil
}

// String implements the flag.Value interface.
func (c *ConcurrencyLevels) String() string {
	return fmt.Sprint(*c)
}
