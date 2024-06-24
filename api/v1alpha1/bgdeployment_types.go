package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BGDeploymentSpec defines the desired state of BGDeployment
type BGDeploymentSpec struct {
	Replicas int32 `json:"replicas"`
	Image string `json:"image"`
}

// BGDeploymentStatus defines the observed state of BGDeployment
type BGDeploymentStatus struct {
	ActiveColor string `json:"activeColor"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BGDeployment is the Schema for the bgdeployments API
type BGDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BGDeploymentSpec   `json:"spec,omitempty"`
	Status BGDeploymentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BGDeploymentList contains a list of BGDeployment
type BGDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BGDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BGDeployment{}, &BGDeploymentList{})
}
