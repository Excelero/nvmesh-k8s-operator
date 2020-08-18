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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nvmeshv1alpha1 "excelero.com/nvmesh-k8s-operator/api/v1alpha1"
)

// NVMeshReconciler reconciles a NVMesh object
type NVMeshReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

type NVMeshComponent interface {
	ShouldUpdateObject(*nvmeshv1alpha1.NVMesh, *runtime.Object) bool
	InitiateObject(*nvmeshv1alpha1.NVMesh, *runtime.Object) error
}

type NVMeshCSIReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

type NVMeshMgmtReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

type NVMeshCoreReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=nvmesh.excelero.com,resources=nvmeshes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nvmesh.excelero.com,resources=nvmeshes/status,verbs=get;update;patch
// +kubebuilder:subresource:status

func (r *NVMeshReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("nvmesh", req.NamespacedName)

	// Fetch the NVMesh instance
	instance := &nvmeshv1alpha1.NVMesh{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
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

	csi := NVMeshCSIReconciler(*r)
	mgmt := NVMeshMgmtReconciler(*r)
	core := NVMeshCoreReconciler(*r)
	components := []NVMeshComponent{&csi, &mgmt, &core}
	var errorList []error
	for _, component := range components {
		err = r.ReconcileComponent(instance, component)
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
		For(&nvmeshv1alpha1.NVMesh{}).
		Complete(r)
}

func (r *NVMeshReconciler) ReconcileComponent(cr *nvmeshv1alpha1.NVMesh, comp NVMeshComponent) error {
	files, err := ListFilesInSubDirs(CSIAssetsLocation)
	if err != nil {
		return err
	}

	for _, file := range files {
		err = r.ReconcileYamlObjectsFromFile(cr, file, comp)
		if err != nil {
			return err
		}
	}

	return nil
}
