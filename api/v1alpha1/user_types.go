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

type UserState string

const (
	WaitingForApprovalUserState UserState = "WaitingForApproval"
	ActiveUserState             UserState = "Active"
	SuspendedUserState          UserState = "Suspended"
	BannedUserState             UserState = "Banned"
)

// UserSpec defines the desired state of User
type UserSpec struct {
	//+required
	Email string `json:"email"`
	//+required
	Username string `json:"username"`

	//+optional
	//+kubebuilder:default:="WaitingForApproval"
	//+kubebuilder:validation:Enum:=WaitingForApproval;Active;Suspended;Banned
	State UserState `json:"state,omitempty"`
	//+optional
	Expiration *metav1.Time `json:"expiration,omitempty"`

	//+optional
	DisplayName *string `json:"displayName,omitempty"`
	//+optional
	GivenName *string `json:"givenName,omitempty"`
	//+optional
	FamilyName *string `json:"familyName,omitempty"`

	//+optional
	Company *string `json:"company,omitempty"`
	//+optional
	SecondaryMail *string `json:"secondaryMail,omitempty"`
}

// UserStatus defines the observed state of User
type UserStatus struct {
	// InitialGeneration is the first observed resource generation
	InitialGeneration *int64 `json:"initialGeneration,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// User is the Schema for the users API
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec,omitempty"`
	Status UserStatus `json:"status,omitempty"`
}

func (u User) IsNewUser() bool {
	return u.Status.InitialGeneration == nil ||
		*u.Status.InitialGeneration == u.ObjectMeta.Generation
}

//+kubebuilder:object:root=true

// UserList contains a list of User
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}
