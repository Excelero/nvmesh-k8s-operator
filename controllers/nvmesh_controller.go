package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
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
	//GloballyNamedKinds - a list of Globally named kinds that are used in this operator
	GloballyNamedKinds = []string{
		"CSIDriver",
		"ClusterRole",
		"ClusterRoleBinding",
		"StorageClass",
		"CustomResourceDefinition",
		"SecurityContextConstraints",
	}

	reconcileCycles = 0
)

const (
	defaultRegistry           string = "registry.excelero.com"
	clusterServiceAccountName        = "nvmesh-cluster"
)

// NVMeshBaseReconciler - a base for NVMesh Component reconcilers
type NVMeshBaseReconciler struct {
	client.Client
	Log           logr.Logger
	Scheme        *runtime.Scheme
	DynamicClient dynamic.Interface
	Manager       ctrl.Manager
	EventManager  *EventManager
	Options       OperatorOptions
}

// NVMeshReconciler - Reconciles an NVMesh CR
type NVMeshReconciler struct {
	NVMeshBaseReconciler
}

// NVMeshComponent - defines the interface for NVMeshComponents (CSI, Core, Management)
type NVMeshComponent interface {
	InitObject(*nvmeshv1.NVMesh, *runtime.Object) error
	ShouldUpdateObject(cr *nvmeshv1.NVMesh, exp *runtime.Object, found *runtime.Object) bool
	Reconcile(*nvmeshv1.NVMesh, *NVMeshReconciler) error
}

// +kubebuilder:rbac:groups=nvmesh.excelero.com,resources=nvmeshes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nvmesh.excelero.com,resources=nvmeshes/status,verbs=get;update;patch
// +kubebuilder:subresource:status

// Reconcile - Reconciles an NVMesh CR
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
			return r.ManageSuccess(nil, DoNotRequeue())
		}
		// Error reading the object - requeue the request.
		return r.ManageError(cr, err)
	}

	// Make sure all fields are initialized
	r.initializeEmptyFieldsOnCustomResource(cr)

	//Validate CustomResource
	err = r.isValid(cr)
	if err != nil {
		return r.ManageError(cr, err)
	}

	r.stopAllUnstructuredWatchers()

	if r.Options.IsOpenShift {
		// Make sure SCC and ServiceAccount exists
		if err := r.makeSureSCCExists(cr); err != nil {
			return r.ManageError(cr, err)
		}
	}

	if err := r.makeSureServiceAccountExists(cr); err != nil {
		return r.ManageError(cr, err)
	}

	// Handle Cluster Deletion
	finResult, err := r.handleFinalizer(cr)
	if err != nil {
		return r.ManageError(cr, err)
	}

	if finResult.Requeue {
		return r.ManageSuccess(cr, finResult)
	}

	// Reconcile
	err = r.reconcileAllcomponents(cr)
	if err != nil {
		return r.ManageError(cr, err)
	}

	// Handle Stale Action Statuses
	result := r.removeFinishedActionStatuses(cr)
	if result.Requeue {
		return r.ManageSuccess(cr, result)
	}

	// Handle Actions
	if r.hasActions(cr) {
		actionResult, err := r.handleActions(cr)
		if err != nil {
			return r.ManageError(cr, err)
		}

		if actionResult.Requeue {
			return r.ManageSuccess(cr, actionResult)
		}
	}

	return r.ManageSuccess(cr, DoNotRequeue())
}

func (r *NVMeshReconciler) reconcileAllcomponents(cr *nvmeshv1.NVMesh) error {
	mgmt := NVMeshMgmtReconciler(*r)
	core := NVMeshCoreReconciler(*r)
	csi := NVMeshCSIReconciler(*r)

	components := []NVMeshComponent{&mgmt, &core, &csi}
	var errorList []error
	for _, component := range components {
		err := component.Reconcile(cr, r)
		if err != nil {
			errorList = append(errorList, err)
		}
	}

	if len(errorList) > 0 {
		for _, e := range errorList {
			r.Log.Error(e, "Error from ReconcileComponent")
		}
		return errorList[0]
	}

	return nil
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

//SetupWithManager - adds this reconciler to a manager
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

func (r *NVMeshReconciler) initializeEmptyFieldsOnCustomResource(cr *nvmeshv1.NVMesh) {
	if cr.Spec.Core.ImageRegistry == "" {
		cr.Spec.Core.ImageRegistry = defaultRegistry
	}

	if cr.Spec.Management.ImageRegistry == "" {
		cr.Spec.Management.ImageRegistry = defaultRegistry
	}

	if cr.Spec.Management.ImageRegistry == "" {
		cr.Spec.Management.ImageRegistry = defaultRegistry
	}

	if cr.Spec.Actions == nil {
		cr.Spec.Actions = make([]nvmeshv1.ClusterAction, 0)
	}

	if cr.Spec.Operator.FileServer == nil {
		cr.Spec.Operator.FileServer = &v1.OperatorFileServerSpec{}
	}
}
