package controllers

import (
	goerrors "errors"

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

func (r *NVMeshCSIReconciler) InitiateObject(cr *nvmeshv1.NVMesh, obj *runtime.Object) error {
	name, kind := GetRunetimeObjectNameAndKind(obj)
	if kind == "StatefulSet" {
		r.Log.Info("InitiateObject before switch ", "name", name, "kind", kind)
	}

	switch o := (*obj).(type) {
	case *appsv1.StatefulSet:
		r.Log.Info("InitiateObject case *appsv1.StatefulSet", "name", name, "kind", kind)

		switch name {
		case "nvmesh-csi-controller":
			r.Log.Info("InitiateObject case nvmesh-csi-controller", "name", name, "kind", kind)
			err := initiateCSIControllerStatefulSet(cr, o)
			r.Log.Info("InitiateObject case *appsv1.StatefulSet", "name", name, "kind", kind, "version", o.Spec.Template.Spec.Containers[0].Image)
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

func (r *NVMeshCSIReconciler) ShouldUpdateObject(cr *nvmeshv1.NVMesh, obj *runtime.Object) bool {
	name, kind := GetRunetimeObjectNameAndKind(obj)

	switch o := (*obj).(type) {
	case *appsv1.StatefulSet:
		r.Log.Info("ShouldUpdateObject case *appsv1.StatefulSet", "name", name, "kind", kind)

		switch name {
		case "nvmesh-csi-controller":
			r.Log.Info("ShouldUpdateObject case nvmesh-csi-controller", "name", name, "kind", kind)
			return shouldUpdateCSIControllerStatefulSet(cr, o)
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
		return true
	}

	return false
}

func shouldUpdateCSIControllerStatefulSet(cr *nvmeshv1.NVMesh, ss *appsv1.StatefulSet) bool {
	if cr.Spec.CSI.ControllerReplicas != *ss.Spec.Replicas {
		return true
	}

	if getCSIImageFromVersion(cr.Spec.CSI.Version) != ss.Spec.Template.Spec.Containers[0].Image {
		return true
	}

	return false
}
