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
	Running    MemberPhase = "Running"
	Failed     MemberPhase = "Failed"
	NotFound   MemberPhase = "NotFound"
	NotEnabled MemberPhase = "NotEnabled"
	None       MemberPhase = ""
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

// SecretShareStatus defines the observed status of SecretShare
type SecretShareStatus struct {
	// Members represnets the current operand status of the set
	// +optional
	Members SecretConfigmapMembers `json:"members,omitempty"`
}

// SecretConfigmapMembers defines the observed status of SecretShare
type SecretConfigmapMembers struct {
	// SecretMembers represnets the current operand status of the set
	// +optional
	SecretMembers map[string]MemberPhase `json:"secretMembers,omitempty"`

	// ConfigmapMembers represnets the current operand status of the set
	// +optional
	ConfigmapMembers map[string]MemberPhase `json:"configmapMembers,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SecretShare is the Schema for the secretshares API
type SecretShare struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecretShareSpec   `json:"spec,omitempty"`
	Status SecretShareStatus `json:"status,omitempty"`
}

// InitStatus check the status of a secret/configmap
func (r *SecretShare) InitStatus() {
	if r.Status.Members.ConfigmapMembers == nil {
		r.Status.Members.ConfigmapMembers = map[string]MemberPhase{}
	}
	if r.Status.Members.SecretMembers == nil {
		r.Status.Members.SecretMembers = map[string]MemberPhase{}
	}
}

// UpdateSecretStatus updates the status of a secret
func (r *SecretShare) UpdateSecretStatus(namespacedName string, status MemberPhase) bool {
	if r.Status.Members.SecretMembers[namespacedName] == status {
		return false
	}
	r.Status.Members.SecretMembers[namespacedName] = status
	return true
}

// CheckSecretStatus check the status of a secret
func (r *SecretShare) CheckSecretStatus(namespacedName string, status MemberPhase) bool {
	if len(r.Status.Members.SecretMembers) == 0 {
		return false
	}
	if r.Status.Members.SecretMembers[namespacedName] == status {
		return true
	}
	return false
}

// RemoveSecretStatus removes the status of a secret
func (r *SecretShare) RemoveSecretStatus(namespacedName string) {
	delete(r.Status.Members.SecretMembers, namespacedName)
}

// UpdateConfigmapStatus updates the status of a configmap
func (r *SecretShare) UpdateConfigmapStatus(namespacedName string, status MemberPhase) bool {
	if r.Status.Members.ConfigmapMembers[namespacedName] == status {
		return false
	}
	r.Status.Members.ConfigmapMembers[namespacedName] = status
	return true
}

// CheckConfigmapStatus check the status of a configmap
func (r *SecretShare) CheckConfigmapStatus(namespacedName string, status MemberPhase) bool {
	if len(r.Status.Members.ConfigmapMembers) == 0 {
		return false
	}
	if r.Status.Members.ConfigmapMembers[namespacedName] == status {
		return true
	}
	return false
}

// RemoveConfigmapStatus removes the status of a configmap
func (r *SecretShare) RemoveConfigmapStatus(namespacedName string) {
	delete(r.Status.Members.ConfigmapMembers, namespacedName)
}

// +kubebuilder:object:root=true

// SecretShareList contains a list of SecretShare
type SecretShareList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretShare `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecretShare{}, &SecretShareList{})
}
