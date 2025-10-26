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

// SystemSpec defines the desired state of System
type SystemSpec struct {
	// DisplayName is the human friendly name of the system
	DisplayName string `json:"displayName"`

	// Description is a short description of the system
	// +optional
	Description string `json:"description,omitempty"`

	// SystemId is the unique identifier of the system. It is optional to allow
	// the controller to assign an ID automatically if not provided by the user.
	// +optional
	SystemId string `json:"systemId,omitempty"`

	// Version is the version of the system
	Version Version `json:"version"`
}

// Version defines information about the version of of an element of the system.
type Version struct {
	// Version is the identifier of the version, e.g. "1.0.0". The use of semantic versioning is recommended.
	Version string `json:"version"`

	// AvailableFrom is the date when this version became available.
	// +optional
	// +kubebuilder:validation:Format=date
	AvailableFrom string `json:"availableFrom,omitempty"`

	// DeprecatedFrom is the date when this version became deprecated.
	// +optional
	// +kubebuilder:validation:Format=date
	DeprecatedFrom string `json:"deprecatedFrom,omitempty"`

	// TerminatedFrom is the date when this version became terminated.
	// +optional
	// +kubebuilder:validation:Format=date
	TerminatedFrom string `json:"terminatedFrom,omitempty"`
}

// define a reference to another entity.
type VersionRef struct {
	// Name of the entity to reference.
	Name string `json:"name"`
	// Version of the blueprint to reference
	Version string `json:"version"`
}

// SystemStatus defines the observed state of System
type SystemStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// System is the Schema for the systems API
type System struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SystemSpec   `json:"spec,omitempty"`
	Status SystemStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SystemList contains a list of System
type SystemList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []System `json:"items"`
}

func init() {
	SchemeBuilder.Register(&System{}, &SystemList{})
}
