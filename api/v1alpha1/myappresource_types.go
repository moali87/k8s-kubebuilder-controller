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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type SpecImage struct {
    Repository string `json:"repository"`
    Tag string `json:"tag"`
}

type SpecResources struct {
    MemoryLimit string `json:"memoryLimit,omitempty"`
    CPURequest string `json:"cpuLimit,omitempty"`
}

type SpecRedis struct {
    // Redis enabled or disabled.  Default false.
    // +optional
    Enabled string `json:"enabled,omitempty"`
}

type SpecUI struct {
    // Color Hex value of color
    Color string `json:"color"`

    // Message value to print to screen from podInfo
    Message string `json:"message"`
}
// MyAppResourceSpec defines the desired state of MyAppResource
type MyAppResourceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// // Foo is an example field of MyAppResource. Edit myappresource_types.go to remove/update
	// Foo string `json:"foo,omitempty"`

    // kubebuilder:validation:Minimum=0

    // Image repository details
    Image SpecImage `json:"image"`

    // ReplicaCount field specifies how many podInfo containers should be running.  Default 1
    // +optional
    ReplicaCount *int32 `json:"replicaCount,omitempty"`

    // Resources (CPU/MEM) for pod/container
    Resources SpecResources `json:"resources"`

    // Redis enabled or disabled.  Default false
    Redis SpecRedis `json:"redis"`

    // UI display config for podInfo
    UI SpecUI `json:"ui"`
}

// MyAppResourceStatus defines the observed state of MyAppResource
type MyAppResourceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

    // ReplicaCount is the number of running replicas for podInfo container
    // Created with MyAppResource kind
    // +optional
    // +kubebuilder:validation:Minimum=0
    ReplicaCount int32 `json:"runningReplicas,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MyAppResource is the Schema for the myappresources API
type MyAppResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MyAppResourceSpec   `json:"spec,omitempty"`
	Status MyAppResourceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MyAppResourceList contains a list of MyAppResource
type MyAppResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MyAppResource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MyAppResource{}, &MyAppResourceList{})
}
