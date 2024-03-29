package controllers

import (
	"context"
	"fmt"
	ioutil "io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"time"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"

	rbac "k8s.io/api/rbac/v1"

	errors "github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/pointer"

	securityv1 "github.com/openshift/api/security/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	nvmeshMgmtLabelKey        = "nvmesh.excelero.com/nvmesh-management"
	nvmeshClientLabelKey      = "nvmesh.excelero.com/nvmesh-client"
	nvmeshTargetLabelKey      = "nvmesh.excelero.com/nvmesh-target"
	nvmeshClusterNameLabelKey = "nvmesh.excelero.com/cluster-name"

	fileServerSecretName       = "nvmesh-file-server-cred"
	exceleroRegistrySecretName = "excelero-registry-cred"

	operatorSCCName = "nvmesh-operator"
)

var (

	// keep all watcher to stop them before creating new ones
	watchers []watch.Interface
)

func (r *NVMeshReconciler) reconcileObject(cr *nvmeshv1.NVMesh, newObj *runtime.Object, component *NVMeshComponent, removeObject bool) error {
	if removeObject == false {
		return r.makeSureObjectExists(cr, newObj, component)
	}

	return r.makeSureObjectRemoved(cr, newObj, component)
}

func (r *NVMeshReconciler) getOperatorLabels(cr *nvmeshv1.NVMesh) map[string]string {
	return map[string]string{
		"app.kubernetes.io/managed-by": "nvmesh-operator",
		nvmeshClusterNameLabelKey:      cr.GetName(),
	}
}

func (r *NVMeshReconciler) makeSureObjectExists(cr *nvmeshv1.NVMesh, newObj *runtime.Object, component *NVMeshComponent) error {
	v1obj := (*newObj).(metav1.Object)
	v1obj.SetNamespace(cr.GetNamespace())

	// Add general labels
	labels := v1obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	opLabels := r.getOperatorLabels(cr)
	for k, v := range opLabels {
		labels[k] = v
	}

	v1obj.SetLabels(labels)

	name, kind := getRunetimeObjectNameAndKind(newObj)
	log := r.Log.WithValues("method", "makeSureObjectExists", "name", name, "kind", kind)

	if component != nil {
		err := (*component).InitObject(cr, newObj)
		if err != nil {
			log.Info("Error running InitObject")
			return err
		}
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
		if k8serrors.IsAlreadyExists(err) {
			log.Info(fmt.Sprintf("WARNING: Object Already exists. while trying to create %s %s", kind, name))
			return nil
		}
		return err
	} else if err != nil {
		log.Error(err, "Error while getting object")
		return err
	} else if component != nil && (*component).ShouldUpdateObject(cr, newObj, foundObj) {
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
		//log.Info("Nothing to do")
	}

	return nil
}

func (r *NVMeshReconciler) makeSureObjectRemoved(cr *nvmeshv1.NVMesh, newObj *runtime.Object, component *NVMeshComponent) error {
	v1obj := (*newObj).(metav1.Object)
	v1obj.SetNamespace(cr.GetNamespace())
	name, kind := getRunetimeObjectNameAndKind(newObj)
	log := r.Log.WithValues("method", "makeSureObjectRemoved", "name", name, "kind", kind)

	_, err := r.getGenericObject(newObj, cr.GetNamespace())
	if err != nil && k8serrors.IsNotFound(err) {
		//log.Info("Nothing to do")
	} else if err != nil {
		if _, ok := err.(*meta.NoKindMatchError); ok && kind == "SecurityContextConstraints" {
			if r.Options.IsOpenShift {
				log.Info("Ignoring no kind SecurityContextConstraints error, As this is not an openshift cluster (check -openshift flag)")
			} else {
				log.Info("Got No kind SecurityContextConstraints error from Kubernetes API, Seems this is not an openshift cluster, But -openshift flag is set")
			}

			return nil

		}

		log.Error(err, "Error while trying to find out if object exists")

		return err
	} else {
		log.Info("Deleting Object")
		err = r.Client.Delete(context.TODO(), *newObj)
		if err != nil && !k8serrors.IsNotFound(err) {
			log.Info("Error deleting object")
		}
		return err
	}

	return nil
}

func (r *NVMeshReconciler) getGenericObject(fromObject *runtime.Object, namespace string) (*runtime.Object, error) {
	// Extract name and namespace without knowing the type
	name, kind := getRunetimeObjectNameAndKind(fromObject)
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

func (r *NVMeshReconciler) reconcileYamlObjectsFromFile(cr *nvmeshv1.NVMesh, filename string, component NVMeshComponent, removeObject bool) error {
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
		err = r.reconcileObject(cr, &obj, &component, removeObject)
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

func (r *NVMeshReconciler) createObjectsFromDir(cr *nvmeshv1.NVMesh, comp NVMeshComponent, dir string, recursive bool) error {
	return r.reconcileYamlObjectsFromDir(cr, comp, dir, false, recursive)
}

func (r *NVMeshReconciler) removeObjectsFromDir(cr *nvmeshv1.NVMesh, comp NVMeshComponent, dir string, recursive bool) error {
	return r.reconcileYamlObjectsFromDir(cr, comp, dir, true, recursive)
}

func (r *NVMeshReconciler) reconcileYamlObjectsFromDir(cr *nvmeshv1.NVMesh, comp NVMeshComponent, dir string, removeObjects bool, recursive bool) error {
	files, err := listFilesInDir(dir, recursive)
	if err != nil {
		return err
	}

	for _, file := range files {
		err = r.reconcileYamlObjectsFromFile(cr, file, comp, removeObjects)
		if err != nil {
			return err
		}
	}

	return nil
}

func getRunetimeObjectNameAndKind(obj *runtime.Object) (string, string) {
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

func listFilesInDir(dir string, recursive bool) ([]string, error) {
	if recursive == true {
		return listFilesInSubDirs(dir)
	}

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

func listFilesInSubDirs(root string) ([]string, error) {
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

func setControllerReferenceOnUnstructured(owner metav1.Object, object *unstructured.Unstructured, gvk *schema.GroupVersionKind) {
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
		// NOTE: this runs in a separate goroutine so we should not reconcile here but enqueue the reconcile request
		// TODO: Enqueue Reconcile
		// This flow is relevant only for the mongodb custom-resource object when deploying a mongodb operator
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

type unstructuredUpdater func(*nvmeshv1.NVMesh, *unstructured.Unstructured, *schema.GroupVersionKind)

func (r *NVMeshReconciler) reconcileUnstructuredObjects(cr *nvmeshv1.NVMesh, directoryPath string, shouldCreate bool, processFunc unstructuredUpdater) error {
	log := r.Log.WithValues("method", "reconcileUnstructuredObjects")

	var errList []error = make([]error, 0)

	files, listFilesErr := listFilesInSubDirs(directoryPath)
	if listFilesErr != nil {
		return listFilesErr
	}

	for _, file := range files {
		obj, gvk, err := YamlFileToUnstructured(file)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Error while trying to read Unstructured Object from YAML file %s", file))
		}

		processFunc(cr, obj, gvk)

		gvrMapping, err := findGVR(gvk, r.Manager.GetConfig())
		if err != nil {
			// Ingore this error, it will occure when we are looking for a Custom Resource but it's CRD is not deployed.
			//log.Info(fmt.Sprintf("Warning: failed to find GroupVersionResource for object %s, if this is a CustomResource it is possible the CRD for it is not loaded\n", gvk))
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

		if shouldCreate == true {
			err = r.addUnstructuredWatch(res, obj)
			if err != nil {
				return err
			}
		}

		_, err = res.Get(context.TODO(), objName, metav1.GetOptions{})
		if err != nil && k8serrors.IsNotFound(err) {
			if shouldCreate == true {
				setControllerReferenceOnUnstructured(cr, obj, gvk)
				_, err = res.Create(context.TODO(), obj, metav1.CreateOptions{})
				if err != nil {
					objJSON := unstructuredToString(*obj)
					wrappedErr := errors.Wrap(err, fmt.Sprintf("Error while trying to create object using dynamic client %s. Object: %s", gvrMapping.Resource, objJSON))
					errList = append(errList, wrappedErr)
					log.Info(fmt.Sprintln(wrappedErr))
				} else {
					log.Info(fmt.Sprintf("%s %s Object Created\n", gvk.Kind, objName))
				}
			}

		} else if err != nil {
			wrappedErr := errors.Wrap(err, fmt.Sprintf("Error while trying to get object using dynamic client %s", gvrMapping.Resource))
			errList = append(errList, wrappedErr)
		} else {
			//TODO: Object found - check if we need to update
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
	}

	return nil
}

//Requeue Returns Controller Result with Requeue
func Requeue(duration time.Duration) ctrl.Result {
	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: duration,
	}
}

//DoNotRequeue Returns Controller Result without Requeue
func DoNotRequeue() ctrl.Result {
	return ctrl.Result{}
}

func (r *NVMeshBaseReconciler) getImagePullPolicy(cr *nvmeshv1.NVMesh) corev1.PullPolicy {
	if cr.Spec.Debug.ImagePullPolicyAlways {
		return corev1.PullAlways
	}

	return corev1.PullIfNotPresent
}

func (r *NVMeshBaseReconciler) imagePullSecretsFromName(secretName string) []corev1.LocalObjectReference {
	return []corev1.LocalObjectReference{{Name: secretName}}
}

func (r *NVMeshBaseReconciler) getExceleroRegistryPullSecrets() []corev1.LocalObjectReference {
	return r.imagePullSecretsFromName(exceleroRegistrySecretName)
}

func (r *NVMeshBaseReconciler) getClusterServiceAccountName(cr *nvmeshv1.NVMesh) string {
	return clusterServiceAccountName
}

func (r *NVMeshBaseReconciler) getClusterServiceAccount(cr *nvmeshv1.NVMesh) *corev1.ServiceAccount {
	sa := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{Kind: "ServiceAccount"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getClusterServiceAccountName(cr),
			Namespace: cr.GetNamespace(),
		},
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: exceleroRegistrySecretName},
		},
	}

	return sa
}

func (r *NVMeshBaseReconciler) getNVMeshClusterRoleAndRoleBinding(cr *nvmeshv1.NVMesh) (*rbac.Role, *rbac.RoleBinding) {
	role := &rbac.Role{
		TypeMeta:   metav1.TypeMeta{Kind: "Role"},
		ObjectMeta: metav1.ObjectMeta{Name: "nvmesh-cluster-role"},
		Rules: []rbac.PolicyRule{
			{
				APIGroups:     []string{"security.openshift.io"},
				Resources:     []string{"securitycontextconstraints"},
				ResourceNames: []string{operatorSCCName},
				Verbs:         []string{"use"},
			},
		},
	}

	rb := &rbac.RoleBinding{
		TypeMeta:   metav1.TypeMeta{Kind: "RoleBinding"},
		ObjectMeta: metav1.ObjectMeta{Name: "nvmesh-cluster-rb"},
		RoleRef: rbac.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     role.GetName(),
		},
		Subjects: []rbac.Subject{{
			Kind:      "ServiceAccount",
			Name:      clusterServiceAccountName,
			Namespace: cr.GetNamespace(),
		}},
	}

	return role, rb
}

func (r *NVMeshReconciler) getNVMeshSCC(cr *nvmeshv1.NVMesh) *securityv1.SecurityContextConstraints {
	scc := &securityv1.SecurityContextConstraints{
		ObjectMeta: metav1.ObjectMeta{
			Name: operatorSCCName,
			Annotations: map[string]string{
				"kubernetes.io/description": "SCC for allowing NVMesh Operator to deploy privileged containers",
			},
			Labels: r.getOperatorLabels(cr),
		},
		AllowHostDirVolumePlugin: true,
		AllowHostIPC:             true,
		AllowHostNetwork:         true,
		AllowHostPID:             true,
		AllowHostPorts:           true,
		AllowPrivilegedContainer: true,
		ReadOnlyRootFilesystem:   false,
		RunAsUser: securityv1.RunAsUserStrategyOptions{
			Type: securityv1.RunAsUserStrategyRunAsAny,
		},
		SELinuxContext: securityv1.SELinuxContextStrategyOptions{
			Type: securityv1.SELinuxStrategyRunAsAny,
		},
		FSGroup: securityv1.FSGroupStrategyOptions{
			Type: securityv1.FSGroupStrategyRunAsAny,
		},
		SupplementalGroups: securityv1.SupplementalGroupsStrategyOptions{
			Type: securityv1.SupplementalGroupsStrategyRunAsAny,
		},
		SeccompProfiles:     []string{"*"},
		AllowedCapabilities: []corev1.Capability{"*"},
		Volumes:             []securityv1.FSType{"*"},
	}

	return scc
}

func (r *NVMeshReconciler) makeSureSCCExists(cr *nvmeshv1.NVMesh) (ctrl.Result, error) {
	scc := r.getNVMeshSCC(cr)

	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: scc.GetName()}, scc)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			r.Log.Info(fmt.Sprintf("Creating SCC %s", scc.GetName()))

			err = r.Client.Create(context.TODO(), scc)
			if err != nil && !k8serrors.IsAlreadyExists(err) {
				return DoNotRequeue(), errors.Wrap(err, fmt.Sprintf("failed to create SCC %s. Error: %s", scc.GetName(), err))
			}

			return Requeue(time.Millisecond * 100), nil
		} else {
			return DoNotRequeue(), errors.Wrap(err, fmt.Sprintf("failed to get SCC %s", scc.GetName()))
		}
	}
	return DoNotRequeue(), nil
}

func (r *NVMeshReconciler) makeSureServiceAccountExists(cr *nvmeshv1.NVMesh) error {
	var objToCreate runtime.Object
	sa := r.getClusterServiceAccount(cr)

	objToCreate = sa
	err1 := r.makeSureObjectExists(cr, &objToCreate, nil)
	if err1 != nil {
		return err1
	}

	role, rb := r.getNVMeshClusterRoleAndRoleBinding(cr)

	objToCreate = role
	err2 := r.makeSureObjectExists(cr, &objToCreate, nil)
	if err2 != nil {
		return err2
	}

	objToCreate = rb
	err3 := r.makeSureObjectExists(cr, &objToCreate, nil)
	if err3 != nil {
		return err3
	}

	if r.Options.IsOpenShift {
		err4 := r.addClusterServiceAccountToSCC(cr)
		if err4 != nil {
			return errors.Wrap(err4, fmt.Sprintf("Failed to add ServiceAccount to SCC. Error: %s", err4))
		}
	}

	return nil
}

func (r *NVMeshReconciler) addClusterServiceAccountToSCC(cr *nvmeshv1.NVMesh) error {
	accountName := r.getClusterServiceAccountName(cr)
	ns := cr.GetNamespace()
	scc := &securityv1.SecurityContextConstraints{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: operatorSCCName}, scc)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to get SCC %s", operatorSCCName))
	}

	if scc.Users == nil {
		scc.Users = []string{}
	}

	userRef := "system:serviceaccount:" + ns + ":" + accountName

	for _, user := range scc.Users {
		if user == userRef {
			// if already exists - exit the function
			return nil
		}
	}

	scc.Users = append(scc.Users, userRef)

	err = r.Client.Update(context.TODO(), scc)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf(" Failed to update SCC after adding service account %s in namespace %s to SCC %s.", accountName, ns, operatorSCCName))
	}

	return nil
}

func (r *NVMeshReconciler) removeClusterServiceAccountFromSCC(cr *nvmeshv1.NVMesh) error {
	accountName := r.getClusterServiceAccountName(cr)
	ns := cr.GetNamespace()
	scc := &securityv1.SecurityContextConstraints{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: operatorSCCName}, scc)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// SCC not found, then  we don't need to remove the service account from it
			return nil
		}

		return errors.Wrap(err, fmt.Sprintf("Failed to get SCC %s", operatorSCCName))
	}

	if scc.Users != nil {
		userRef := "system:serviceaccount:" + ns + ":" + accountName

		newUsersList := make([]string, 0)
		for _, user := range scc.Users {
			if user != userRef {
				newUsersList = append(newUsersList, user)
			} else {
				r.Log.Info(fmt.Sprintf("Removing service account %s from SCC %s", userRef, scc.GetName()))
			}
		}

		scc.Users = newUsersList
	}

	if scc.Users == nil || len(scc.Users) == 0 {
		// We can delete the SCC as it has no users assigned to it now
		r.Log.Info(fmt.Sprintf("Deleting SCC %s", scc.GetName()))
		err = r.Client.Delete(context.TODO(), scc)
		if err != nil && !k8serrors.IsNotFound(err) {
			return errors.Wrap(err, fmt.Sprintf("Failed to remove SCC %s after removing service account %s in namespace %s.", operatorSCCName, ns, accountName))
		}
	} else {
		err = r.Client.Update(context.TODO(), scc)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to update SCC %s after removing service account %s in namespace %s.", operatorSCCName, ns, accountName))
		}
	}

	return nil
}

func (r *NVMeshReconciler) printAllPodsStatuses(namespace string) {
	allPodsList := &corev1.PodList{}
	err := r.Client.List(context.TODO(), allPodsList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		r.Log.Info(fmt.Sprintf("DEBUG: Failed to find all pods: %s", err))
	}

	r.Log.Info(fmt.Sprintf("DEBUG: all pods - found %d pods in namespace %s", len(allPodsList.Items), namespace))

	for _, pod := range allPodsList.Items {
		if pod.Status.ContainerStatuses != nil {
			r.Log.Info(fmt.Sprintf("DEBUG: all pods - pod: %s status: %+v", pod.GetName(), pod.Status.ContainerStatuses))
		}
	}
}
