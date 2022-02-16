/*
Copyright 2022 gRPC authors.
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

package xds

import (
	"log"
)

// Logger implements the Logger interface required.
type Logger struct {
}

// Debugf prints out debug information.
func (logger Logger) Debugf(format string, args ...interface{}) {
	log.Printf(format+"\n", args...)
}

// Infof prints out useful information.
func (logger Logger) Infof(format string, args ...interface{}) {
	log.Printf(format+"\n", args...)
}

// Warnf prints out warnings.
func (logger Logger) Warnf(format string, args ...interface{}) {
	log.Printf(format+"\n", args...)
}

// Errorf prints out the error message and stops the process.
func (logger Logger) Errorf(format string, args ...interface{}) {
	log.Fatalf(format+"\n", args...)
}
