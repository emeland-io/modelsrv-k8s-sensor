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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SystemInstanceSpec defines the desired state of SystemInstance
type SystemInstanceSpec struct {

	// InstanceId is the unique identifier of this system instance.
	InstanceId string `json:"instanceId"`

	// SystemId is a reference to the system by UUID, that this instance belongs to.
	SystemId string `json:"systemId"`

	// DisplayName is indented as a label to display for humans to recognize this system instance.
	// Machines should use the InstanceId above instead. Ideally this would be unique across an overall IT-System.
	DisplayName string `json:"displayName"`
}

// SystemInstanceStatus defines the observed state of SystemInstance
type SystemInstanceStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SystemInstance is the Schema for the systeminstances API
type SystemInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SystemInstanceSpec   `json:"spec,omitempty"`
	Status SystemInstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SystemInstanceList contains a list of SystemInstance
type SystemInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SystemInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SystemInstance{}, &SystemInstanceList{})
}
