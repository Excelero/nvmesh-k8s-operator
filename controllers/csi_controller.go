package controllers

import (
	"context"
	goerrors "errors"
	"reflect"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1alpha1"
	"excelero.com/nvmesh-k8s-operator/importutil"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	CSIAssetsLocation  = "config/samples/csi/"
	CSIDaemonSetName   = "nvmesh-csi-node-driver"
	CSIStatefulSetName = "nvmesh-csi-controller"
	CSIDriverImageName = "excelero/nvmesh-csi-driver"
)

type NVMeshCSIReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

type CSIManagedObject interface {
	shouldUpdate(*nvmeshv1.NVMesh, *runtime.Object) bool
	newObject(*nvmeshv1.NVMesh) (*runtime.Object, error)
	getObject(*nvmeshv1.NVMesh, *NVMeshCSIReconciler) (*runtime.Object, error)
}

type nodeDriverDaemonSet struct{}

func (r *nodeDriverDaemonSet) newObject(cr *nvmeshv1.NVMesh) (*runtime.Object, error) {
	obj, err := importutil.YamlFileToObject(CSIAssetsLocation + "daemonset_node_driver.yaml")
	if err != nil {
		return nil, err
	}

	// set fields
	o := obj.(*appsv1.DaemonSet)

	if cr.Spec.CSI.Image != "" {
		o.Spec.Template.Spec.Containers[0].Image = cr.Spec.CSI.Image
	} else {
		if cr.Spec.CSI.Version == "" {
			return nil, goerrors.New("Missing NVMesh CSI Driver Version (NVMesh.Spec.CSI.Version)")
		}

		o.Spec.Template.Spec.Containers[0].Image = r.getImageFromVersion(cr.Spec.CSI.Version)
	}

	//TODO: set still use configMap or set values directly into the daemonset ?
	return &obj, err
}

func (r *nodeDriverDaemonSet) getImageFromVersion(version string) string {
	return CSIDriverImageName + ":" + version
}

func (r *nodeDriverDaemonSet) shouldUpdate(cr *nvmeshv1.NVMesh, o *runtime.Object) bool {
	ds := (*o).(*appsv1.DaemonSet)
	if cr.Spec.CSI.Image != "" {
		if cr.Spec.CSI.Image != ds.Spec.Template.Spec.Containers[0].Image {
			return true
		}
	} else {
		// Image field not defined - check version matches
		if r.getImageFromVersion(cr.Spec.CSI.Version) != ds.Spec.Template.Spec.Containers[0].Image {
			return true
		}
	}

	return false
}

func (r *nodeDriverDaemonSet) getObject(cr *nvmeshv1.NVMesh, reconciler *NVMeshCSIReconciler) (*runtime.Object, error) {
	foundDaemonSet := &appsv1.DaemonSet{}
	err := reconciler.Client.Get(context.TODO(), types.NamespacedName{Name: CSIDaemonSetName, Namespace: cr.Namespace}, foundDaemonSet)
	var obj runtime.Object
	obj = foundDaemonSet
	return &obj, err
}

type ctrlStatefulSet struct{}

func (r *ctrlStatefulSet) shouldUpdate(cr *nvmeshv1.NVMesh, o *runtime.Object) bool {
	ss := (*o).(*appsv1.StatefulSet)
	if cr.Spec.CSI.ControllerReplicas != *ss.Spec.Replicas {
		return true
	}

	if cr.Spec.CSI.Image != "" && cr.Spec.CSI.Image != ss.Spec.Template.Spec.Containers[0].Image {
		return true
	}

	return false
}

func (r *ctrlStatefulSet) getObject(cr *nvmeshv1.NVMesh, reconciler *NVMeshCSIReconciler) (*runtime.Object, error) {
	foundStatefulSet := &appsv1.StatefulSet{}
	err := reconciler.Client.Get(context.TODO(), types.NamespacedName{Name: CSIStatefulSetName, Namespace: cr.Namespace}, foundStatefulSet)
	var obj runtime.Object
	obj = foundStatefulSet
	return &obj, err
}

func (r *ctrlStatefulSet) newObject(cr *nvmeshv1.NVMesh) (*runtime.Object, error) {
	var statefulset *appsv1.StatefulSet
	obj, err := importutil.YamlFileToObject(CSIAssetsLocation + "statefulset_controller.yaml")
	if err != nil {
		return nil, err
	}

	statefulset = obj.(*appsv1.StatefulSet)

	if cr.Spec.CSI.Image != "" {
		statefulset.Spec.Template.Spec.Containers[0].Image = cr.Spec.CSI.Image
	}

	// set replicas from CustomResource
	statefulset.Spec.Replicas = &cr.Spec.CSI.ControllerReplicas
	return &obj, err
}

func (r *NVMeshCSIReconciler) Reconcile(expected *nvmeshv1.NVMesh) error {
	var ds CSIManagedObject = &nodeDriverDaemonSet{}
	var ss CSIManagedObject = &ctrlStatefulSet{}
	managedObjects := []CSIManagedObject{ds, ss}
	for _, o := range managedObjects {
		err := r.reconcileObject(expected, o)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *NVMeshCSIReconciler) reconcileObject(expected *nvmeshv1.NVMesh, managedObject CSIManagedObject) error {
	log := r.Log.WithValues("CSI managedObject", reflect.TypeOf(managedObject).Elem().Name())

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
		log.Info("Object not found, creating new object")
		err = r.Client.Create(context.TODO(), *newObj)
		return err
	} else if err != nil {
		log.Error(err, "Error while getting object")
		return err
	} else if managedObject.shouldUpdate(expected, foundObj) {
		log.Info("shouldUpdate returned true > Updating...")
		err = r.Client.Update(context.TODO(), *newObj)
		return err
	} else {
		log.Info("reconcileObject Nothing to do")
	}

	return nil
}
