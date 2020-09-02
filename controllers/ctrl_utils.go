package controllers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *NVMeshReconciler) ReconcileObject(cr *nvmeshv1.NVMesh, newObj *runtime.Object, component *NVMeshComponent, removeObject bool) error {
	if removeObject == false {
		return r.MakeSureObjectExists(cr, newObj, component)
	} else {
		return r.MakeSureObjectRemoved(cr, newObj, component)
	}
}

func (r *NVMeshReconciler) MakeSureObjectExists(cr *nvmeshv1.NVMesh, newObj *runtime.Object, component *NVMeshComponent) error {
	v1obj := (*newObj).(v1.Object)
	v1obj.SetNamespace(cr.GetNamespace())
	name, kind := GetRunetimeObjectNameAndKind(newObj)
	log := r.Log.WithValues("method", "MakeSureObjectExists", "name", name, "kind", kind)

	err := (*component).InitObject(cr, newObj)
	if err != nil {
		log.Info("Error running InitObject")
		return err
	}

	// Set NVMesh instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, v1obj, r.Scheme); err != nil {
		if err != nil {
			log.Error(err, "Error running SetControllerReference")
		}
	}

	foundObj, err := r.getGenericObject(newObj, cr.GetNamespace())
	if err != nil && errors.IsNotFound(err) {
		log.Info("Object not found, creating new object")
		err = r.Client.Create(context.TODO(), *newObj)
		return err
	} else if err != nil {
		log.Error(err, "Error while getting object")
		return err
	} else if (*component).ShouldUpdateObject(cr, newObj, foundObj) {
		log.Info("shouldUpdate returned true > Updating...")
		err = r.Client.Update(context.TODO(), *newObj)
		if err != nil {
			log.Info("Error updating object")
		} else {
			log.Info("update Successfull")
		}
		return err
	} else {
		log.Info("Nothing to do")
	}

	return nil
}

func (r *NVMeshReconciler) MakeSureObjectRemoved(cr *nvmeshv1.NVMesh, newObj *runtime.Object, component *NVMeshComponent) error {
	v1obj := (*newObj).(v1.Object)
	v1obj.SetNamespace(cr.GetNamespace())
	name, kind := GetRunetimeObjectNameAndKind(newObj)
	log := r.Log.WithValues("method", "MakeSureObjectRemoved", "name", name, "kind", kind)

	_, err := r.getGenericObject(newObj, cr.GetNamespace())
	if err != nil && errors.IsNotFound(err) {
		log.Info("Nothing to do")
	} else if err != nil {
		log.Error(err, "Error while trying to find out if object exists")
		return err
	} else {
		log.Info("Deleting Object")
		err = r.Client.Delete(context.TODO(), *newObj)
		if err != nil {
			log.Info("Error deleting object")
		}
		return err
	}

	return nil
}

func (r *NVMeshReconciler) getGenericObject(fromObject *runtime.Object, namespace string) (*runtime.Object, error) {
	// Extract name and namespace without knowing the type
	name, kind := GetRunetimeObjectNameAndKind(fromObject)
	//r.Log.Info("Going to get Object", "ns", namespace, "name", name, "kind", kind)
	if stringInSlice(kind, GloballyNamedKinds) {
		namespace = ""
	}

	foundObject := (*fromObject).DeepCopyObject()
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, foundObject)
	return &foundObject, err
}

func (r *NVMeshReconciler) ReconcileYamlObjectsFromFile(cr *nvmeshv1.NVMesh, filename string, component NVMeshComponent, removeObject bool) error {
	log := r.Log.WithValues("method", "reconcileYamlObjectsFromFile", "filename", filename)

	objects, err := YamlFileToObjects(filename)
	if err != nil {
		if _, ok := err.(*YamlFileParseError); ok {
			// this is ok
			msg := fmt.Sprintf("Some Documents in the file failed to parse %+v", err)
			fmt.Print(msg)
			log.Info(msg)
		} else {
			return err
		}
	}

	var reconcileErrors []error
	for _, obj := range objects {
		err = r.ReconcileObject(cr, &obj, &component, removeObject)
		if err != nil {
			fmt.Printf("Failed to Reconcile %s %+v\n", reflect.TypeOf(obj), err)
			reconcileErrors = append(reconcileErrors, err)
		}
	}

	if len(reconcileErrors) > 0 {
		return reconcileErrors[0]
	}

	return nil
}

func (r *NVMeshReconciler) CreateObjectsFromDir(cr *nvmeshv1.NVMesh, comp NVMeshComponent, dir string) error {
	return r.ReconcileYamlObjectsFromDir(cr, comp, dir, false)
}

func (r *NVMeshReconciler) RemoveObjectsFromDir(cr *nvmeshv1.NVMesh, comp NVMeshComponent, dir string) error {
	return r.ReconcileYamlObjectsFromDir(cr, comp, dir, true)
}

func (r *NVMeshReconciler) ReconcileYamlObjectsFromDir(cr *nvmeshv1.NVMesh, comp NVMeshComponent, dir string, removeObjects bool) error {
	files, err := ListFilesInSubDirs(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		err = r.ReconcileYamlObjectsFromFile(cr, file, comp, removeObjects)
		if err != nil {
			return err
		}
	}

	return nil
}

func GetRunetimeObjectNameAndKind(obj *runtime.Object) (string, string) {
	v1obj := (*obj).(v1.Object)
	name := v1obj.GetName()
	kind := (*obj).GetObjectKind().GroupVersionKind().Kind
	return name, kind
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func ListFilesInSubDirs(root string) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})

	return files, err
}

func addNamespaceToClusterRoleBinding(cr *nvmeshv1.NVMesh, crb *rbac.ClusterRoleBinding) {
	ns := cr.GetNamespace()
	for _, sub := range crb.Subjects {
		sub.Namespace = ns
	}
}
