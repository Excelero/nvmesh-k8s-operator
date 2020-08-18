/*


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

type NVMeshCore struct {
	// Deploy controls wether to deploy NVMesh Core
	Deploy bool `json:"deploy,omitempty"`

	//The version of NVMesh Core to be deployed. to perform an upgrade simply update this value to the required version.
	Version string `json:"version"`
}

type NVMeshManagement struct {
	// Deploy controls wether to deploy NVMesh Management
	Deploy bool `json:"deploy,omitempty"`

	//The version of NVMesh Management to be deployed. to perform an upgrade simply update this value to the required version.
	Version string `json:"version"`

	//The number of replicas of the NVMesh Managemnet
	Replicas int32 `json:"replica"`

	// DeployMongo controls wether to deploy a MongoDB Operator for NVMesh Management
	DeployMongo bool `json:"deployMongo,omitempty"`
}

type NVMeshCSI struct {
	// Deploy controls wether the NVMesh CSI Driver should be deployed or not
	Deploy bool `json:"deploy,omitempty"`

	//ControllerReplicas describes the number of replicas for the NVMesh CSI Controller Statefulset
	ControllerReplicas int32 `json:"controller,omitempty"`

	//Version controls which version of the NVMesh CSI Controller will be deployed. to perform an upgrade simply update this value to the required version.
	Version string `json:"version"`
}

// NVMeshSpec defines the desired state of NVMesh
type NVMeshSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Core is an object describing the nvmesh-core deployment
	Core NVMeshCore `json:"core,omitempty"`

	// Management is an object describing the nvmesh-management deployment
	Management NVMeshManagement `json:"management,omitempty"`

	// CSI is an object describing the nvmesh-csi-driver deployment
	CSI NVMeshCSI `json:"csi,omitempty"`
}

// NVMeshStatus defines the observed state of NVMesh
type NVMeshStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// NVMesh is the Schema for the nvmeshes API
type NVMesh struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NVMeshSpec   `json:"spec,omitempty"`
	Status NVMeshStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NVMeshList contains a list of NVMesh
type NVMeshList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NVMesh `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NVMesh{}, &NVMeshList{})
}
