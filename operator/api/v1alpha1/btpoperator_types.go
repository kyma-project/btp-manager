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
	"github.com/kyma-project/module-manager/pkg/types"
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

	Spec   BtpOperatorSpec `json:"spec,omitempty"`
	Status types.Status    `json:"status,omitempty"`
}

// BtpOperatorSpec defines the desired state of BtpOperator
type BtpOperatorSpec struct{}

var _ types.CustomObject = &BtpOperator{}

func (o *BtpOperator) ComponentName() string {
	return componentName
}

func (o *BtpOperator) GetStatus() types.Status {
	return o.Status
}

func (o *BtpOperator) SetStatus(status types.Status) {
	o.Status = status
}

func (o *BtpOperator) IsReasonStringEqual(reason string) bool {
	var condition *metav1.Condition
	if len(o.Status.Conditions) > 0 {
		condition = o.Status.Conditions[0]
	}
	return condition != nil && condition.Reason == reason
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
