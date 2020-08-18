package controllers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1alpha1"
	"excelero.com/nvmesh-k8s-operator/importutil"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *NVMeshReconciler) ReconcileGenericObject(cr *nvmeshv1.NVMesh, newObj *runtime.Object, component NVMeshComponent) error {
	v1obj := (*newObj).(v1.Object)
	v1obj.SetNamespace(cr.GetNamespace())
	name, kind := GetRunetimeObjectNameAndKind(newObj)
	log := r.Log.WithValues("method", "ReconcileGenericObject", "name", name, "kind", kind)
	if kind == "StatefulSet" {
		o := (*newObj).(*appsv1.StatefulSet)
		fmt.Printf("before callInitObj name: %s kind: %s version: %s\n", name, kind, o.Spec.Template.Spec.Containers[0].Image)
	}

	component.InitiateObject(cr, newObj)

	if kind == "StatefulSet" {
		o := (*newObj).(*appsv1.StatefulSet)
		fmt.Printf("after callInitObj name: %s kind: %s version: %s\n", name, kind, o.Spec.Template.Spec.Containers[0].Image)
	}

	// Set NVMesh instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, v1obj, r.Scheme); err != nil {
		//return err
	}

	foundObj, err := r.getGenericObject(newObj, cr.GetNamespace())
	if err != nil && errors.IsNotFound(err) {
		log.Info("Object not found, creating new object")
		err = r.Client.Create(context.TODO(), *newObj)
		return err
	} else if err != nil {
		log.Error(err, "Error while getting object")
		return err
	} else if component.ShouldUpdateObject(cr, foundObj) {
		log.Info("shouldUpdate returned true > Updating...")
		err = r.Client.Update(context.TODO(), *newObj)
		//FIXME - Update not having any effect
		if err != nil {
			log.Info("Error updating object")
		} else {
			log.Info("update Successfull")
		}
		return err
	} else {
		log.Info("reconcileObject Nothing to do")
	}

	return nil
}

func (r *NVMeshReconciler) getGenericObject(fromObject *runtime.Object, namespace string) (*runtime.Object, error) {
	runtimeObj := (*fromObject).(runtime.Object)

	// Extract name and namespace without knowing the type
	name, kind := GetRunetimeObjectNameAndKind(fromObject)
	//r.Log.Info("Going to get Object", "ns", namespace, "name", name, "kind", kind)
	if stringInSlice(kind, GloballyNamedKinds) {
		namespace = ""
	}

	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, runtimeObj)
	return &runtimeObj, err
}

func (r *NVMeshReconciler) ReconcileYamlObjectsFromFile(cr *nvmeshv1.NVMesh, filename string, component NVMeshComponent) error {
	log := r.Log.WithValues("method", "reconcileYamlObjectsFromFile", "filename", filename)

	objects, err := importutil.YamlFileToObjects(filename)
	if err != nil {
		if _, ok := err.(*importutil.YamlFileParseError); ok {
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
		err = r.ReconcileGenericObject(cr, &obj, component)
		if err != nil {
			fmt.Printf("Failed to Reconcile %s %+v", reflect.TypeOf(obj), err)
			reconcileErrors = append(reconcileErrors, err)
		}
	}

	if len(reconcileErrors) > 0 {
		return reconcileErrors[0]
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
