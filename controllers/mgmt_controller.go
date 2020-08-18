package controllers

import (
	goerrors "errors"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	MgmtAssetsLocation  = "config/samples/management/"
	MgmtStatefulSetName = "nvmesh-management"
	MgmtImageName       = "docker.excelero.com/nvmesh-management"
)

func (r *NVMeshMgmtReconciler) InitiateObject(cr *nvmeshv1.NVMesh, obj *runtime.Object) error {
	name, _ := GetRunetimeObjectNameAndKind(obj)
	switch o := (*obj).(type) {
	case *appsv1.StatefulSet:
		switch name {
		case "nvmesh-management":
			return initiateMgmtStatefulSet(cr, o)
		}
	default:
		//o is unknown for us
		//log.Info(fmt.Sprintf("Object type %s not handled", o))
	}

	return nil
}

func (r *NVMeshMgmtReconciler) ShouldUpdateObject(cr *nvmeshv1.NVMesh, obj *runtime.Object) bool {
	name, _ := GetRunetimeObjectNameAndKind(obj)
	switch o := (*obj).(type) {
	case *appsv1.StatefulSet:
		switch name {
		case "nvmesh-management":
			return shouldUpdateMgmtStatefulSet(cr, o)
		}
	default:
		//o is unknown for us
		//log.Info(fmt.Sprintf("Object type %s not handled", o))
	}

	return false
}

func initiateMgmtStatefulSet(cr *nvmeshv1.NVMesh, o *appsv1.StatefulSet) error {

	if cr.Spec.Management.Version == "" {
		return goerrors.New("Missing Management Version (NVMesh.Spec.Management.Version)")
	}

	o.Spec.Template.Spec.Containers[0].Image = getMgmtImageFromVersion(cr.Spec.Management.Version)

	//TODO: set still use configMap or set values directly into the daemonset ?
	return nil
}

func getMgmtImageFromVersion(version string) string {
	return MgmtImageName + ":" + version
}

func shouldUpdateMgmtStatefulSet(cr *nvmeshv1.NVMesh, ss *appsv1.StatefulSet) bool {

	if getMgmtImageFromVersion(cr.Spec.Management.Version) != ss.Spec.Template.Spec.Containers[0].Image {
		return true
	}

	return false
}
