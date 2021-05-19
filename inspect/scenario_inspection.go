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
	"encoding/json"
	"fmt"

	grpcv1 "github.com/grpc/test-infra/api/v1"
)

// GetScenarioNames attempts to unmarshal untyped JSON to find the name of the
// scenarios embedded in the .spec.scenariosJSON field of a LoadTest.
func GetScenarioNames(config *grpcv1.LoadTest) ([]string, error) {
	var scenarioNames []string

	jsonKV := make(map[string]interface{})
	if err := json.Unmarshal([]byte(config.Spec.ScenariosJSON), &jsonKV); err != nil {
		return nil, err
	}

	scenariosValue, ok := jsonKV["scenarios"]
	if !ok {
		return nil, fmt.Errorf("JSON is malformed: expected a \"scenarios\" key in `%s`", config.Spec.ScenariosJSON)
	}

	scenariosList, ok := scenariosValue.([]map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("JSON is malformed: the value for the \"scenarios\" key was not a list in `%s`", config.Spec.ScenariosJSON)
	}

	for i, scenarioMap := range scenariosList {
		scenarioNameInterface, ok := scenarioMap["name"]
		if !ok {
			return nil, fmt.Errorf("JSON is malformed: scenario[%d] does not have a name attribute in `%s`", i, config.Spec.ScenariosJSON)
		}

		scenarioName, ok := scenarioNameInterface.(string)
		if !ok {
			return nil, fmt.Errorf("JSON is malformed: scenario[%d] has a \"name\" key, but it's value is not a string in `%s`", i, config.Spec.ScenariosJSON)
		}

		scenarioNames = append(scenarioNames, scenarioName)
	}

	return scenarioNames, nil
}
