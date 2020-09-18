package controllers

import (
	goerrors "errors"
	"fmt"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	CSIAssetsLocation  = "resources/csi/"
	CSIDaemonSetName   = "nvmesh-csi-node-driver"
	CSIStatefulSetName = "nvmesh-csi-controller"
	CSIDriverImageName = "excelero/nvmesh-csi-driver"
)

type NVMeshCSIReconciler struct {
	client.Client
	Log           logr.Logger
	Scheme        *runtime.Scheme
	DynamicClient dynamic.Interface
	Manager       ctrl.Manager
	EventManager  *EventManager
}

func (r *NVMeshCSIReconciler) Reconcile(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	var err error
	recursive := true

	if cr.Spec.CSI.Deploy {
		err = nvmeshr.CreateObjectsFromDir(cr, r, CSIAssetsLocation, recursive)
	} else {
		err = nvmeshr.RemoveObjectsFromDir(cr, r, CSIAssetsLocation, recursive)
	}

	return err
}

func (r *NVMeshCSIReconciler) InitObject(cr *nvmeshv1.NVMesh, obj *runtime.Object) error {
	name, _ := GetRunetimeObjectNameAndKind(obj)

	switch o := (*obj).(type) {
	case *appsv1.StatefulSet:
		switch name {
		case "nvmesh-csi-controller":
			err := initCSIControllerStatefulSet(cr, o)
			return err
		}
	case *appsv1.DaemonSet:
		switch name {
		case "nvmesh-csi-node-driver":
			return initCSINodeDriverDaemonSet(cr, o)
		}
	case *appsv1.Deployment:
	case *v1.ServiceAccount:
	case *v1.ConfigMap:
		return initCSIConfigMap(cr, o)
	case *rbac.RoleBinding:
		addNamespaceToRoleBinding(cr, o)
		return nil
	case *rbac.ClusterRoleBinding:
		addNamespaceToClusterRoleBinding(cr, o)
		return nil
	default:
		//o is unknown for us
		//log.Info(fmt.Sprintf("Object type %s not handled", o))
	}

	return nil
}

func (r *NVMeshCSIReconciler) ShouldUpdateObject(cr *nvmeshv1.NVMesh, exp *runtime.Object, found *runtime.Object) bool {
	name, _ := GetRunetimeObjectNameAndKind(found)

	switch o := (*found).(type) {
	case *appsv1.StatefulSet:
		switch name {
		case "nvmesh-csi-controller":
			expected := (*exp).(*appsv1.StatefulSet)
			return r.shouldUpdateCSIControllerStatefulSet(cr, expected, o)
		}
	case *appsv1.DaemonSet:
		switch name {
		case "nvmesh-csi-node-driver":
			return r.shouldUpdateCSINodeDriverDaemonSet(cr, o)
		}

	case *appsv1.Deployment:
	case *v1.ServiceAccount:
	case *v1.ConfigMap:
	default:
		//o is unknown for us
		//log.Info(fmt.Sprintf("Object type %s not handled", o))
	}

	return false
}

func initCSINodeDriverDaemonSet(cr *nvmeshv1.NVMesh, ds *appsv1.DaemonSet) error {
	if cr.Spec.CSI.Version == "" {
		return goerrors.New("Missing NVMesh CSI Driver Version (NVMesh.Spec.CSI.Version)")
	}

	ds.Spec.Template.Spec.Containers[0].Image = getCSIFullImageName(cr)

	return nil
}

func initCSIControllerStatefulSet(cr *nvmeshv1.NVMesh, ss *appsv1.StatefulSet) error {
	if cr.Spec.CSI.Version == "" {
		return goerrors.New("Missing NVMesh CSI Driver Version (NVMesh.Spec.CSI.Version)")
	}

	ss.Spec.Template.Spec.Containers[0].Image = getCSIFullImageName(cr)

	// set replicas from CustomResource
	ss.Spec.Replicas = &cr.Spec.CSI.ControllerReplicas

	return nil
}

func initCSIConfigMap(cr *nvmeshv1.NVMesh, conf *v1.ConfigMap) error {
	conf.Data["management.protocol"] = MgmtProtocol
	conf.Data["management.servers"] = MgmtGuiServiceName + "." + cr.GetNamespace() + ".svc.cluster.local:4000"
	return nil
}

func getCSIFullImageName(cr *nvmeshv1.NVMesh) string {
	imageName := CSIDriverImageName
	if cr.Spec.CSI.ImageName != "" {
		imageName = cr.Spec.CSI.ImageName
	}

	version := cr.Spec.CSI.Version

	return imageName + ":" + version
}

func (r *NVMeshCSIReconciler) shouldUpdateCSINodeDriverDaemonSet(cr *nvmeshv1.NVMesh, ds *appsv1.DaemonSet) bool {
	log := r.Log.WithValues("method", "shouldUpdateCSINodeDriverDaemonSet")
	if getCSIFullImageName(cr) != ds.Spec.Template.Spec.Containers[0].Image {
		log.Info(fmt.Sprintf("CSI Node Driver Image needs to be updated expected: %s found: %s\n", getCSIFullImageName(cr), ds.Spec.Template.Spec.Containers[0].Image))
		return true
	}

	return false
}

func (r *NVMeshCSIReconciler) shouldUpdateCSIControllerStatefulSet(cr *nvmeshv1.NVMesh, expected *appsv1.StatefulSet, ss *appsv1.StatefulSet) bool {
	log := r.Log.WithValues("method", "shouldUpdateCSIControllerStatefulSet")

	if *(expected.Spec.Replicas) != *(ss.Spec.Replicas) {
		log.Info(fmt.Sprintf("CSI controller replica number needs to be updated expected: %d found: %d\n", *expected.Spec.Replicas, *ss.Spec.Replicas))
		return true
	}

	if expected.Spec.Template.Spec.Containers[0].Image != ss.Spec.Template.Spec.Containers[0].Image {
		log.Info(fmt.Sprintf("CSI controller Image needs to be updated expected: %s found: %s\n", getCSIFullImageName(cr), ss.Spec.Template.Spec.Containers[0].Image))
		return true
	}

	return false
}
