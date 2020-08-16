package controllers

import (
	"context"
	goerrors "errors"
	"reflect"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1alpha1"
	"excelero.com/nvmesh-k8s-operator/importutil"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	MgmtAssetsLocation  = "config/samples/management/"
	MgmtStatefulSetName = "nvmesh-management"
	MgmtImageName       = "docker.excelero.com/nvmesh-management"
)

type NVMeshMgmtReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

type MgmtManagedObject interface {
	shouldUpdate(*nvmeshv1.NVMesh, *runtime.Object) bool
	newObject(*nvmeshv1.NVMesh) (*runtime.Object, error)
	getObject(*nvmeshv1.NVMesh, *NVMeshMgmtReconciler) (*runtime.Object, error)
}

type mgmtStatefulset struct{}

func (r *mgmtStatefulset) newObject(cr *nvmeshv1.NVMesh) (*runtime.Object, error) {
	obj, err := importutil.YamlFileToObject(MgmtAssetsLocation + "statefulset.yaml")
	if err != nil {
		return nil, err
	}

	// set fields
	o := obj.(*appsv1.StatefulSet)

	if cr.Spec.Management.Image != "" {
		o.Spec.Template.Spec.Containers[0].Image = cr.Spec.Management.Image
	} else {
		if cr.Spec.Management.Version == "" {
			return nil, goerrors.New("Missing Management Version (NVMesh.Spec.Management.Version)")
		}
		o.Spec.Template.Spec.Containers[0].Image = r.getImageFromVersion(cr.Spec.Management.Version)
	}

	//TODO: set still use configMap or set values directly into the daemonset ?
	return &obj, err
}

func (r *mgmtStatefulset) getImageFromVersion(version string) string {
	return MgmtImageName + ":" + version
}

func (r *mgmtStatefulset) shouldUpdate(cr *nvmeshv1.NVMesh, o *runtime.Object) bool {
	ss := (*o).(*appsv1.StatefulSet)
	if cr.Spec.Management.Image != "" {
		if cr.Spec.Management.Image != ss.Spec.Template.Spec.Containers[0].Image {
			return true
		}
	} else {
		// Image field not defined - check version matches
		if r.getImageFromVersion(cr.Spec.Management.Version) != ss.Spec.Template.Spec.Containers[0].Image {
			return true
		}
	}

	return false
}

func (r *mgmtStatefulset) getObject(cr *nvmeshv1.NVMesh, reconciler *NVMeshMgmtReconciler) (*runtime.Object, error) {
	foundStatefulSet := &appsv1.StatefulSet{}
	err := reconciler.Client.Get(context.TODO(), types.NamespacedName{Name: MgmtStatefulSetName, Namespace: cr.Namespace}, foundStatefulSet)
	var obj runtime.Object
	obj = foundStatefulSet
	return &obj, err
}

func (r *NVMeshMgmtReconciler) Reconcile(expected *nvmeshv1.NVMesh) error {
	var ss MgmtManagedObject = &mgmtStatefulset{}
	managedObjects := []MgmtManagedObject{ss}
	for _, o := range managedObjects {
		err := r.reconcileObject(expected, o)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *NVMeshMgmtReconciler) reconcileObject(expected *nvmeshv1.NVMesh, managedObject MgmtManagedObject) error {
	mylog := r.Log.WithValues("Management managedObject", reflect.TypeOf(managedObject).Elem().Name())

	newObj, err := managedObject.newObject(expected)
	if err != nil {
		return err
	}

	// Set NVMesh instance as the owner and controller
	v1obj := (*newObj).(v1.Object)
	v1obj.SetNamespace(expected.GetNamespace())
	if err := controllerutil.SetControllerReference(expected, v1obj, r.Scheme); err != nil {
		return err
	}

	foundObj, err := managedObject.getObject(expected, r)
	if err != nil && errors.IsNotFound(err) {
		mylog.Info("Object not found, creating new object")
		err = r.Client.Create(context.TODO(), *newObj)
		return err
	} else if err != nil {
		mylog.Error(err, "Error while getting object")
		return err
	} else if managedObject.shouldUpdate(expected, foundObj) {
		mylog.Info("Should update object returned true > Updating...")
		err = r.Client.Update(context.TODO(), *newObj)
		return err
	} else {
		mylog.Info("reconcileObject Nothing to do")
	}

	return nil
}
