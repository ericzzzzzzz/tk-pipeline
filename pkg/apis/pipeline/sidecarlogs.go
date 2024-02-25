/*
Copyright 2022 The Tekton Authors

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

package pipeline

const (
	// ReservedResultsSidecarName is the name of the results sidecar that outputs the results to stdout
	// when the results-from feature-flag is set to "sidecar-logs".
	ReservedResultsSidecarName = "tekton-log-results"

	// ReservedArtifactsSidecarName is the name of the artifacts sidecar that outputs the artifacts to stdout
	// when the results-from feature-flag is set to "sidecar-logs".
	ReservedArtifactsSidecarName = "tekton-log-artifacts"

	// ReservedResultsSidecarContainerName is the name of the results sidecar container that is injected
	// by the reconciler.
	ReservedResultsSidecarContainerName = "sidecar-tekton-log-results"

	// ReservedArtifactsSidecarContainerName is the name of the artifacts sidecar container that is injected
	// by the reconciler.
	ReservedArtifactsSidecarContainerName = "sidecar-tekton-log-artifacts"
)
