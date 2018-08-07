package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ZenkoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Zenko `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Zenko struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ZenkoSpec   `json:"spec"`
	Status            ZenkoStatus `json:"status,omitempty"`
}

type ZenkoSpec struct {
	AppVersion string `json:"appVersion,requireed"`
	NodeCount  int    `json:"nodeCount"`
}
type ZenkoStatus struct {
	InstanceID                types.UID `json:"instanceID"`
	DeployedVersion           string    `json:"deployedVersion"`
	DeployedConfigurationHash string    `json:"deployedConfigurationHash"`
}
