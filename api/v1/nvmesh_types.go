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

package v1

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

	//The address of the image registry where the nvmesh core images are stored
	ImageRegistry string `json:"imageRegistry"`
}

type MongoDBCluster struct {
	// Deploy MongoDB for NVMesh Management, if this is true a MongoDB cluster will automatically be deployed. If this is false the Management server will try to connect to an external MongoDB cluster using the address defined NVMesh.Spec.Management.MongoAddress
	Deploy bool `json:"deploy,omitempty"`
}

type NVMeshManagement struct {
	// Deploy controls wether to deploy NVMesh Management
	Deploy bool `json:"deploy,omitempty"`

	//The version of NVMesh Management to be deployed. to perform an upgrade simply update this value to the required version.
	Version string `json:"version,omitempty"`

	//The address of the image registry where the nvmesh management image is stored
	ImageRegistry string `json:"imageRegistry"`

	//The number of replicas of the NVMesh Managemnet
	// +kubebuilder:validation:Minimum=1
	Replicas int32 `json:"replicas,omitempty"`

	//The MongoDB connection string i.e "mongo-0.mongo.nvmesh.svc.local:27017"
	MongoAddress string `json:"mongoAddress,omitempty"`

	//Configuration for deploying a MongoDB cluster"
	MongoDB MongoDBCluster `json:"mongoDB,omitempty"`

	//The ExternalIP that will be used for the management GUI service LoadBalancer
	ExternalIPs []string `json:"externalIPs,omitempty"`

	// Wether the management should a secure TLS/SSL connection on websocket and HTTP connections
	UseSSL bool `json:"useSSL,omitempty"`
}

type NVMeshCSI struct {
	// Deploy controls wether the NVMesh CSI Driver should be deployed or not
	Deploy bool `json:"deploy,omitempty"`

	//ControllerReplicas describes the number of replicas for the NVMesh CSI Controller Statefulset
	// +kubebuilder:validation:Minimum=1
	ControllerReplicas int32 `json:"controllerReplicas,omitempty"`

	//Version controls which version of the NVMesh CSI Controller will be deployed. to perform an upgrade simply update this value to the required version.
	Version string `json:"version,omitempty"`

	//ImageName - Optional, if given will override the default repositroy/image-name
	ImageName string `json:"imageName,omitempty"`
}

type NVMeshOperatorSpec struct {
	// If IgnoreVolumeAttachmentOnDelete is true, The operator will allow deleting this cluster when there are active attachments of NVMesh volumes. This can lead to an unclean state left on the k8s cluster
	IgnoreVolumeAttachmentOnDelete bool `json:"ignoreVolumeAttachmentOnDelete,omitempty"`

	// If IgnorePersistentVolumesOnDelete is true, The operator will allow deleting this cluster when there are NVMesh PersistentVolumes on the cluster. This can lead to an unclean state left on the k8s cluster
	IgnorePersistentVolumesOnDelete bool `json:"ignorePersistentVolumesOnDelete,omitempty"`
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

	// Control the behavior of the NVMesh operator for this NVMesh Cluster
	Operator NVMeshOperatorSpec `json:"operator,omitempty"`
}

// NVMeshStatus defines the observed state of NVMesh
type NVMeshStatus struct {
	// define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The URL of NVMesh Web GUI
	WebUIURL string `json:"WebUIURL,omitempty"`

	ReconcileStatus ReconcileStatus `json:"reconcileStatus,omitempty"`
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

type ReconcileStatus struct {
	LastUpdate metav1.Time `json:"lastUpdate,omitempty"`
	Reason     string      `json:"reason,omitempty"`
	Status     string      `json:"status,omitempty"`
}

func init() {
	SchemeBuilder.Register(&NVMesh{}, &NVMeshList{})
}
