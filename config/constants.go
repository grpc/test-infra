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

const (
	// BazelCacheVolumeName holds the name of the volume which allows images to
	// share a bazel cache.
	BazelCacheVolumeName = "bazel-cache"

	// BazelCacheMountPath stores the directory where the bazel cache resides. For
	// a description of the bazel image and its cache/output directories, see
	// https://docs.bazel.build/versions/master/output_directories.html.
	BazelCacheMountPath = "/root/.cache/bazel"

	// BigQueryTableEnv specifies the name of the env variable that holds the name
	// of the table where results should be written.
	BigQueryTableEnv = "BQ_RESULT_TABLE"

	// BuildInitContainerName holds the name of the init container that assembles
	// a binary or other bundle required to run the tests.
	BuildInitContainerName = "build"

	// ClientRole is the value the controller expects for the RoleLabel
	// on a client component.
	ClientRole = "client"

	// CloneGitRefEnv specifies the name of the env variable that contains the
	// commit, tag or branch to checkout after cloning a git repository.
	CloneGitRefEnv = "CLONE_GIT_REF"

	// CloneInitContainerName holds the name of the init container that obtains
	// a copy of the code at a specific point in time.
	CloneInitContainerName = "clone"

	// CloneRepoEnv specifies the name of the env variable that contains the git
	// repository to clone.
	CloneRepoEnv = "CLONE_REPO"

	// ComponentNameLabel is a label used to distinguish between test
	// components with the same role.
	ComponentNameLabel = "loadtest-component"

	// DriverRole is the value the controller expects for the RoleLabel
	// on a driver component.
	DriverRole = "driver"

	// DriverPort is the number of the port that the servers and clients expose
	// for the driver to connect to. This connection allows the driver to send
	// instructions and receive results from the servers and clients.
	DriverPort = 10000

	// DriverPortEnv specifies the name of the env variable that contains driver port.
	DriverPortEnv = "DRIVER_PORT"

	// PoolLabel is the key for a label which will have the name of a pool as
	// the value.
	PoolLabel = "pool"

	// ReadyInitContainerName holds the name of the init container that blocks a
	// driver from running until all worker pods are ready.
	ReadyInitContainerName = "ready"

	// ReadyMountPath is the absolute path where the ready volume should be
	// mounted in both the ready init container and the driver's run container.
	ReadyMountPath = "/var/data/qps_workers"

	// ReadyOutputFile is the name of the file where the ready init container
	// should write all IP addresses and port numbers for ready workers.
	ReadyOutputFile = ReadyMountPath + "/addresses"

	// ReadyMetadataOutputFile is the name of the file where the ready init container
	// should write all Metadata.
	ReadyMetadataOutputFile = ReadyMountPath + "/metadata.json"

	// ReadyNodeInfoOutputFile is the name of the file where the ready init container
	// should write node infomation.
	ReadyNodeInfoOutputFile = ReadyMountPath + "/node_info.json"

	// ReadyVolumeName is the name of the volume that permits sharing files
	// between the ready init container and the driver's run container.
	ReadyVolumeName = "worker-addresses"

	// RoleLabel is a label with the role  of a test component. For
	// example, "loadtest-role=server" indicates a server component.
	RoleLabel = "loadtest-role"

	// RunContainerName holds the name of the main container where the test is
	// executed.
	RunContainerName = "run"

	// ScenariosFileEnv specifies the name of an env variable that specifies the
	// path to a JSON file with scenarios.
	ScenariosFileEnv = "SCENARIOS_FILE"

	// ScenariosMountPath specifies where the JSON file with the scenario should
	// be mounted in the driver container.
	ScenariosMountPath = "/src/scenarios"

	// ServerRole is the value the controller expects for the RoleLabel
	// on a server component.
	ServerRole = "server"

	// WorkspaceMountPath contains the path to mount the volume identified by
	// `workspaceVolume`.
	WorkspaceMountPath = "/src/workspace"

	// WorkspaceVolumeName contains the name of the volume that is shared between
	// the init containers and containers for a driver or worker pod.
	WorkspaceVolumeName = "workspace"

	// KillAfterEnv specifies the name of the env variable that sets the allowed response time for a pod after timeout.
	KillAfterEnv = "KILL_AFTER"

	// PodTimeoutEnv specifies the name of the env variable that sets the timeout for a pod.
	PodTimeoutEnv = "POD_TIMEOUT"
)
