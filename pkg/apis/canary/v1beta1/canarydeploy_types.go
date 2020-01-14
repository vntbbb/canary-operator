package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CanaryDeploySpec defines the desired state of CanaryDeploy
// +k8s:openapi-gen=true
type CanaryDeploySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	DeployRef DeployReference `json:"deployReference"`
	DeployReplicas int32 `json:"deployReplicas"`
	CanaryReplicas int32 `json:"canaryReplicas"`
	CanaryVersion string `json:"canaryVersion"`
}

//DeployRef defines reference to target deploy
type DeployReference struct {
	Kind string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Namespace string `json:"namespace"`
	Name string `json:"name"`
}

// CanaryDeployStatus defines the observed state of CanaryDeploy
// +k8s:openapi-gen=true
type CanaryDeployStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	DeployReplicas int32 `json:"deployReplicas"`
	CanaryReplicas int32 `json:"canaryReplicas"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CanaryDeploy is the Schema for the canarydeploys API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=canarydeploys,scope=Namespaced
type CanaryDeploy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CanaryDeploySpec   `json:"spec,omitempty"`
	Status CanaryDeployStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CanaryDeployList contains a list of CanaryDeploy
type CanaryDeployList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CanaryDeploy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CanaryDeploy{}, &CanaryDeployList{})
}
