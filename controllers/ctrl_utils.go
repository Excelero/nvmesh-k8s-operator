package controllers

import (
	"context"
	"fmt"
	ioutil "io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"

	rbac "k8s.io/api/rbac/v1"

	errors "github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/watch"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/pointer"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (

	// keep all watcher to stop them before cerating new ones
	watchers []watch.Interface
)

func (r *NVMeshReconciler) ReconcileObject(cr *nvmeshv1.NVMesh, newObj *runtime.Object, component *NVMeshComponent, removeObject bool) error {
	if removeObject == false {
		return r.MakeSureObjectExists(cr, newObj, component)
	} else {
		return r.MakeSureObjectRemoved(cr, newObj, component)
	}
}

func (r *NVMeshReconciler) MakeSureObjectExists(cr *nvmeshv1.NVMesh, newObj *runtime.Object, component *NVMeshComponent) error {
	v1obj := (*newObj).(metav1.Object)
	v1obj.SetNamespace(cr.GetNamespace())

	// Add general labels
	labels := v1obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["app.kubernetes.io/managed-by"] = "nvmesh-operator"
	v1obj.SetLabels(labels)

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
	if err != nil && k8serrors.IsNotFound(err) {
		log.Info("Creating new object")
		err = r.Client.Create(context.TODO(), *newObj)
		return err
	} else if err != nil {
		log.Error(err, "Error while getting object")
		return err
	} else if (*component).ShouldUpdateObject(cr, newObj, foundObj) {
		log.Info("shouldUpdate returned true > Updating...")

		// Update the resource version before an update
		v1objFound := (*foundObj).(metav1.Object)
		v1objNew := (*newObj).(metav1.Object)
		v1objNew.SetResourceVersion(v1objFound.GetResourceVersion())

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
	v1obj := (*newObj).(metav1.Object)
	v1obj.SetNamespace(cr.GetNamespace())
	name, kind := GetRunetimeObjectNameAndKind(newObj)
	log := r.Log.WithValues("method", "MakeSureObjectRemoved", "name", name, "kind", kind)

	_, err := r.getGenericObject(newObj, cr.GetNamespace())
	if err != nil && k8serrors.IsNotFound(err) {
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

func (r *NVMeshReconciler) getDecoder() runtime.Decoder {
	var Codecs = serializer.NewCodecFactory(r.Scheme)
	return Codecs.UniversalDeserializer()
}

func (r *NVMeshReconciler) ReconcileYamlObjectsFromFile(cr *nvmeshv1.NVMesh, filename string, component NVMeshComponent, removeObject bool) error {
	log := r.Log.WithValues("method", "reconcileYamlObjectsFromFile", "filename", filename)

	decoder := r.getDecoder()
	objects, err := YamlFileToObjects(filename, decoder)
	if err != nil {
		if _, ok := err.(*YamlFileParseError); ok {
			// this is ok
			msg := fmt.Sprintf("Some Documents in the file failed to parse %+v", err)
			log.Info(msg)
		} else {
			return err
		}
	}

	var reconcileErrors []error
	for _, obj := range objects {
		err = r.ReconcileObject(cr, &obj, &component, removeObject)
		if err != nil {
			log.Info(fmt.Sprintf("Failed to Reconcile %s %+v\n", reflect.TypeOf(obj), err))
			reconcileErrors = append(reconcileErrors, err)
		}
	}

	if len(reconcileErrors) > 0 {
		return reconcileErrors[0]
	}

	return nil
}

func (r *NVMeshReconciler) CreateObjectsFromDir(cr *nvmeshv1.NVMesh, comp NVMeshComponent, dir string, recursive bool) error {
	return r.ReconcileYamlObjectsFromDir(cr, comp, dir, false, recursive)
}

func (r *NVMeshReconciler) RemoveObjectsFromDir(cr *nvmeshv1.NVMesh, comp NVMeshComponent, dir string, recursive bool) error {
	return r.ReconcileYamlObjectsFromDir(cr, comp, dir, true, recursive)
}

func (r *NVMeshReconciler) ReconcileYamlObjectsFromDir(cr *nvmeshv1.NVMesh, comp NVMeshComponent, dir string, removeObjects bool, recursive bool) error {
	files, err := ListFilesInDir(dir, recursive)
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
	v1obj := (*obj).(metav1.Object)
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

func ListFilesInDir(dir string, recursive bool) ([]string, error) {
	if recursive == true {
		return ListFilesInSubDirs(dir)
	} else {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			return nil, err
		}

		filenames := make([]string, 0)
		for _, f := range files {
			if !f.IsDir() {
				filenames = append(filenames, path.Join(dir, f.Name()))
			}
		}

		return filenames, nil
	}
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
	for i := range crb.Subjects {
		crb.Subjects[i].Namespace = ns
	}
}

func addNamespaceToRoleBinding(cr *nvmeshv1.NVMesh, rb *rbac.RoleBinding) {
	ns := cr.GetNamespace()
	for i := range rb.Subjects {
		rb.Subjects[i].Namespace = ns
	}
}

func SetControllerReferenceOnUnstructured(owner metav1.Object, object *unstructured.Unstructured, gvk *schema.GroupVersionKind) {
	ref := metav1.OwnerReference{
		APIVersion:         gvk.GroupVersion().String(),
		Kind:               gvk.Kind,
		Name:               owner.GetName(),
		UID:                owner.GetUID(),
		BlockOwnerDeletion: pointer.BoolPtr(true),
		Controller:         pointer.BoolPtr(true),
	}

	addOwnerReference(ref, object)
}

// Returns true if a and b point to the same object
func referSameObject(a, b metav1.OwnerReference) bool {
	aGV, err := schema.ParseGroupVersion(a.APIVersion)
	if err != nil {
		return false
	}

	bGV, err := schema.ParseGroupVersion(b.APIVersion)
	if err != nil {
		return false
	}

	return aGV.Group == bGV.Group && a.Kind == b.Kind && a.Name == b.Name
}

func addOwnerReference(ref metav1.OwnerReference, object *unstructured.Unstructured) {
	owners := object.GetOwnerReferences()
	idx := indexOwnerRef(owners, ref)
	if idx == -1 {
		owners = append(owners, ref)
	} else {
		owners[idx] = ref
	}
	object.SetOwnerReferences(owners)
}

// indexOwnerRef returns the index of the owner reference in the slice if found, or -1.
func indexOwnerRef(ownerReferences []metav1.OwnerReference, ref metav1.OwnerReference) int {
	for index, r := range ownerReferences {
		if referSameObject(r, ref) {
			return index
		}
	}
	return -1
}

func (r *NVMeshReconciler) stopAllUnstructuredWatchers() {
	for _, w := range watchers {
		w.Stop()
	}
}

func (r *NVMeshReconciler) listenOnChanAndReconcile(ch <-chan watch.Event) {
	log := r.Log.WithValues("method", "listenOnChanAndReconcile")

	for e := range ch {
		log.Info(fmt.Sprintf("received Event %s %s", e.Object.GetObjectKind().GroupVersionKind().Kind, e.Type))
		//fmt.Printf("received Event %+v", e)
		// TODO: Enqueue Reconcile
		// NOTE: this runs in a separate goroutine so we should not reconcile here but enqueue the reconcile request
	}
}

func (r *NVMeshReconciler) addUnstructuredWatch(res dynamic.ResourceInterface, obj *unstructured.Unstructured) error {
	opt := metav1.ListOptions{FieldSelector: "metadata.name=" + obj.GetName()}
	watcher, err := res.Watch(context.TODO(), opt)
	if err != nil {
		return err
	}
	watchers = append(watchers, watcher)
	go r.listenOnChanAndReconcile(watcher.ResultChan())
	return nil
}

func (r *NVMeshReconciler) ReconcileUnstructuredObjects(cr *nvmeshv1.NVMesh, directoryPath string, shouldCreate bool) error {
	log := r.Log.WithValues("method", "ReconcileUnstructuredObjects")

	var errList []error = make([]error, 0)

	files, listFilesErr := ListFilesInSubDirs(directoryPath)
	if listFilesErr != nil {
		return listFilesErr
	}

	for _, file := range files {
		obj, gvk, err := YamlFileToUnstructured(file)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Error while trying to read Unstructured Object from YAML file %s", file))
		}

		gvrMapping, err := findGVR(gvk, r.Manager.GetConfig())
		if err != nil {
			log.Info(fmt.Sprintf("Warning: failed to find GroupVersionResource for object %s, if this is a CustomResource it is possible the CRD for it is not loaded\n", gvk))
			continue
		}

		var ns string
		if stringInSlice(gvk.Kind, GloballyNamedKinds) {
			// Object Kind does not require namespace
			ns = ""
		} else {
			// Object Kind requires namespace
			ns = cr.GetNamespace()
		}
		res := r.DynamicClient.Resource(gvrMapping.Resource).Namespace(ns)
		objName := obj.GetName()

		err = r.addUnstructuredWatch(res, obj)
		if err != nil {
			return err
		}

		_, err = res.Get(context.TODO(), objName, metav1.GetOptions{})
		if err != nil && k8serrors.IsNotFound(err) {
			if shouldCreate == true {
				SetControllerReferenceOnUnstructured(cr, obj, gvk)
				_, err = res.Create(context.TODO(), obj, metav1.CreateOptions{})
				if err != nil {
					objJson := UnstructuredToString(*obj)
					wrappedErr := errors.Wrap(err, fmt.Sprintf("Error while trying to create object using dynamic client %s. Object: %s", gvrMapping.Resource, objJson))
					errList = append(errList, wrappedErr)
					log.Info(fmt.Sprintln(wrappedErr))
				} else {
					log.Info(fmt.Sprintf("%s %s Object Created\n", gvk.Kind, objName))
				}
			} else {
				log.Info(fmt.Sprintf("%s %s Nothing to do\n", gvk.Kind, objName))
			}

		} else if err != nil {
			wrappedErr := errors.Wrap(err, fmt.Sprintf("Error while trying to get object using dynamic client %s", gvrMapping.Resource))
			errList = append(errList, wrappedErr)
		} else {
			//TODO: Object found - check if we need to update ?
			if shouldCreate == true {
				log.Info(fmt.Sprintf("%s %s already exists\n", gvk.Kind, objName))
			} else {
				err = res.Delete(context.TODO(), objName, metav1.DeleteOptions{})
				if err != nil {
					wrappedErr := errors.Wrap(err, fmt.Sprintf("Error while trying to delete object using dynamic client %s", gvrMapping.Resource))
					errList = append(errList, wrappedErr)
					log.Info(fmt.Sprintln(wrappedErr))
				} else {
					log.Info(fmt.Sprintf("%s %s Object Deleted\n", gvk.Kind, objName))
				}
			}
		}
	}

	if len(errList) > 0 {
		return errList[0]
	} else {
		return nil
	}
}
