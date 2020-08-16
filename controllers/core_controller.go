package controllers

import (
	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NVMeshCoreReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

type CoreManagedObject interface {
	shouldUpdate(*nvmeshv1.NVMesh, *runtime.Object) bool
	newObject(*nvmeshv1.NVMesh) (*runtime.Object, error)
	getObject(*nvmeshv1.NVMesh, *NVMeshCoreReconciler) (*runtime.Object, error)
}

type nvmeshClientDriverDaemonSet struct{}
type nvmeshTargetDriverDaemonSet struct{}

type nvmeshTomaDaemonSet struct{}
type nvmeshMcsDaemonSet struct{}
type nvmeshAgentDaemonSet struct{}
