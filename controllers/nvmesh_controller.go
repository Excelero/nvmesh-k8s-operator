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

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
)

var GloballyNamedKinds = []string{
	"CSIDriver",
	"ClusterRole",
	"ClusterRoleBinding",
	"StorageClass",
	"CustomResourceDefinition",
}

// NVMeshReconciler reconciles a NVMesh object
type NVMeshReconciler struct {
	client.Client
	Log           logr.Logger
	Scheme        *runtime.Scheme
	DynamicClient dynamic.Interface
	Manager       ctrl.Manager
}

type NVMeshComponent interface {
	InitObject(*nvmeshv1.NVMesh, *runtime.Object) error
	ShouldUpdateObject(cr *nvmeshv1.NVMesh, exp *runtime.Object, found *runtime.Object) bool
	Reconcile(*nvmeshv1.NVMesh, *NVMeshReconciler) error
}

// +kubebuilder:rbac:groups=nvmesh.excelero.com,resources=nvmeshes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nvmesh.excelero.com,resources=nvmeshes/status,verbs=get;update;patch
// +kubebuilder:subresource:status

func (r *NVMeshReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("nvmesh", req.NamespacedName)

	// Fetch the NVMesh instance
	cr := &nvmeshv1.NVMesh{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, cr)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	mgmt := NVMeshMgmtReconciler(*r)
	core := NVMeshCoreReconciler(*r)
	csi := NVMeshCSIReconciler(*r)

	components := []NVMeshComponent{&mgmt, &core, &csi}
	var errorList []error
	for _, component := range components {
		err = component.Reconcile(cr, r)
		if err != nil {
			errorList = append(errorList, err)
		}
	}

	if len(errorList) > 0 {
		for _, e := range errorList {
			r.Log.Error(e, "Error from ReconcileComponent")
		}
		return reconcile.Result{}, errorList[0]
	}

	return ctrl.Result{}, nil
}

func (r *NVMeshReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nvmeshv1.NVMesh{}).
		Complete(r)
}

// Add MongoDB Community Operator Schema's Group and version
var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: "mongodb.com", Version: "v1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)
