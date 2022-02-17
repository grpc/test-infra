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

package kubehelpers

import (
	"encoding/json"
)

// UpdateConfigMapWithServerPort accepts a server port string and a scenarioString string.
// It returns a updated scenarioString with server port inserted. Currently only supports
// single scenario.
// TODO: wanlin31 to update the function to support array of scenarios.
func UpdateConfigMapWithServerPort(port string, scenarioString string) (string, error) {
	updatedScenarios := ""
	var jsonScenarioMap map[string]map[string]json.RawMessage
	if err := json.Unmarshal([]byte(scenarioString), &jsonScenarioMap); err != nil {
		return updatedScenarios, err
	}
	var serverConfig map[string]interface{}
	if err := json.Unmarshal(jsonScenarioMap["scenarios"]["server_config"], &serverConfig); err != nil {
		return updatedScenarios, err
	}
	serverConfig["port"] = port
	serverConfigByte, err := json.Marshal(serverConfig)
	if err != nil {
		return updatedScenarios, err
	}
	jsonScenarioMap["scenarios"]["server_config"] = serverConfigByte
	scenariosJSONByte, err := json.Marshal(jsonScenarioMap)
	if err != nil {
		return updatedScenarios, err
	}
	updatedScenarios = string(scenariosJSONByte)
	return updatedScenarios, nil
}
