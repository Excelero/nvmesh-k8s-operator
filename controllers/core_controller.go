package controllers

import (
	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (r *NVMeshCoreReconciler) InitObject(cr *nvmeshv1.NVMesh, obj *runtime.Object) error {
	//name, _ := GetRunetimeObjectNameAndKind(obj)
	// switch o := (*obj).(type) {
	// case *appsv1.DaemonSet:
	// default:
	// 	//o is unknown for us
	// 	//log.Info(fmt.Sprintf("Object type %s not handled", o))
	// }

	return nil
}

func (r *NVMeshCoreReconciler) ShouldUpdateObject(cr *nvmeshv1.NVMesh, exp *runtime.Object, obj *runtime.Object) bool {
	//name, _ := GetRunetimeObjectNameAndKind(obj)
	// switch o := (*obj).(type) {
	// case *appsv1.DaemonSet:
	// default:
	// 	//o is unknown for us
	// 	//log.Info(fmt.Sprintf("Object type %s not handled", o))
	// }

	return false
}
