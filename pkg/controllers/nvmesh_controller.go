package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/pkg/api/v1"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
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
	clusterServiceAccountName = "nvmesh-cluster"
	defaultRegistry           = "registry.excelero.com"
	VerboseLogging            = 5
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
	InitObject(*nvmeshv1.NVMesh, client.Object) error
	ShouldUpdateObject(cr *nvmeshv1.NVMesh, exp client.Object, found client.Object) bool
	Reconcile(*nvmeshv1.NVMesh, *NVMeshReconciler) (ctrl.Result, error)
}

// +kubebuilder:rbac:groups=nvmesh.excelero.com,resources=nvmeshes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nvmesh.excelero.com,resources=nvmeshes/status,verbs=get;update;patch
// +kubebuilder:subresource:status

// Reconcile - Reconciles an NVMesh CR
func (r *NVMeshReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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
		return r.ManageError(cr, err, RequeueWithDefaultBackOff())
	}

	// Make sure all fields are initialized
	r.initializeEmptyFieldsOnCustomResource(cr)

	//Validate CustomResource
	err = r.isValid(cr)
	if err != nil {
		return r.ManageError(cr, err, RequeueWithDefaultBackOff())
	}

	r.stopAllUnstructuredWatchers()

	if err := r.makeSureServiceAccountExists(cr); err != nil {
		return r.ManageError(cr, err, RequeueWithDefaultBackOff())
	}

	// Handle Cluster Deletion
	finResult, err := r.handleFinalizer(cr)
	if err != nil {
		return r.ManageError(cr, err, RequeueWithDefaultBackOff())
	}

	if finResult.Requeue {
		return r.ManageSuccess(cr, finResult)
	}

	// Reconcile
	result, err := r.reconcileAllcomponents(cr)

	if result.Requeue {
		if err != nil {
			_, err = r.ManageError(cr, err, result)
		}

		// drop the result returned by ManageError
		return result, err
	}

	// Handle Stale Action Statuses
	result = r.removeFinishedActionStatuses(cr)
	if result.Requeue {
		return r.ManageSuccess(cr, result)
	}

	// Handle Actions
	if r.hasActions(cr) {
		actionResult, err := r.handleActions(cr)
		if err != nil {
			return r.ManageError(cr, err, actionResult)
		}

		if actionResult.Requeue {
			return r.ManageSuccess(cr, actionResult)
		}
	}

	return r.ManageSuccess(cr, DoNotRequeue())
}

func (r *NVMeshReconciler) reconcileAllcomponents(cr *nvmeshv1.NVMesh) (ctrl.Result, error) {
	mgmt := NVMeshMgmtReconciler(*r)
	core := NVMeshCoreReconciler(*r)
	csi := NVMeshCSIReconciler(*r)
	components := []NVMeshComponent{&mgmt, &core, &csi}
	var errorList []error
	var errToReturn error
	resultWithMinimalRequeue := DoNotRequeue()
	for _, component := range components {
		result, err := component.Reconcile(cr, r)

		// We collect errors and keep on Reconciling other components
		// We then requeue another reconcile cycle with the shortest reconcile requested
		if err != nil {
			errorList = append(errorList, err)
		}

		if result.Requeue {
			resultWithMinimalRequeue.Requeue = true
			if result.RequeueAfter < resultWithMinimalRequeue.RequeueAfter {
				resultWithMinimalRequeue.RequeueAfter = result.RequeueAfter
			}
		}
	}

	if len(errorList) > 0 {
		for _, e := range errorList {
			r.Log.Error(e, "Error from ReconcileComponent")
		}

		errToReturn = errorList[0]
	}

	return resultWithMinimalRequeue, errToReturn
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

//SetupWithManager - adds this reconciler to a manager
func (r *NVMeshReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// Reconcile only if generation field changed - this is to prevent cycle loop after status updates
	generationChangePredicate := predicate.GenerationChangedPredicate{}

	controllerBuilder := ctrl.NewControllerManagedBy(mgr).
		For(&nvmeshv1.NVMesh{}).
		WithEventFilter(generationChangePredicate).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Service{}).
		Owns(&rbac.ClusterRole{}).
		Owns(&rbac.ClusterRoleBinding{}).
		Owns(&rbac.Role{}).
		Owns(&rbac.RoleBinding{}).
		Owns(&storagev1.CSIDriver{}).
		Owns(&storagev1.StorageClass{})
	return controllerBuilder.Complete(r)
}

func (r *NVMeshReconciler) initializeEmptyFieldsOnCustomResource(cr *nvmeshv1.NVMesh) {
	if cr.Spec.Core.ImageRegistry == "" {
		cr.Spec.Core.ImageRegistry = defaultRegistry
	}

	if cr.Spec.Core.ImageVersionTag == "" {
		cr.Spec.Core.ImageVersionTag = r.Options.DefaultCoreImageTag
	}

	if cr.Spec.Management.ImageRegistry == "" {
		cr.Spec.Management.ImageRegistry = defaultRegistry
	}

	if cr.Spec.CSI.ControllerReplicas == 0 {
		cr.Spec.CSI.ControllerReplicas = 1
	}

	if cr.Spec.Management.Replicas == 0 {
		cr.Spec.Management.Replicas = 1
	}

	if cr.Spec.Actions == nil {
		cr.Spec.Actions = make([]nvmeshv1.ClusterAction, 0)
	}

	// if cr.Spec.Operator.FileServer == nil {
	// 	cr.Spec.Operator.FileServer = &v1.OperatorFileServerSpec{}
	// }
}

func (r *NVMeshBaseReconciler) addKeepRunningAfterFailureEnvVar(cr *nvmeshv1.NVMesh, container *corev1.Container) {
	if !cr.Spec.Debug.ContainersKeepRunningAfterFailure {
		return
	}

	env := corev1.EnvVar{
		Name:  "KEEP_RUNNING_WHEN_FINISHED",
		Value: "true",
	}

	container.Env = append(container.Env, env)
}

func (r *NVMeshBaseReconciler) getCoreFullImageName(cr *nvmeshv1.NVMesh, imageName string) string {
	return cr.Spec.Core.ImageRegistry + "/" + imageName + ":" + cr.Spec.Core.ImageVersionTag
}
