/*
Copyright 2025 Lutz Behnke <lutz.behnke@emeland.io>.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// APISpec defines the desired state of API
// +kubebuilder:validation:ExactlyOneOf=SystemId;SystemRef
type APISpec struct {
	// DisplayName is the human friendly name of the API
	DisplayName string `json:"displayName"`

	// Description is a short description of the API
	// +optional
	Description string `json:"description,omitempty"`

	// apiId is the unique identifier of the API. It is optional to allow
	// the controller to assign an ID automatically if not provided by the user.
	// +optional
	ApiId string `json:"apiId,omitempty"`

	// Version is the version of the API
	Version Version `json:"version"`

	// Type is the type of the API, e.g. OpenAPI, GraphQL, gRPC, etc.
	// +kubebuilder:validation:Enum=OpenAPI;GraphQL;gRPC;Other
	Type string `json:"type"`

	// System is a reference to the system by name and version, that this API belongs to. Either SystemRef or SystemId must be set.
	SystemRef VersionRef `json:"systemRef"`

	// SystemId is a reference to the system by UUID, that this API belongs to. Either SystemRef or SystemId must be set.
	// +optional
	SystemId string `json:"systemId,omitempty"`
}

// +kubebuilder:validation:ExactlyOneOf=VersionRef;ApiId
type APIRef struct {
	// Name of the API to reference.
	// +optional
	VersionRef APIVersionRef `json:"versionRef,omitempty"`

	// Version of the API to reference
	// +optional
	ApiId string `json:"apiId,omitempty"`
}

type APIVersionRef struct {
	// Name of the API to reference.
	Name string `json:"name"`
	// Version of the API to reference
	Version string `json:"version"`
}

// APIStatus defines the observed state of API
type APIStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// API is the Schema for the apis API
type API struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   APISpec   `json:"spec,omitempty"`
	Status APIStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// APIList contains a list of API
type APIList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []API `json:"items"`
}

func init() {
	SchemeBuilder.Register(&API{}, &APIList{})
}
