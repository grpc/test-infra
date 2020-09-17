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

package optional

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TimePtr accepts a Kubernetes Time object and returns a pointer to it.
func TimePtr(t metav1.Time) *metav1.Time {
	return &t
}

// CurrentTimePtr determines the current time and returns a pointer to a
// Kubernetes Time representation of it.
func CurrentTimePtr() *metav1.Time {
	return TimePtr(metav1.Now())
}
