package controllers

import (
	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NVMeshCoreAssestLocation = "config/samples/nvmesh-core"
)

type NVMeshCoreReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *NVMeshCoreReconciler) Reconcile(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	var err error

	if cr.Spec.Core.Deploy {
		err = nvmeshr.CreateObjectsFromDir(cr, r, NVMeshCoreAssestLocation)
	} else {
		err = nvmeshr.RemoveObjectsFromDir(cr, r, NVMeshCoreAssestLocation)
	}

	return err
}

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
