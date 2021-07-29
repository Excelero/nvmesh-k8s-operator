// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in ActionStatus) DeepCopyInto(out *ActionStatus) {
	{
		in := &in
		*out = make(ActionStatus, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ActionStatus.
func (in ActionStatus) DeepCopy() ActionStatus {
	if in == nil {
		return nil
	}
	out := new(ActionStatus)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterAction) DeepCopyInto(out *ClusterAction) {
	*out = *in
	if in.Args != nil {
		in, out := &in.Args, &out.Args
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterAction.
func (in *ClusterAction) DeepCopy() *ClusterAction {
	if in == nil {
		return nil
	}
	out := new(ClusterAction)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DebugOptions) DeepCopyInto(out *DebugOptions) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DebugOptions.
func (in *DebugOptions) DeepCopy() *DebugOptions {
	if in == nil {
		return nil
	}
	out := new(DebugOptions)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MongoDBCluster) DeepCopyInto(out *MongoDBCluster) {
	*out = *in
	in.DataVolumeClaim.DeepCopyInto(&out.DataVolumeClaim)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MongoDBCluster.
func (in *MongoDBCluster) DeepCopy() *MongoDBCluster {
	if in == nil {
		return nil
	}
	out := new(MongoDBCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NVMesh) DeepCopyInto(out *NVMesh) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NVMesh.
func (in *NVMesh) DeepCopy() *NVMesh {
	if in == nil {
		return nil
	}
	out := new(NVMesh)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NVMesh) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NVMeshCSI) DeepCopyInto(out *NVMeshCSI) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NVMeshCSI.
func (in *NVMeshCSI) DeepCopy() *NVMeshCSI {
	if in == nil {
		return nil
	}
	out := new(NVMeshCSI)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NVMeshCore) DeepCopyInto(out *NVMeshCore) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NVMeshCore.
func (in *NVMeshCore) DeepCopy() *NVMeshCore {
	if in == nil {
		return nil
	}
	out := new(NVMeshCore)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NVMeshList) DeepCopyInto(out *NVMeshList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]NVMesh, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NVMeshList.
func (in *NVMeshList) DeepCopy() *NVMeshList {
	if in == nil {
		return nil
	}
	out := new(NVMeshList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NVMeshList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NVMeshManagement) DeepCopyInto(out *NVMeshManagement) {
	*out = *in
	in.MongoDB.DeepCopyInto(&out.MongoDB)
	if in.ExternalIPs != nil {
		in, out := &in.ExternalIPs, &out.ExternalIPs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.BackupsVolumeClaim.DeepCopyInto(&out.BackupsVolumeClaim)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NVMeshManagement.
func (in *NVMeshManagement) DeepCopy() *NVMeshManagement {
	if in == nil {
		return nil
	}
	out := new(NVMeshManagement)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NVMeshOperatorSpec) DeepCopyInto(out *NVMeshOperatorSpec) {
	*out = *in
	if in.FileServer != nil {
		in, out := &in.FileServer, &out.FileServer
		*out = new(OperatorFileServerSpec)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NVMeshOperatorSpec.
func (in *NVMeshOperatorSpec) DeepCopy() *NVMeshOperatorSpec {
	if in == nil {
		return nil
	}
	out := new(NVMeshOperatorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NVMeshSpec) DeepCopyInto(out *NVMeshSpec) {
	*out = *in
	out.Core = in.Core
	in.Management.DeepCopyInto(&out.Management)
	out.CSI = in.CSI
	in.Operator.DeepCopyInto(&out.Operator)
	out.Debug = in.Debug
	if in.Actions != nil {
		in, out := &in.Actions, &out.Actions
		*out = make([]ClusterAction, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NVMeshSpec.
func (in *NVMeshSpec) DeepCopy() *NVMeshSpec {
	if in == nil {
		return nil
	}
	out := new(NVMeshSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NVMeshStatus) DeepCopyInto(out *NVMeshStatus) {
	*out = *in
	in.ReconcileStatus.DeepCopyInto(&out.ReconcileStatus)
	if in.ActionsStatus != nil {
		in, out := &in.ActionsStatus, &out.ActionsStatus
		*out = make(map[string]ActionStatus, len(*in))
		for key, val := range *in {
			var outVal map[string]string
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = make(ActionStatus, len(*in))
				for key, val := range *in {
					(*out)[key] = val
				}
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NVMeshStatus.
func (in *NVMeshStatus) DeepCopy() *NVMeshStatus {
	if in == nil {
		return nil
	}
	out := new(NVMeshStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OperatorFileServerSpec) DeepCopyInto(out *OperatorFileServerSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OperatorFileServerSpec.
func (in *OperatorFileServerSpec) DeepCopy() *OperatorFileServerSpec {
	if in == nil {
		return nil
	}
	out := new(OperatorFileServerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReconcileStatus) DeepCopyInto(out *ReconcileStatus) {
	*out = *in
	in.LastUpdate.DeepCopyInto(&out.LastUpdate)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReconcileStatus.
func (in *ReconcileStatus) DeepCopy() *ReconcileStatus {
	if in == nil {
		return nil
	}
	out := new(ReconcileStatus)
	in.DeepCopyInto(out)
	return out
}
