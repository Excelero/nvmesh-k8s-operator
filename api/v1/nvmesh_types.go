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
	//The version of NVMesh Core to be deployed. to perform an upgrade simply update this value to the required version.
	// +required
	Version string `json:"version"`

	//The address of the image registry where the nvmesh core images are stored
	// +optional
	ImageRegistry string `json:"imageRegistry"`

	// Disabled - if true NVMesh Core will not be deployed
	// +optional
	Disabled bool `json:"disabled,omitempty"`
}

type MongoDBCluster struct {
	// External - if true MongoDB is expected to be already deployed, and MongoAddress should be given, if false - MongoDB will be automatically deployed
	// +optional
	External bool `json:"external,omitempty"`

	//The MongoDB connection string i.e "mongo-0.mongo.nvmesh.svc.local:27017"
	// +optional
	Address string `json:"address,omitempty"`

	//The number of MongoDB replicas in the MongoDB Cluster - This field is ignored if management.mongoDB.external=true
	// +optional
	Replicas string `json:"replicas,omitempty"`
}

type NVMeshManagement struct {
	//The version of NVMesh Management to be deployed. to perform an upgrade simply update this value to the required version.
	// +required
	Version string `json:"version,omitempty"`

	//The address of the image registry where the nvmesh management image is stored
	// +optional
	ImageRegistry string `json:"imageRegistry"`

	//The number of replicas of the NVMesh Managemnet
	// +kubebuilder:validation:Minimum=1
	// +required
	Replicas int32 `json:"replicas,omitempty"`

	//Configuration for deploying a MongoDB cluster"
	// +optional
	MongoDB MongoDBCluster `json:"mongoDB,omitempty"`

	//The ExternalIP that will be used for the management GUI service LoadBalancer
	// +optional
	ExternalIPs []string `json:"externalIPs,omitempty"`

	// Disable TLS/SSL on NVMesh-Management websocket and HTTP connections
	// +optional
	NoSSL bool `json:"noSSL,omitempty"`

	// Disabled - if true NVMesh Management will not be deployed
	// +optional
	Disabled bool `json:"disabled,omitempty"`
}

type NVMeshCSI struct {
	//ControllerReplicas describes the number of replicas for the NVMesh CSI Controller Statefulset
	// +kubebuilder:validation:Minimum=1
	// +optional
	ControllerReplicas int32 `json:"controllerReplicas,omitempty"`

	//Version controls which version of the NVMesh CSI Controller will be deployed. to perform an upgrade simply update this value to the required version.
	// +optional
	Version string `json:"version,omitempty"`

	//ImageName - Optional, if given will override the default repositroy/image-name
	// +optional
	ImageName string `json:"imageName,omitempty"`

	// Disabled - if true NVMesh CSI Driver will not be deployed
	// +optional
	Disabled bool `json:"disabled,omitempty"`
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
	// +optional
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
