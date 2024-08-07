/*
Copyright 2024.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PodRecordSpec defines the desired state of PodRecord
type PodRecordSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of PodRecord. Edit podrecord_types.go to remove/update

	UserID     string `json:"userID,omitempty"`
	PodID      string `json:"podID,omitempty"`
	PodName    string `json:"podName,omitempty"`
	CpuRequest string `json:"cpuRequest,omitempty"`
	MemRequest string `json:"memRequest,omitempty"`
	CpuLimit   string `json:"cpuLimit,omitempty"`
	MemLimit   string `json:"memLimit,omitempty"`
	Gpu        int64  `json:"gpu,omitempty"`
	Node       string `json:"node,omitempty"`
	NodeMem    string `json:"nodeMem,omitempty"`
	NodeCpu    string `json:"nodeCpu,omitempty"`
	StartTime  string `json:"startTime,omitempty"`
	EndTime    string `json:"endTime,omitempty"`
	EndStatus  string `json:"endStatus,omitempty"`
}

// PodRecordStatus defines the observed state of PodRecord
type PodRecordStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PodRecord is the Schema for the podrecords API
type PodRecord struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodRecordSpec   `json:"spec,omitempty"`
	Status PodRecordStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PodRecordList contains a list of PodRecord
type PodRecordList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PodRecord `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PodRecord{}, &PodRecordList{})
}
