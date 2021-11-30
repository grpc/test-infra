package xds

import (
	"fmt"
	"io/ioutil"

	"github.com/grpc/test-infra/tools/xds/api/v1"

	"gopkg.in/yaml.v2"
)

// ParseYaml takes in a yaml envoy config and returns a typed version
func ParseYaml(file string) (*v1.Endpoint, error) {
	var endpoiint v1.Endpoint

	yamlConfigFile, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("error reading YAML file: %s", err)
	}

	err = yaml.Unmarshal(yamlConfigFile, &endpoiint)
	if err != nil {
		return nil, err
	}

	return &endpoiint, nil
}
