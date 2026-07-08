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

// FindingRuleSpec defines the desired state of FindingRule
type FindingRuleSpec struct {
	// Target selects the Kubernetes resources this rule applies to.
	Target Target `json:"target"`

	// Condition is a CEL expression evaluated against each target resource.
	// When the expression evaluates to true, a finding is raised.
	// See https://kubernetes.io/docs/reference/using-api/cel/
	Condition string `json:"condition"`

	// Finding describes the alert to emit when the condition matches.
	Finding Finding `json:"finding"`
}

// Target selects Kubernetes resources by API group and resource name.
type Target struct {
	// APIGroups is the list of API groups to match (e.g. "" for core).
	APIGroups []string `json:"apiGroups"`

	// Resources is the list of resource names to match (e.g. "namespaces").
	Resources []string `json:"resources"`
}

// Finding describes the alert emitted when a rule's condition matches.
type Finding struct {
	// Severity is the severity level of the finding.
	// +kubebuilder:validation:Enum=low;middle;high
	Severity string `json:"severity"`

	// DisplayName is the human-friendly name of the finding.
	DisplayName string `json:"displayName"`

	// Description is the message shown when the finding is raised.
	Description string `json:"description"`

	// Type classifies the finding.
	Type FindingType `json:"type"`
}

// FindingType classifies a finding.
type FindingType struct {
	// UUID is the unique identifier of the finding type.
	UUID string `json:"uuid"`

	// DisplayName is the human-friendly name of the finding type.
	DisplayName string `json:"displayName"`

	// Description is an optional longer description of the finding type.
	// +optional
	Description string `json:"description,omitempty"`
}

// FindingRuleStatus defines the observed state of FindingRule
type FindingRuleStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// FindingRule is the Schema for the findingrules API
type FindingRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FindingRuleSpec   `json:"spec,omitempty"`
	Status FindingRuleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FindingRuleList contains a list of FindingRule
type FindingRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FindingRule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FindingRule{}, &FindingRuleList{})
}
