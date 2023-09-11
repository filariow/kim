/*
Copyright 2023 Francesco Ilario.

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

// PersonalAccessTokenSpec defines the desired state of PersonalAccessToken
type PersonalAccessTokenSpec struct {
	// PersonalAccessToken validity
	Deadline *metav1.Timestamp `json:"deadline,omitempty"`
}

// PersonalAccessTokenStatus defines the observed state of PersonalAccessToken
type PersonalAccessTokenStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PersonalAccessToken is the Schema for the personalaccesstokens API
type PersonalAccessToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PersonalAccessTokenSpec   `json:"spec,omitempty"`
	Status PersonalAccessTokenStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PersonalAccessTokenList contains a list of PersonalAccessToken
type PersonalAccessTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PersonalAccessToken `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PersonalAccessToken{}, &PersonalAccessTokenList{})
}
