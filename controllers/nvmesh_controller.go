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
	"fmt"

	"github.com/go-logr/logr"
	"github.com/prometheus/common/log"
	appsv1 "k8s.io/api/apps/v1"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
	v1 "excelero.com/nvmesh-k8s-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
)

var (
	GloballyNamedKinds = []string{
		"CSIDriver",
		"ClusterRole",
		"ClusterRoleBinding",
		"StorageClass",
		"CustomResourceDefinition",
	}

	reconcileCycles = 0
)

const (
	defaultRegistry string = "registry.excelero.com"
)

// NVMeshReconciler reconciles a NVMesh object
type NVMeshBaseReconciler struct {
	client.Client
	Log           logr.Logger
	Scheme        *runtime.Scheme
	DynamicClient dynamic.Interface
	Manager       ctrl.Manager
	EventManager  *EventManager
}
type NVMeshReconciler struct {
	NVMeshBaseReconciler
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

	reconcileCycles = reconcileCycles + 1

	// Fetch the NVMesh instance
	cr := &nvmeshv1.NVMesh{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, cr)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return r.ManageSuccess(nil, false)
		}
		// Error reading the object - requeue the request.
		return r.ManageError(cr, err)
	}

	// Make sure Initialized
	ok, err := r.MakeSureInitialized(cr)
	if err != nil {
		r.ManageError(cr, err)
	} else if !ok {
		// object was initialized and updated - brun another reconcile cycle
		r.ManageSuccess(cr, true)
	}

	//Validate CustomResource
	err = r.IsValid(cr)
	if err != nil {
		r.ManageError(cr, err)
	}

	r.stopAllUnstructuredWatchers()
	err = r.HandleFinalizer(cr)
	if err != nil {
		return r.ManageError(cr, err)
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
		return r.ManageError(cr, errorList[0])
	}

	return r.ManageSuccess(cr, false)
}

func (r *NVMeshReconciler) getManagementGUIURL(cr *nvmeshv1.NVMesh) string {
	// Get Management GUI External URL

	if cr.Spec.Management.ExternalIPs == nil || len(cr.Spec.Management.ExternalIPs) == 0 {
		return ""
	}

	var protocol string

	if cr.Spec.Management.NoSSL {
		protocol = "http"
	} else {
		protocol = "https"
	}

	address := cr.Spec.Management.ExternalIPs[0]
	port := 4000

	url := fmt.Sprintf("%s://%s:%d", protocol, address, port)
	return url
}

func (r *NVMeshReconciler) setStatusOnCustomResource(cr *nvmeshv1.NVMesh) {
	cr.Status.WebUIURL = r.getManagementGUIURL(cr)
}

func (r *NVMeshReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// this handler will initiate Reconcile cycle whenever an object is Created, Deleted or Updated
	handler := &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &v1.NVMesh{},
	}

	// Reconcile only if generation field changed - this is to prevent cycle loop after status updates
	generationChangePredicate := predicate.GenerationChangedPredicate{}

	return ctrl.NewControllerManagedBy(mgr).
		For(&nvmeshv1.NVMesh{}).
		WithEventFilter(generationChangePredicate).
		Watches(&source.Kind{Type: &appsv1.StatefulSet{}}, handler).
		Watches(&source.Kind{Type: &appsv1.Deployment{}}, handler).
		Watches(&source.Kind{Type: &appsv1.DaemonSet{}}, handler).
		Watches(&source.Kind{Type: &corev1.Service{}}, handler).
		Watches(&source.Kind{Type: &corev1.ServiceAccount{}}, handler).
		Watches(&source.Kind{Type: &corev1.ConfigMap{}}, handler).
		Watches(&source.Kind{Type: &corev1.Secret{}}, handler).
		Watches(&source.Kind{Type: &corev1.Service{}}, handler).
		Watches(&source.Kind{Type: &rbac.ClusterRole{}}, handler).
		Watches(&source.Kind{Type: &rbac.ClusterRoleBinding{}}, handler).
		Watches(&source.Kind{Type: &rbac.Role{}}, handler).
		Watches(&source.Kind{Type: &rbac.RoleBinding{}}, handler).
		Watches(&source.Kind{Type: &storagev1beta1.CSIDriver{}}, handler).
		Watches(&source.Kind{Type: &storagev1.StorageClass{}}, handler).
		Watches(&source.Kind{Type: &apiext.CustomResourceDefinition{}}, handler).
		Complete(r)
}

func (r *NVMeshReconciler) IsInitialized(cr *nvmeshv1.NVMesh) bool {
	var ok bool = true // was the object already initialized
	if cr.Spec.Core.ImageRegistry == "" {
		ok = false
		cr.Spec.Core.ImageRegistry = defaultRegistry
	}

	if cr.Spec.Management.ImageRegistry == "" {
		ok = false
		cr.Spec.Management.ImageRegistry = defaultRegistry
	}

	return ok
}

func (r *NVMeshReconciler) MakeSureInitialized(cr *nvmeshv1.NVMesh) (bool, error) {
	ok := r.IsInitialized(cr)
	if !ok {
		// object was not initialized - we'll update it and return ok = false so that we run another reconcile cycle
		err := r.Client.Update(context.TODO(), cr)
		if err != nil {
			log.Error(err, "unable to update instance", "instance", cr)
			return ok, err
		}
	}

	return ok, nil
}
