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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SecretType is the type of derived secret
// +kubebuilder:validation:Enum=password;encryption-key;custom
type SecretType string

const (
	// SecretTypePassword generates a 26-character password
	SecretTypePassword SecretType = "password"
	// SecretTypeEncryptionKey generates a 48-character encryption key
	SecretTypeEncryptionKey SecretType = "encryption-key"
	// SecretTypeCustom generates a secret of custom length
	SecretTypeCustom SecretType = "custom"
)

// DerivedKeySpec defines how to derive a single key
type DerivedKeySpec struct {
	// Type is the type of secret to generate
	// +kubebuilder:validation:Required
	Type SecretType `json:"type"`

	// MasterPassword is the name of the MasterPassword to use
	// +optional
	// +kubebuilder:default="default"
	MasterPassword string `json:"masterPassword,omitempty"`

	// Length is the length of the generated secret (only for custom type)
	// +optional
	// +kubebuilder:validation:Minimum=22
	// +kubebuilder:validation:Maximum=256
	Length int `json:"length,omitempty"`
}

// DerivedSecretSpec defines the desired state of DerivedSecret
type DerivedSecretSpec struct {
	// Type is the type of secret to create
	// +optional
	// +kubebuilder:default=Opaque
	Type corev1.SecretType `json:"type,omitempty"`

	// Annotations to apply to the generated secret
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels to apply to the generated secret
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Keys is a map of key names to their derivation specifications
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinProperties=1
	Keys map[string]DerivedKeySpec `json:"keys"`
}

// DerivedSecretStatus defines the observed state of DerivedSecret.
type DerivedSecretStatus struct {
	// SecretName is the name of the generated secret
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// Ready indicates whether the derived secret is ready
	// +optional
	Ready bool `json:"ready,omitempty"`

	// LastUpdated is the last time the secret was updated
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`

	// KeyHashes contains hash values (0-999) for each derived key to track updates without revealing passwords
	// +optional
	KeyHashes map[string]int `json:"keyHashes,omitempty"`

	// Conditions represent the current state of the DerivedSecret resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Secret",type=string,JSONPath=`.status.secretName`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DerivedSecret is the Schema for the derivedsecrets API
type DerivedSecret struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of DerivedSecret
	// +required
	Spec DerivedSecretSpec `json:"spec"`

	// status defines the observed state of DerivedSecret
	// +optional
	Status DerivedSecretStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// DerivedSecretList contains a list of DerivedSecret
type DerivedSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DerivedSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DerivedSecret{}, &DerivedSecretList{})
}
