package controllers

import (
	goerrors "errors"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	csiAssetsLocation  = "resources/csi/"
	csiDaemonSetName   = "nvmesh-csi-node-driver"
	csiStatefulSetName = "nvmesh-csi-controller"
	csiDriverImageName = "excelero/nvmesh-csi-driver"
)

//NVMeshCSIReconciler is a Reconciler for CSI
type NVMeshCSIReconciler struct {
	NVMeshBaseReconciler
}

//Reconcile Reconciles CSI
func (r *NVMeshCSIReconciler) Reconcile(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	if !cr.Spec.CSI.Disabled {
		return r.deployCSI(cr, nvmeshr)
	}

	return r.removeCSI(cr, nvmeshr)
}

func (r *NVMeshCSIReconciler) deployCSI(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	return nvmeshr.createObjectsFromDir(cr, r, csiAssetsLocation, true)
}

func (r *NVMeshCSIReconciler) removeCSI(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	return nvmeshr.removeObjectsFromDir(cr, r, csiAssetsLocation, true)
}

//InitObject Initializes CSI Objects
func (r *NVMeshCSIReconciler) InitObject(cr *nvmeshv1.NVMesh, obj *runtime.Object) error {
	name, _ := getRunetimeObjectNameAndKind(obj)

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

//ShouldUpdateObject Manages CIS object updates
func (r *NVMeshCSIReconciler) ShouldUpdateObject(cr *nvmeshv1.NVMesh, exp *runtime.Object, found *runtime.Object) bool {
	name, _ := getRunetimeObjectNameAndKind(found)

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
	conf.Data["management.protocol"] = mgmtProtocol
	conf.Data["management.servers"] = mgmtGuiServiceName + "." + cr.GetNamespace() + ".svc.cluster.local:4000"
	return nil
}

func getCSIFullImageName(cr *nvmeshv1.NVMesh) string {
	imageName := csiDriverImageName
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
