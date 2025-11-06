/*
Copyright 2025.

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

// SecretReference defines the secret where the master password is stored
type SecretReference struct {
	// Name is the name of the secret
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Create indicates whether to create the secret if it doesn't exist
	// +optional
	// +kubebuilder:default=true
	Create bool `json:"create,omitempty"`
}

// MasterPasswordSpec defines the desired state of MasterPassword
type MasterPasswordSpec struct {
	// Length is the length of the generated master password
	// +optional
	// +kubebuilder:default=86
	// +kubebuilder:validation:Minimum=22
	// +kubebuilder:validation:Maximum=256
	Length int `json:"length,omitempty"`

	// Secret defines the secret where the master password is stored
	// If not specified, defaults to <name>-mp in the operator namespace
	// +optional
	Secret *SecretReference `json:"secret,omitempty"`

	// Annotations to apply to the generated secret
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// MasterPasswordStatus defines the observed state of MasterPassword.
type MasterPasswordStatus struct {
	// SecretName is the name of the secret containing the master password
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// SecretNamespace is the namespace of the secret containing the master password
	// +optional
	SecretNamespace string `json:"secretNamespace,omitempty"`

	// Ready indicates whether the master password secret is ready
	// +optional
	Ready bool `json:"ready,omitempty"`

	// DependentSecrets is the count of DerivedSecret resources using this MasterPassword
	// +optional
	DependentSecrets int `json:"dependentSecrets,omitempty"`

	// Conditions represent the current state of the MasterPassword resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Dependent Secrets",type=integer,JSONPath=`.status.dependentSecrets`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// MasterPassword is the Schema for the masterpasswords API
type MasterPassword struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of MasterPassword
	// +required
	Spec MasterPasswordSpec `json:"spec"`

	// status defines the observed state of MasterPassword
	// +optional
	Status MasterPasswordStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// MasterPasswordList contains a list of MasterPassword
type MasterPasswordList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MasterPassword `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MasterPassword{}, &MasterPasswordList{})
}
