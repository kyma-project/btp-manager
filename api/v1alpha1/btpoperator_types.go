/*
Copyright 2022.

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

const componentName = "btp-operator"

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=".status.state"

// BtpOperator is the Schema for the btpoperators API
type BtpOperator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	//+nullable
	Spec   BtpOperatorSpec `json:"spec,omitempty"`
	Status Status          `json:"status,omitempty"`
}

// BtpOperatorSpec defines the desired state of BtpOperator
type BtpOperatorSpec struct{}

type State string

// Valid CustomObject States.
const (
	// StateReady signifies CustomObject is ready and has been installed successfully.
	StateReady State = "Ready"

	// StateProcessing signifies CustomObject is reconciling and is in the process of installation.
	// Processing can also signal that the Installation previously encountered an error and is now recovering.
	StateProcessing State = "Processing"

	// StateWarning signifies a warning for CustomObject. This signifies that the Installation
	// process encountered a problem.
	StateWarning State = "Warning"

	// StateError signifies an error for CustomObject. This signifies that the Installation
	// process encountered an error.
	// Contrary to Processing, it can be expected that this state should change on the next retry.
	StateError State = "Error"

	// StateDeleting signifies CustomObject is being deleted. This is the state that is used
	// when a deletionTimestamp was detected and Finalizers are picked up.
	StateDeleting State = "Deleting"
)

// Status defines the observed state of CustomObject.
// +k8s:deepcopy-gen=true
// Status defines the observed state of CustomObject.
type Status struct {
	// State signifies current state of CustomObject.
	// Value can be one of ("Ready", "Processing", "Error", "Deleting", "Warning").
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Processing;Deleting;Ready;Error;Warning
	State State `json:"state"`

	// Conditions associated with CustomStatus.
	Conditions []*metav1.Condition `json:"conditions,omitempty"`
}

func (s *Status) WithState(state State) Status {
	s.State = state
	return *s
}

// LastOperation defines the last operation from the control-loop.
// +k8s:deepcopy-gen=true
type LastOperation struct {
	Operation      string      `json:"operation"`
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}

type Resource struct {
	Name                    string `json:"name"`
	Namespace               string `json:"namespace"`
	metav1.GroupVersionKind `json:",inline"`
}

func (o *BtpOperator) ComponentName() string {
	return componentName
}

func (o *BtpOperator) GetStatus() Status {
	return o.Status
}

func (o *BtpOperator) SetStatus(status Status) {
	o.Status = status
}

func (o *BtpOperator) IsReasonStringEqual(reason string) bool {
	for _, cnd := range o.Status.Conditions {
		if cnd != nil && cnd.Reason == reason {
			return true
		}
	}
	return false
}

func (o *BtpOperator) IsMsgForGivenReasonEqual(reason, message string) bool {
	for _, cnd := range o.Status.Conditions {
		if cnd != nil && cnd.Reason == reason && cnd.Message == message {
			return true
		}
	}
	return false
}

//+kubebuilder:object:root=true

// BtpOperatorList contains a list of BtpOperator
type BtpOperatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BtpOperator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BtpOperator{}, &BtpOperatorList{})
}
