# Excelero NVMesh Kubernetes Operator

Excelero NVMesh is a low-latency distributed block storage solution providing web-scale applications access to hot data in any cloud, private or public. NVMesh enables pooling and sharing NVMe across any network. It drives both local and distributed file systems. The solution features an intelligent management layer that abstracts underlying hardware with CPU offload, creates logical volumes with redundancy and provides centralized, intelligent management and monitoring. Applications can enjoy the latency, throughput and IO/s of local NVMe devices with the convenience of centralized storage while avoiding proprietary hardware lock-in and reducing the overall storage TCO. In public cloud environments, NVMesh supports instances, both virtualized and containerized, that feature NVMe drives. Public cloud instances with local NVMe drives have become widely available allowing easy transition between on-premises and public cloud deployments of NVMesh.

Excelero NVMesh has a flexible distributed data protection architecture providing multiple redundancy schemes that can be tuned for specific use cases and data center restrictions and requirements to ensure reliability and to reduce cost. The system can also work around failure to strive for maximal data availability. In NVMesh, drives are perceived as resources that are pooled into a large storage area. Logical volumes are then carved out of the storage area and presented to clients as block devices. Volumes may span multiple physical drives and target hosts, but do not need to use entire drives, so a single drive can be allocated to multiple volumes.

Volumes can be configured in any of the following redundancy levels:

**Concatenated** – Data is laid out on a single or multiple drives with no data redundancy. This volume type can be used for applications requiring temporary storage. Failures are typically isolated to a single device or host.

**Striped** – Data is laid out across a set of drives and hosts with no data redundancy. This volume type can be used for applications requiring high performance temporary storage.

**Mirrored** – Data is protected by mirroring data across drive segments. To increase data availability, the drive segments are allocated from drives on different target hosts. These hosts should be connected to different power supplies, preferably in separate upgrade zones and availability zones and any other zoning used to protect against risk. The software’s management layer provides the agility to ensure such separation. Multi-way, active-active multipath networking is used for availability and for performance.

**Striped** and Mirrored – Data is protected by mirroring data across drive segments and striping across these mirrors. Data is serviced from many drives and hosts, achieving high performance without sacrificing redundancy.

For details on licensing, support and usage under Red Hat OpenShift please access [https://www.excelero.com/openshift](https://www.excelero.com/openshift)
