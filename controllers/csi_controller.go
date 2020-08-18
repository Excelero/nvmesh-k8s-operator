package controllers

import (
	goerrors "errors"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	CSIAssetsLocation  = "config/samples/csi/"
	CSIDaemonSetName   = "nvmesh-csi-node-driver"
	CSIStatefulSetName = "nvmesh-csi-controller"
	CSIDriverImageName = "excelero/nvmesh-csi-driver"
)

var GloballyNamedKinds = []string{
	"CSIDriver",
	"ClusterRole",
	"ClusterRoleBinding",
	"StorageClass",
}

func (r *NVMeshCSIReconciler) InitObject(cr *nvmeshv1.NVMesh, obj *runtime.Object) error {
	name, _ := GetRunetimeObjectNameAndKind(obj)

	switch o := (*obj).(type) {
	case *appsv1.StatefulSet:
		switch name {
		case "nvmesh-csi-controller":
			err := initiateCSIControllerStatefulSet(cr, (*obj).(*appsv1.StatefulSet))
			return err
		}
	case *appsv1.DaemonSet:
		switch name {
		case "nvmesh-csi-node-driver":
			return initiateCSINodeDriverDaemonSet(cr, o)
		}
	case *appsv1.Deployment:
	case *v1.ServiceAccount:
	case *v1.ConfigMap:
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
			return shouldUpdateCSIControllerStatefulSet(cr, expected, o)
		}
	case *appsv1.DaemonSet:
		switch name {
		case "nvmesh-csi-node-driver":
			return shouldUpdateCSINodeDriverDaemonSet(cr, o)
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

func initiateCSINodeDriverDaemonSet(cr *nvmeshv1.NVMesh, ds *appsv1.DaemonSet) error {
	if cr.Spec.CSI.Version == "" {
		return goerrors.New("Missing NVMesh CSI Driver Version (NVMesh.Spec.CSI.Version)")
	}

	ds.Spec.Template.Spec.Containers[0].Image = getCSIImageFromVersion(cr.Spec.CSI.Version)

	//TODO: set still use configMap or set values directly into the daemonset ?
	return nil
}

func initiateCSIControllerStatefulSet(cr *nvmeshv1.NVMesh, ss *appsv1.StatefulSet) error {
	if cr.Spec.CSI.Version == "" {
		return goerrors.New("Missing NVMesh CSI Driver Version (NVMesh.Spec.CSI.Version)")
	}

	ss.Spec.Template.Spec.Containers[0].Image = getCSIImageFromVersion(cr.Spec.CSI.Version)

	// set replicas from CustomResource
	ss.Spec.Replicas = &cr.Spec.CSI.ControllerReplicas

	return nil
}

func getCSIImageFromVersion(version string) string {
	return CSIDriverImageName + ":" + version
}

func shouldUpdateCSINodeDriverDaemonSet(cr *nvmeshv1.NVMesh, ds *appsv1.DaemonSet) bool {
	if getCSIImageFromVersion(cr.Spec.CSI.Version) != ds.Spec.Template.Spec.Containers[0].Image {
		fmt.Printf("CSI Node Driver Image needs to be updated expected: %s found: %s\n", getCSIImageFromVersion(cr.Spec.CSI.Version), ds.Spec.Template.Spec.Containers[0].Image)
		return true
	}

	return false
}

func shouldUpdateCSIControllerStatefulSet(cr *nvmeshv1.NVMesh, expected *appsv1.StatefulSet, ss *appsv1.StatefulSet) bool {
	if *(expected.Spec.Replicas) != *(ss.Spec.Replicas) {
		fmt.Printf("CSI controller replica number needs to be updated expected: %d found: %d\n", *expected.Spec.Replicas, *ss.Spec.Replicas)
		return true
	}

	if expected.Spec.Template.Spec.Containers[0].Image != ss.Spec.Template.Spec.Containers[0].Image {
		fmt.Printf("CSI controller Image needs to be updated expected: %s found: %s\n", getCSIImageFromVersion(cr.Spec.CSI.Version), ss.Spec.Template.Spec.Containers[0].Image)
		return true
	}

	return false
}
