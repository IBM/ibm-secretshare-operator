//
// Copyright 2020 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MemberPhase identifies the status of the
type MemberPhase string

// Constants are used for status management
const (
	Running MemberPhase = "Running"
	Failed  MemberPhase = "Failed"
	None    MemberPhase = ""
)

// SecretShareSpec defines the desired state of SecretShare
type SecretShareSpec struct {
	// Secretshares defines a list of secret sharing information
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Secretshares []Secretshare `json:"secretshares,omitempty"`
	// Configmapshares defines a list of configmap sharing information
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Configmapshares []Configmapshare `json:"configmapshares,omitempty"`
}

// TargetNamespace identifies the namespace the secret/configmap will be shared to
type TargetNamespace struct {
	// Namespace is the target namespace of the secret or configmap
	Namespace string `json:"namespace"`
}

// Secretshare identifies a secret required to be shared to another namespace
type Secretshare struct {
	// Secretname is the name of the secret waiting for sharing
	Secretname string `json:"secretname"`
	// Sharewith is a list of the target namespace for sharing
	Sharewith []TargetNamespace `json:"sharewith"`
}

// Configmapshare identifies a Configmap required to be shared to another namespace
type Configmapshare struct {
	// Configmapname is the name of the configmap waiting for sharing
	Configmapname string `json:"configmapname"`
	// Sharewith is a list of the target namespace for sharing
	Sharewith []TargetNamespace `json:"sharewith"`
}

// SecretShareStatus defines the observed state of SecretShare
type SecretShareStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	// Members represnets the current operand status of the set
	// +optional
	Members map[string]MemberPhase `json:"members,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SecretShare is the Schema for the secretshares API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=secretshares,scope=Namespaced
type SecretShare struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecretShareSpec   `json:"spec,omitempty"`
	Status SecretShareStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SecretShareList contains a list of SecretShare
type SecretShareList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretShare `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecretShare{}, &SecretShareList{})
}

// InitStatus check the status of a secret/configmap
func (r *SecretShare) InitStatus() bool {
	if r.Status.Members == nil {
		r.Status.Members = map[string]MemberPhase{}
		return true
	}
	return false
}

// UpdateStatus updates the status of a secret/configmap
func (r *SecretShare) UpdateStatus(namespacedName string, status MemberPhase) bool {
	if r.Status.Members[namespacedName] == status {
		return false
	}
	r.Status.Members[namespacedName] = status
	return true
}

// CheckStatus check the status of a secret/configmap
func (r *SecretShare) CheckStatus(namespacedName string, status MemberPhase) bool {
	if len(r.Status.Members) == 0 {
		return false
	}
	if r.Status.Members[namespacedName] == status {
		return true
	}
	return false
}

// RemoveStatus removes the status of a secret/configmap
func (r *SecretShare) RemoveStatus(namespacedName string) {
	delete(r.Status.Members, namespacedName)
}
