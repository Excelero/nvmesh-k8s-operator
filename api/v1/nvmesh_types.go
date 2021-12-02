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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Core",type=string,JSONPath=`.spec.core.version`
// +kubebuilder:printcolumn:name="Mgmt",type=string,JSONPath=`.spec.management.version`
// +kubebuilder:printcolumn:name="CSI",type=string,JSONPath=`.spec.csi.version`
// +kubebuilder:printcolumn:name="TCP",type=boolean,JSONPath=`.spec.core.tcpOnly`,priority=10
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.reconcileStatus.status`,priority=10
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.reconcileStatus.status`,priority=10

// Represents a NVMesh Cluster
type NVMesh struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +required
	Spec   NVMeshSpec   `json:"spec"`
	Status NVMeshStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NVMeshList contains a list of NVMesh
type NVMeshList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NVMesh `json:"items"`
}

type NVMeshCore struct {
	//The version of NVMesh Core to be deployed. to perform an upgrade simply update this value to the required version.
	// +required
	Version string `json:"version"`

	//The address of the image registry where the nvmesh core images are stored
	// +optional
	ImageRegistry string `json:"imageRegistry,omitempty"`

	//The version tag of the nvmesh core docker images
	// +optional
	ImageVersionTag string `json:"imageVersionTag"`

	// Disabled - if true NVMesh Core will not be deployed
	// +optional
	Disabled bool `json:"disabled,omitempty"`

	// ConfiguredNICs - a comma seperated list of nics to use with NVMesh
	// +optional
	ConfiguredNICs string `json:"configuredNICs,omitempty"`

	// TCP Only - Set to true if cluster support only TCP, If false or omitted Infiniband is used
	// +optional
	TCPOnly bool `json:"tcpOnly,omitempty"`

	// Azure Optimized - Make optimizations for running on Azure cloud
	// +optional
	AzureOptimized bool `json:"azureOptimized,omitempty"`

	// Exclude NVMe Drives - Define which NVMe drives should not be used by NVMesh
	// +optional
	ExcludeDrives *ExcludeNVMeDrivesSpec `json:"excludeDrives,omitempty"`

	ModuleParams string `json:"moduleParams,omitempty"`
}

type ExcludeNVMeDrivesSpec struct {
	// A list of NVMe drive serial numbers that should not be used by the NVMesh software. i.e. S3HCNX4K123456
	// +optional
	SerialNumbers []string `json:"serialNumbers,omitempty"`

	// A list of device paths that should not be used by the NVMesh software, These devices will be excluded from each node. i.e. /dev/nvme1n1
	// +optional
	DevicePaths []string `json:"devicePaths,omitempty"`
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
	Replicas int32 `json:"replicas,omitempty"`

	//Overrides fields in the MongoDB data PVC
	// +optional
	DataVolumeClaim v1.PersistentVolumeClaimSpec `json:"dataVolumeClaim,omitempty"`
}

type NVMeshManagement struct {
	//The version of NVMesh Management to be deployed. to perform an upgrade simply update this value to the required version.
	// +required
	Version string `json:"version"`

	//The address of the image registry where the nvmesh management image is stored
	// +optional
	ImageRegistry string `json:"imageRegistry,omitempty"`

	//The number of replicas of the NVMesh Managemnet
	// +kubebuilder:validation:Minimum=1
	// +required
	Replicas int32 `json:"replicas,omitempty"`

	//Configuration for deploying a MongoDB cluster"
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

	//Overrides fields in the Management Backups PVC
	// +optional
	BackupsVolumeClaim v1.PersistentVolumeClaimSpec `json:"backupsVolumeClaim,omitempty"`

	// Disable Auto-Format NVMe drives as they are discovered
	// +optional
	DisableAutoFormatDrives bool `json:"disableAutoFormatDrives,omitempty"`

	// Disable Auto-Evict Missing NVMe drives - This enables NVMesh to auto-rebuild volumes when drives were replaced (for example on the cloud after a machine was restarted)
	// +optional
	DisableAutoEvictDrives bool `json:"disableAutoEvictDrives,omitempty"`
}

type NVMeshCSI struct {
	//The version of the NVMesh CSI Controller which will be deployed. To perform an upgrade simply update this value to the required version.
	// +required
	Version string `json:"version"`

	//The number of replicas for the NVMesh CSI Controller Statefulset
	// +kubebuilder:validation:Minimum=1
	// +optional
	ControllerReplicas int32 `json:"controllerReplicas,omitempty"`

	//Optional, if given will override the default image registry
	// +optional
	ImageRegistry string `json:"imageRegistry,omitempty"`

	//If true NVMesh CSI Driver will not be deployed
	// +optional
	Disabled bool `json:"disabled,omitempty"`
}

type NVMeshOperatorSpec struct {
	// If IgnoreVolumeAttachmentOnDelete is true, The operator will allow deleting this cluster when there are active attachments of NVMesh volumes. This can lead to an unclean state left on the k8s cluster
	IgnoreVolumeAttachmentOnDelete bool `json:"ignoreVolumeAttachmentOnDelete,omitempty"`

	// If IgnorePersistentVolumesOnDelete is true, The operator will allow deleting this cluster when there are NVMesh PersistentVolumes on the cluster. This can lead to an unclean state left on the k8s cluster
	IgnorePersistentVolumesOnDelete bool `json:"ignorePersistentVolumesOnDelete,omitempty"`

	// If SkipUninstall is true, The operator will not clear the mongo db or remove files the NVMesh software has saved locally on the nodes. This can lead to an unclean state left on the k8s cluster
	SkipUninstall bool `json:"skipUninstall,omitempty"`

	// Override the default file server for compiled binaries
	FileServer *OperatorFileServerSpec `json:"fileServer,omitempty"`
}

type OperatorFileServerSpec struct {
	// The url address of the binaries file server
	Address string `json:"address,omitempty"`

	// Allows to connect to a self signed https server
	SkipCheckCertificate bool `json:"skipCheckCertificate,omitempty"`
}

type ClusterAction struct {
	// The type of action to perform
	// +kubebuilder:validation:Enum=collect-logs
	// +kubebuilder:validation:Required
	// +required
	Name string `json:"name"`

	// Arguments for the Action
	// +optional
	Args map[string]string `json:"args,omitempty"`
}

// NVMeshDebugOptions - Operator Debug Options
type DebugOptions struct {

	// If true will try to pull all images even if they exist locally. For use when the same image with the same tag was updated
	ImagePullPolicyAlways bool `json:"imagePullPolicyAlways,omitempty"`

	// Prevent containers from exiting on each error causing the pod to be restarted
	ContainersKeepRunningAfterFailure bool `json:"containersKeepRunningAfterFailure,omitempty"`

	// Makes logs collector job stay running for debugging
	CollectLogsJobsRunForever bool `json:"collectLogsJobsRunForever,omitempty"`

	// Adds additional debug prints to jobs for actions and uninstall processes
	DebugJobs bool `json:"debugJobs,omitempty"`
}

// NVMeshSpec defines the desired state of NVMesh
type NVMeshSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Controls deployment of NVMesh-Core components
	Core NVMeshCore `json:"core"`

	// Controls deployment of NVMesh-Management
	Management NVMeshManagement `json:"management"`

	// Controls deployment of NVMesh CSI Driver
	CSI NVMeshCSI `json:"csi"`

	// Control the behavior of the NVMesh operator for this NVMesh Cluster
	// +optional
	Operator NVMeshOperatorSpec `json:"operator,omitempty"`

	// Debug - debug options
	// +optional
	Debug DebugOptions `json:"debug,omitempty"`

	// Initiate actions such as collecting logs
	Actions []ClusterAction `json:"actions,omitempty"`
}

type ActionStatus map[string]string

// NVMeshStatus defines the observed state of NVMesh
type NVMeshStatus struct {
	// define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The URL of NVMesh Web GUI
	WebUIURL string `json:"WebUIURL,omitempty"`

	ReconcileStatus ReconcileStatus `json:"reconcileStatus,omitempty"`

	ActionsStatus map[string]ActionStatus `json:"actionsStatus,omitempty"`
}

type ReconcileStatus struct {
	LastUpdate metav1.Time `json:"lastUpdate,omitempty"`
	Reason     string      `json:"reason,omitempty"`
	Status     string      `json:"status,omitempty"`
}

func init() {
	SchemeBuilder.Register(&NVMesh{}, &NVMeshList{})
}
