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
func (in *NVMesh) DeepCopyInto(out *NVMesh) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
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
func (in *NVMeshSpec) DeepCopyInto(out *NVMeshSpec) {
	*out = *in
	out.Core = in.Core
	out.Management = in.Management
	out.CSI = in.CSI
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