Packages:
* nvmesh.excelero.com/v1* [NVMesh](#nvmesh-excelero-com-v1-nvmesh)
<h1 id="nvmesh.excelero.com/v1">nvmesh.excelero.com/v1</h1>
<div>
<p>Package v1 contains API Schema definitions for the nvmesh v1 API group</p>
</div>
<h3 id="nvmesh-excelero-com-v1-nvmesh">NVMesh
</h3>
<div>
<p>Represents a NVMesh Cluster</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
<span class="type">string<span>
</td>
<td>
<code>
nvmesh.excelero.com/v1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
<span class="type">string<span>
</td>
<td><code>NVMesh</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#ObjectMeta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#nvmesh-excelero-com-v1-nvmeshspec">
NVMeshSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>core</code><br/>
<em>
<a href="#nvmesh-excelero-com-v1-nvmeshcore">
NVMeshCore
</a>
</em>
</td>
<td>
<p>Controls deployment of NVMesh-Core components</p>
</td>
</tr>
<tr>
<td>
<code>management</code><br/>
<em>
<a href="#nvmesh-excelero-com-v1-nvmeshmanagement">
NVMeshManagement
</a>
</em>
</td>
<td>
<p>Controls deployment of NVMesh-Management</p>
</td>
</tr>
<tr>
<td>
<code>csi</code><br/>
<em>
<a href="#nvmesh-excelero-com-v1-nvmeshcsi">
NVMeshCSI
</a>
</em>
</td>
<td>
<p>Controls deployment of NVMesh CSI Driver</p>
</td>
</tr>
<tr>
<td>
<code>operator</code><br/>
<em>
<a href="#nvmesh-excelero-com-v1-nvmeshoperatorspec">
NVMeshOperatorSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Control the behavior of the NVMesh operator for this NVMesh Cluster</p>
</td>
</tr>
<tr>
<td>
<code>debug</code><br/>
<em>
<a href="#nvmesh-excelero-com-v1-debugoptions">
DebugOptions
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Debug - debug options</p>
</td>
</tr>
<tr>
<td>
<code>actions</code><br/>
<em>
<a href="#nvmesh-excelero-com-v1-clusteraction">
[]ClusterAction
</a>
</em>
</td>
<td>
<p>Initiate actions such as collecting logs</p>
</td>
</tr>
</tbody>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#nvmesh-excelero-com-v1-nvmeshstatus">
NVMeshStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="nvmesh-excelero-com-v1-actionstatus">ActionStatus
(<code>map[string]string</code> alias)</h3>
<div class="alert alert-info col-md-8"><i class="fa fa-info-circle"></i> Appears In:
<ul>
<li><a href="#nvmesh-excelero-com-v1-nvmeshstatus">NVMeshStatus</a></li>
</ul>
</div>
<div>
</div>
<h3 id="nvmesh-excelero-com-v1-clusteraction">ClusterAction
</h3>
<div class="alert alert-info col-md-8"><i class="fa fa-info-circle"></i> Appears In:
<ul>
<li><a href="#nvmesh-excelero-com-v1-nvmeshspec">NVMeshSpec</a></li>
</ul>
</div>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
<span class="type">string<span>
</em>
</td>
<td>
<p>The type of action to perform</p>
</td>
</tr>
<tr>
<td>
<code>args</code><br/>
<em>
<span class="type">map[string]string<span>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Arguments for the Action</p>
</td>
</tr>
</tbody>
</table>
<h3 id="nvmesh-excelero-com-v1-debugoptions">DebugOptions
</h3>
<div class="alert alert-info col-md-8"><i class="fa fa-info-circle"></i> Appears In:
<ul>
<li><a href="#nvmesh-excelero-com-v1-nvmeshspec">NVMeshSpec</a></li>
</ul>
</div>
<div>
<p>NVMeshDebugOptions - Operator Debug Options</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>imagePullPolicyAlways</code><br/>
<em>
<span class="type">bool<span>
</em>
</td>
<td>
<p>If true will try to pull all images even if they exist locally. For use when the same image with the same tag was updated</p>
</td>
</tr>
<tr>
<td>
<code>containersKeepRunningAfterFailure</code><br/>
<em>
<span class="type">bool<span>
</em>
</td>
<td>
<p>Prevent containers from exiting on each error causing the pod to be restarted</p>
</td>
</tr>
<tr>
<td>
<code>collectLogsJobsRunForever</code><br/>
<em>
<span class="type">bool<span>
</em>
</td>
<td>
<p>Makes logs collector job stay running for debugging</p>
</td>
</tr>
<tr>
<td>
<code>debugJobs</code><br/>
<em>
<span class="type">bool<span>
</em>
</td>
<td>
<p>Adds additional debug prints to jobs for actions and uninstall processes</p>
</td>
</tr>
</tbody>
</table>
<h3 id="nvmesh-excelero-com-v1-excludenvmedrivesspec">ExcludeNVMeDrivesSpec
</h3>
<div class="alert alert-info col-md-8"><i class="fa fa-info-circle"></i> Appears In:
<ul>
<li><a href="#nvmesh-excelero-com-v1-nvmeshcore">NVMeshCore</a></li>
</ul>
</div>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>serialNumbers</code><br/>
<em>
<span class="type">[]string<span>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A list of NVMe drive serial numbers that should not be used by the NVMesh software. i.e. S3HCNX4K123456</p>
</td>
</tr>
<tr>
<td>
<code>devicePaths</code><br/>
<em>
<span class="type">[]string<span>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A list of device paths that should not be used by the NVMesh software, These devices will be excluded from each node. i.e. /dev/nvme1n1</p>
</td>
</tr>
</tbody>
</table>
<h3 id="nvmesh-excelero-com-v1-mongodbcluster">MongoDBCluster
</h3>
<div class="alert alert-info col-md-8"><i class="fa fa-info-circle"></i> Appears In:
<ul>
<li><a href="#nvmesh-excelero-com-v1-nvmeshmanagement">NVMeshManagement</a></li>
</ul>
</div>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>external</code><br/>
<em>
<span class="type">bool<span>
</em>
</td>
<td>
<em>(Optional)</em>
<p>External - if true MongoDB is expected to be already deployed, and MongoAddress should be given, if false - MongoDB will be automatically deployed</p>
</td>
</tr>
<tr>
<td>
<code>useOperator</code><br/>
<em>
<span class="type">bool<span>
</em>
</td>
<td>
<em>(Optional)</em>
<p>If true the NVMesh Operator will deploy a MongoDB Operator and a MongoDB Cluster using a CustomResource</p>
</td>
</tr>
<tr>
<td>
<code>address</code><br/>
<em>
<span class="type">string<span>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The MongoDB connection string i.e &ldquo;mongo-0.mongo.nvmesh.svc.local:27017&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>replicas</code><br/>
<em>
<span class="type">int32<span>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The number of MongoDB replicas in the MongoDB Cluster - This field is ignored if management.mongoDB.external=true</p>
</td>
</tr>
<tr>
<td>
<code>dataVolumeClaim</code><br/>
<em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#persistentvolumeclaimspec-v1-core">
Kubernetes core/v1.PersistentVolumeClaimSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Overrides fields in the MongoDB data PVC</p>
</td>
</tr>
</tbody>
</table>
<h3 id="nvmesh-excelero-com-v1-nvmeshcsi">NVMeshCSI
</h3>
<div class="alert alert-info col-md-8"><i class="fa fa-info-circle"></i> Appears In:
<ul>
<li><a href="#nvmesh-excelero-com-v1-nvmeshspec">NVMeshSpec</a></li>
</ul>
</div>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>version</code><br/>
<em>
<span class="type">string<span>
</em>
</td>
<td>
<p>The version of the NVMesh CSI Controller which will be deployed. To perform an upgrade simply update this value to the required version.</p>
</td>
</tr>
<tr>
<td>
<code>controllerReplicas</code><br/>
<em>
<span class="type">int32<span>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The number of replicas for the NVMesh CSI Controller Statefulset</p>
</td>
</tr>
<tr>
<td>
<code>imageRegistry</code><br/>
<em>
<span class="type">string<span>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Optional, if given will override the default repositroy/image-name</p>
</td>
</tr>
<tr>
<td>
<code>disabled</code><br/>
<em>
<span class="type">bool<span>
</em>
</td>
<td>
<em>(Optional)</em>
<p>If true NVMesh CSI Driver will not be deployed</p>
</td>
</tr>
</tbody>
</table>
<h3 id="nvmesh-excelero-com-v1-nvmeshcore">NVMeshCore
</h3>
<div class="alert alert-info col-md-8"><i class="fa fa-info-circle"></i> Appears In:
<ul>
<li><a href="#nvmesh-excelero-com-v1-nvmeshspec">NVMeshSpec</a></li>
</ul>
</div>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>version</code><br/>
<em>
<span class="type">string<span>
</em>
</td>
<td>
<p>The version of NVMesh Core to be deployed. to perform an upgrade simply update this value to the required version.</p>
</td>
</tr>
<tr>
<td>
<code>imageRegistry</code><br/>
<em>
<span class="type">string<span>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The address of the image registry where the nvmesh core images are stored</p>
</td>
</tr>
<tr>
<td>
<code>imageVersionTag</code><br/>
<em>
<span class="type">string<span>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The version tag of the nvmesh core docker images</p>
</td>
</tr>
<tr>
<td>
<code>disabled</code><br/>
<em>
<span class="type">bool<span>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Disabled - if true NVMesh Core will not be deployed</p>
</td>
</tr>
<tr>
<td>
<code>configuredNICs</code><br/>
<em>
<span class="type">string<span>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ConfiguredNICs - a comma seperated list of nics to use with NVMesh</p>
</td>
</tr>
<tr>
<td>
<code>tcpOnly</code><br/>
<em>
<span class="type">bool<span>
</em>
</td>
<td>
<em>(Optional)</em>
<p>TCP Only - Set to true if cluster support only TCP, If false or omitted Infiniband is used</p>
</td>
</tr>
<tr>
<td>
<code>azureOptimized</code><br/>
<em>
<span class="type">bool<span>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Azure Optimized - Make optimizations for running on Azure cloud</p>
</td>
</tr>
<tr>
<td>
<code>excludeDrives</code><br/>
<em>
<a href="#nvmesh-excelero-com-v1-excludenvmedrivesspec">
ExcludeNVMeDrivesSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Exclude NVMe Drives - Define which NVMe drives should not be used by NVMesh</p>
</td>
</tr>
</tbody>
</table>
<h3 id="nvmesh-excelero-com-v1-nvmeshmanagement">NVMeshManagement
</h3>
<div class="alert alert-info col-md-8"><i class="fa fa-info-circle"></i> Appears In:
<ul>
<li><a href="#nvmesh-excelero-com-v1-nvmeshspec">NVMeshSpec</a></li>
</ul>
</div>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>version</code><br/>
<em>
<span class="type">string<span>
</em>
</td>
<td>
<p>The version of NVMesh Management to be deployed. to perform an upgrade simply update this value to the required version.</p>
</td>
</tr>
<tr>
<td>
<code>imageRegistry</code><br/>
<em>
<span class="type">string<span>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The address of the image registry where the nvmesh management image is stored</p>
</td>
</tr>
<tr>
<td>
<code>replicas</code><br/>
<em>
<span class="type">int32<span>
</em>
</td>
<td>
<p>The number of replicas of the NVMesh Managemnet</p>
</td>
</tr>
<tr>
<td>
<code>mongoDB</code><br/>
<em>
<a href="#nvmesh-excelero-com-v1-mongodbcluster">
MongoDBCluster
</a>
</em>
</td>
<td>
<p>Configuration for deploying a MongoDB cluster&rdquo;</p>
</td>
</tr>
<tr>
<td>
<code>externalIPs</code><br/>
<em>
<span class="type">[]string<span>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The ExternalIP that will be used for the management GUI service LoadBalancer</p>
</td>
</tr>
<tr>
<td>
<code>noSSL</code><br/>
<em>
<span class="type">bool<span>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Disable TLS/SSL on NVMesh-Management websocket and HTTP connections</p>
</td>
</tr>
<tr>
<td>
<code>disabled</code><br/>
<em>
<span class="type">bool<span>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Disabled - if true NVMesh Management will not be deployed</p>
</td>
</tr>
<tr>
<td>
<code>backupsVolumeClaim</code><br/>
<em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#persistentvolumeclaimspec-v1-core">
Kubernetes core/v1.PersistentVolumeClaimSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Overrides fields in the Management Backups PVC</p>
</td>
</tr>
</tbody>
</table>
<h3 id="nvmesh-excelero-com-v1-nvmeshoperatorspec">NVMeshOperatorSpec
</h3>
<div class="alert alert-info col-md-8"><i class="fa fa-info-circle"></i> Appears In:
<ul>
<li><a href="#nvmesh-excelero-com-v1-nvmeshspec">NVMeshSpec</a></li>
</ul>
</div>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ignoreVolumeAttachmentOnDelete</code><br/>
<em>
<span class="type">bool<span>
</em>
</td>
<td>
<p>If IgnoreVolumeAttachmentOnDelete is true, The operator will allow deleting this cluster when there are active attachments of NVMesh volumes. This can lead to an unclean state left on the k8s cluster</p>
</td>
</tr>
<tr>
<td>
<code>ignorePersistentVolumesOnDelete</code><br/>
<em>
<span class="type">bool<span>
</em>
</td>
<td>
<p>If IgnorePersistentVolumesOnDelete is true, The operator will allow deleting this cluster when there are NVMesh PersistentVolumes on the cluster. This can lead to an unclean state left on the k8s cluster</p>
</td>
</tr>
<tr>
<td>
<code>skipUninstall</code><br/>
<em>
<span class="type">bool<span>
</em>
</td>
<td>
<p>If SkipUninstall is true, The operator will not clear the mongo db or remove files the NVMesh software has saved locally on the nodes. This can lead to an unclean state left on the k8s cluster</p>
</td>
</tr>
<tr>
<td>
<code>fileServer</code><br/>
<em>
<a href="#nvmesh-excelero-com-v1-operatorfileserverspec">
OperatorFileServerSpec
</a>
</em>
</td>
<td>
<p>Override the default file server for compiled binaries</p>
</td>
</tr>
</tbody>
</table>
<h3 id="nvmesh-excelero-com-v1-nvmeshspec">NVMeshSpec
</h3>
<div class="alert alert-info col-md-8"><i class="fa fa-info-circle"></i> Appears In:
<ul>
<li><a href="#nvmesh-excelero-com-v1-nvmesh">NVMesh</a></li>
</ul>
</div>
<div>
<p>NVMeshSpec defines the desired state of NVMesh</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>core</code><br/>
<em>
<a href="#nvmesh-excelero-com-v1-nvmeshcore">
NVMeshCore
</a>
</em>
</td>
<td>
<p>Controls deployment of NVMesh-Core components</p>
</td>
</tr>
<tr>
<td>
<code>management</code><br/>
<em>
<a href="#nvmesh-excelero-com-v1-nvmeshmanagement">
NVMeshManagement
</a>
</em>
</td>
<td>
<p>Controls deployment of NVMesh-Management</p>
</td>
</tr>
<tr>
<td>
<code>csi</code><br/>
<em>
<a href="#nvmesh-excelero-com-v1-nvmeshcsi">
NVMeshCSI
</a>
</em>
</td>
<td>
<p>Controls deployment of NVMesh CSI Driver</p>
</td>
</tr>
<tr>
<td>
<code>operator</code><br/>
<em>
<a href="#nvmesh-excelero-com-v1-nvmeshoperatorspec">
NVMeshOperatorSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Control the behavior of the NVMesh operator for this NVMesh Cluster</p>
</td>
</tr>
<tr>
<td>
<code>debug</code><br/>
<em>
<a href="#nvmesh-excelero-com-v1-debugoptions">
DebugOptions
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Debug - debug options</p>
</td>
</tr>
<tr>
<td>
<code>actions</code><br/>
<em>
<a href="#nvmesh-excelero-com-v1-clusteraction">
[]ClusterAction
</a>
</em>
</td>
<td>
<p>Initiate actions such as collecting logs</p>
</td>
</tr>
</tbody>
</table>
<h3 id="nvmesh-excelero-com-v1-nvmeshstatus">NVMeshStatus
</h3>
<div class="alert alert-info col-md-8"><i class="fa fa-info-circle"></i> Appears In:
<ul>
<li><a href="#nvmesh-excelero-com-v1-nvmesh">NVMesh</a></li>
</ul>
</div>
<div>
<p>NVMeshStatus defines the observed state of NVMesh</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>WebUIURL</code><br/>
<em>
<span class="type">string<span>
</em>
</td>
<td>
<p>The URL of NVMesh Web GUI</p>
</td>
</tr>
<tr>
<td>
<code>reconcileStatus</code><br/>
<em>
<a href="#nvmesh-excelero-com-v1-reconcilestatus">
ReconcileStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>actionsStatus</code><br/>
<em>
<a href="#nvmesh-excelero-com-v1-actionstatus">
map[string]../../api/v1.ActionStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="nvmesh-excelero-com-v1-operatorfileserverspec">OperatorFileServerSpec
</h3>
<div class="alert alert-info col-md-8"><i class="fa fa-info-circle"></i> Appears In:
<ul>
<li><a href="#nvmesh-excelero-com-v1-nvmeshoperatorspec">NVMeshOperatorSpec</a></li>
</ul>
</div>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>address</code><br/>
<em>
<span class="type">string<span>
</em>
</td>
<td>
<p>The url address of the binaries file server</p>
</td>
</tr>
<tr>
<td>
<code>skipCheckCertificate</code><br/>
<em>
<span class="type">bool<span>
</em>
</td>
<td>
<p>Allows to connect to a self signed https server</p>
</td>
</tr>
</tbody>
</table>
<h3 id="nvmesh-excelero-com-v1-reconcilestatus">ReconcileStatus
</h3>
<div class="alert alert-info col-md-8"><i class="fa fa-info-circle"></i> Appears In:
<ul>
<li><a href="#nvmesh-excelero-com-v1-nvmeshstatus">NVMeshStatus</a></li>
</ul>
</div>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>lastUpdate</code><br/>
<em>
<a href="https://v1-18.docs.kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>reason</code><br/>
<em>
<span class="type">string<span>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<span class="type">string<span>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<hr/>
Generated using <a href="https://github.com/company/project"><code>crd-docs-generator</code></a>
.
