package controllers

import (
	"fmt"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NVMeshCoreAssestLocation   = "config/samples/nvmesh-core"
	CoreUserspaceDaemonSetName = "nvmesh-core-user-space"
	TargetDriverDaemonSetName  = "nvmesh-target-driver-container"
	ClientDriverDaemonSetName  = "nvmesh-client-driver-container"
	DriverContainerImageName   = "nvmesh-driver-container"
	//TODO: expose ImageRegistry to user in custom resource
	ImageRegistry = ""
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
	name, _ := GetRunetimeObjectNameAndKind(obj)
	switch o := (*obj).(type) {
	case *appsv1.DaemonSet:
		switch name {
		case CoreUserspaceDaemonSetName:
			err := r.initUserspaceDaemonSet(cr, o)
			return err
		}
	default:
	}

	return nil
}

func (r *NVMeshCoreReconciler) ShouldUpdateObject(cr *nvmeshv1.NVMesh, exp *runtime.Object, obj *runtime.Object) bool {
	name, _ := GetRunetimeObjectNameAndKind(obj)
	switch o := (*obj).(type) {
	case *appsv1.DaemonSet:
		expDS := (*exp).(*appsv1.DaemonSet)
		switch name {
		case CoreUserspaceDaemonSetName:
			fallthrough
		case TargetDriverDaemonSetName:
			fallthrough
		case "nvmesh-client-driver-container":
			return r.shouldUpdateDaemonSet(cr, expDS, o)
		}
	default:
	}

	return false
}

func (r *NVMeshCoreReconciler) shouldUpdateDaemonSet(cr *nvmeshv1.NVMesh, expected *appsv1.DaemonSet, ds *appsv1.DaemonSet) bool {
	log := r.Log.WithValues("method", "shouldUpdateDaemonSet")

	for i, c := range ds.Spec.Template.Spec.Containers {
		expectedImage := expected.Spec.Template.Spec.Containers[i].Image
		if c.Image != expectedImage {
			log.Info(fmt.Sprintf("Image missmatch on DaemonSet %s Container %s expected: %s found: %s", ds.ObjectMeta.Name, c.Name, expectedImage, c.Image))
			return true
		}
	}

	return false
}

func (r *NVMeshCoreReconciler) initUserspaceDaemonSet(cr *nvmeshv1.NVMesh, ds *appsv1.DaemonSet) error {
	var imageName string
	for i, c := range ds.Spec.Template.Spec.Containers {
		switch c.Name {
		case "mcs":
			fallthrough
		case "agent":
			imageName = "nvmesh-mcs"
		case "toma":
			imageName = "nvmesh-toma"
		case "tracer":
			imageName = "nvmesh-tracer"
		}

		ds.Spec.Template.Spec.Containers[i].Image = ImageRegistry + imageName + ":" + cr.Spec.Core.Version
	}

	return nil
}

func (r *NVMeshCoreReconciler) initDriverContainerDaemonSet(cr *nvmeshv1.NVMesh, ds *appsv1.DaemonSet) error {
	for _, c := range ds.Spec.Template.Spec.Containers {
		if c.Name == "driver-container" {
			c.Image = ImageRegistry + DriverContainerImageName + ":" + cr.Spec.Core.Version
		}
	}

	return nil
}
