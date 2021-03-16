package controllers

import (
	"context"
	goerrors "errors"
	"fmt"
	"strconv"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

const (
	mgmtAssetsLocation             = "resources/management/"
	mongoDBOperatorAssetsLocation  = "resources/mongodb-operator"
	mongoDBCustomResourceLocation  = "resources/mongodb-operator/custom-resource"
	mongoDBUnManagedAssetsLocation = "resources/mongodb-unmanaged"
	mgmtStatefulSetName            = "nvmesh-management"
	mgmtImageName                  = "nvmesh-management"
	mgmtGuiServiceName             = "nvmesh-management-gui"
	mgmtProtocol                   = "https"
	recursive                      = true
	nonRecursive                   = false
)

//NVMeshMgmtReconciler - Reconciler for NVMesh-Management
type NVMeshMgmtReconciler struct {
	NVMeshBaseReconciler
}

//Reconcile - Reconciles for NVMesh-Management
func (r *NVMeshMgmtReconciler) Reconcile(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	var err error

	if !cr.Spec.Management.Disabled && cr.Spec.Management.MongoDB.UseOperator {
		err = r.deployMongoDBOperator(cr, nvmeshr)
	} else {
		err = r.removeMongoDBOperator(cr, nvmeshr)
	}

	if err != nil {
		return err
	}

	if !cr.Spec.Management.Disabled && !cr.Spec.Management.MongoDB.External {
		err = r.deployMongoDBWithoutOperator(cr, nvmeshr)
	} else {
		err = r.removeMongoDBWithoutOperator(cr, nvmeshr)
	}

	if err != nil {
		return err
	}

	// Reconcile MongoDB custom resource using the unstructured client
	if !cr.Spec.Management.Disabled && !cr.Spec.Management.MongoDB.External {
		err = r.deployMongoCustomResource(cr, nvmeshr)
	} else {
		err = r.removeMongoCustomResource(cr, nvmeshr)
	}

	if err != nil {
		return err
	}

	if cr.Spec.Management.Disabled {
		err = nvmeshr.removeObjectsFromDir(cr, r, mgmtAssetsLocation, recursive)
	} else {
		err = nvmeshr.createObjectsFromDir(cr, r, mgmtAssetsLocation, recursive)
	}

	return err
}

func (r *NVMeshMgmtReconciler) removeMongoCustomResource(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	return nvmeshr.reconcileUnstructuredObjects(cr, mongoDBCustomResourceLocation, false, updateMongoDBObjects)
}

func (r *NVMeshMgmtReconciler) deployMongoCustomResource(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	return nvmeshr.reconcileUnstructuredObjects(cr, mongoDBCustomResourceLocation, true, updateMongoDBObjects)
}

func (r *NVMeshMgmtReconciler) deployMongoDBWithoutOperator(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	return nvmeshr.createObjectsFromDir(cr, r, mongoDBUnManagedAssetsLocation, nonRecursive)
}

func (r *NVMeshMgmtReconciler) removeMongoDBWithoutOperator(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	return nvmeshr.removeObjectsFromDir(cr, r, mongoDBUnManagedAssetsLocation, nonRecursive)
}

func (r *NVMeshMgmtReconciler) deployMongoDBOperator(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	return nvmeshr.createObjectsFromDir(cr, r, mongoDBOperatorAssetsLocation, nonRecursive)
}

func (r *NVMeshMgmtReconciler) removeMongoDBOperator(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	return nvmeshr.removeObjectsFromDir(cr, r, mongoDBOperatorAssetsLocation, nonRecursive)
}

func (r *NVMeshMgmtReconciler) deployManagement(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	return nvmeshr.createObjectsFromDir(cr, r, mgmtAssetsLocation, recursive)
}

func (r *NVMeshMgmtReconciler) removeManagement(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	return nvmeshr.removeObjectsFromDir(cr, r, mgmtAssetsLocation, recursive)
}

func updateMongoDBObjects(cr *nvmeshv1.NVMesh, obj *unstructured.Unstructured, gvk *schema.GroupVersionKind) {
	switch gvk.Kind {
	case "MongoDB":
		spec := obj.Object["spec"].(map[string]interface{})
		spec["replicas"] = cr.Spec.Management.MongoDB.Replicas

		// if cr.Spec.Management.MongoDB.Version != "" {
		// 	spec["version"] = cr.Spec.Management.MongoDB.Version
		// }
	}
}

// find the corresponding GVR (available in *meta.RESTMapping) for gvk
func findGVR(gvk *schema.GroupVersionKind, cfg *rest.Config) (*meta.RESTMapping, error) {

	// DiscoveryClient queries API server about the resources
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	return mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
}

//InitObject Initializes  Management objects
func (r *NVMeshMgmtReconciler) InitObject(cr *nvmeshv1.NVMesh, obj *runtime.Object) error {
	name, _ := getRunetimeObjectNameAndKind(obj)
	switch o := (*obj).(type) {
	case *appsv1.StatefulSet:
		switch name {
		case "nvmesh-management":
			return r.initiateMgmtStatefulSet(cr, o)
		case "mongo":
			return r.initiateMongoStatefulSet(cr, o)
		}
	case *v1.ConfigMap:
		switch name {
		case "nvmesh-mgmt-config":
			return r.initiateConfigMap(cr, o)
		}
	case *v1.Service:
		switch name {
		case "nvmesh-management-gui":
			return r.initiateMgmtGuiService(cr, o)
		}
	default:
		//o is unknown for us
		//log.Info(fmt.Sprintf("Object type %s not handled", o))
	}

	return nil
}

// ShouldUpdateObject Manages Management object updates
func (r *NVMeshMgmtReconciler) ShouldUpdateObject(cr *nvmeshv1.NVMesh, exp *runtime.Object, obj *runtime.Object) bool {
	name, _ := getRunetimeObjectNameAndKind(obj)
	switch o := (*obj).(type) {
	case *appsv1.StatefulSet:
		switch name {
		case "nvmesh-management":
			expectedStatefulSet := (*exp).(*appsv1.StatefulSet)
			return r.shouldUpdateStatefulSet(cr, expectedStatefulSet, o)
		}
	case *v1.ConfigMap:
		switch name {
		case "nvmesh-mgmt-config":
			var expectedConf *v1.ConfigMap = (*exp).(*v1.ConfigMap)
			shouldUpdateConf := r.shouldUpdateConfigMap(cr, expectedConf, o)
			if shouldUpdateConf == true {
				r.updateConfAndRestartMgmt(cr, expectedConf, o)
				return false
			}
		}
	case *v1.Service:
		switch name {
		case "nvmesh-management-gui":
			expectedService := (*exp).(*v1.Service)
			return r.shouldUpdateGuiService(cr, expectedService, o)
		}
	default:
		//o is unknown for us
		//log.Info(fmt.Sprintf("Object type %s not handled", o))
	}

	return false
}

func getMongoConnectionString(cr *nvmeshv1.NVMesh) string {
	return fmt.Sprintf("mongo-svc.%s.svc.cluster.local:27017", cr.GetNamespace())
}

func (r *NVMeshMgmtReconciler) initiateConfigMap(cr *nvmeshv1.NVMesh, o *v1.ConfigMap) error {
	o.Data["configVersion"] = cr.Spec.Management.Version

	loggingLevel := "DEBUG"
	useSSL := strconv.FormatBool(!cr.Spec.Management.NoSSL)

	var mongoConnectionString string
	if cr.Spec.Management.MongoDB.External {
		mongoConnectionString = cr.Spec.Management.MongoDB.Address
	} else {
		mongoConnectionString = getMongoConnectionString(cr)
	}

	statisticsCores := 5

	jsonTemplate := `{
		"loggingLevel": "%s",
		"useSSL": %s,
		"mongoConnection": {
		  "hosts": "%s"
		},
		"nvmeshMetadataMongoConnection": {
			"hosts": "%s"
		},
		"statisticsMongoConnection": {
		  "hosts": "%s"
		},
		"statisticsCores": %d
	  }`

	o.Data["config"] = fmt.Sprintf(
		jsonTemplate,
		loggingLevel,
		useSSL,
		mongoConnectionString,
		mongoConnectionString,
		mongoConnectionString,
		statisticsCores)
	return nil
}

func (r *NVMeshMgmtReconciler) initiateMgmtStatefulSet(cr *nvmeshv1.NVMesh, o *appsv1.StatefulSet) error {

	if cr.Spec.Management.Version == "" {
		return goerrors.New("Missing Management Version (NVMesh.Spec.Management.Version)")
	}

	o.Spec.Template.Spec.Containers[0].Image = getMgmtImageFromResource(cr)
	o.Spec.Replicas = &cr.Spec.Management.Replicas

	return nil
}

func getMongoForNVMeshImageName() string {
	return "registry.excelero.com/nvmesh-mongo-instance:" + coreImageVersionTag
}

func (r *NVMeshMgmtReconciler) initiateMongoStatefulSet(cr *nvmeshv1.NVMesh, o *appsv1.StatefulSet) error {
	o.Spec.Template.Spec.Containers[0].Image = getMongoForNVMeshImageName()
	return nil
}

func (r *NVMeshMgmtReconciler) initiateMgmtGuiService(cr *nvmeshv1.NVMesh, svc *v1.Service) error {
	if cr.Spec.Management.ExternalIPs != nil {
		svc.Spec.ExternalIPs = cr.Spec.Management.ExternalIPs
	}

	return nil
}

func getMgmtImageFromResource(cr *nvmeshv1.NVMesh) string {
	imageRegistry := cr.Spec.Management.ImageRegistry
	return imageRegistry + "/" + mgmtImageName + ":" + cr.Spec.Management.Version
}

func (r *NVMeshMgmtReconciler) shouldUpdateStatefulSet(cr *nvmeshv1.NVMesh, expected *appsv1.StatefulSet, ss *appsv1.StatefulSet) bool {
	log := r.Log.WithValues("method", "shouldUpdateStatefulSet")

	expectedVersion := expected.Spec.Template.Spec.Containers[0].Image
	foundVersion := ss.Spec.Template.Spec.Containers[0].Image
	if expectedVersion != foundVersion {
		log.Info(fmt.Sprintf("found mgmt version missmatch - expected: %s found: %s\n", expectedVersion, foundVersion))
		return true
	}

	expectedReplicas := *expected.Spec.Replicas
	foundReplicas := *ss.Spec.Replicas
	if *(expected.Spec.Replicas) != *(ss.Spec.Replicas) {
		log.Info(fmt.Sprintf("Management replica number needs to be updated expected: %d found: %d\n", expectedReplicas, foundReplicas))
		return true
	}

	return false
}

func isStringArraysEqualElements(a []string, b []string) bool {
	dict := make(map[string]bool)

	// add all a items to dict
	for _, key := range a {
		dict[key] = false
	}

	for _, key := range b {
		// if item in dict, mark it as found
		if _, ok := dict[key]; ok {
			dict[key] = true
		} else {
			// item is not expected
			return false
		}
	}

	for _, val := range dict {
		if val == false {
			return false
		}
	}

	return true
}

func (r *NVMeshMgmtReconciler) shouldUpdateGuiService(cr *nvmeshv1.NVMesh, expected *v1.Service, svc *v1.Service) bool {
	// We first copy the already assigned clusterIP otherwise update fails since ClusterIP: "" is an invalid  value for an update
	expected.Spec.ClusterIP = svc.Spec.ClusterIP

	if expected.Spec.ExternalIPs != nil {
		return !isStringArraysEqualElements(expected.Spec.ExternalIPs, svc.Spec.ExternalIPs)
	}

	return false
}

func (r *NVMeshMgmtReconciler) shouldUpdateConfigMap(cr *nvmeshv1.NVMesh, expected *v1.ConfigMap, conf *v1.ConfigMap) bool {
	log := r.Log.WithValues("method", "shouldUpdateConfigMap")

	expectedConfig := expected.Data["config"]
	foundConfig := conf.Data["config"]
	if expectedConfig != foundConfig {
		log.Info(fmt.Sprintf("found mgmt config missmatch - expected: %s\n found: %s\n", expectedConfig, foundConfig))
		return true
	}

	expectedConfVersion := expected.Data["configVersion"]
	foundConfVersion := conf.Data["configVersion"]
	if expectedConfig != foundConfig {
		log.Info(fmt.Sprintf("found mgmt config version missmatch - expected: %s found: %s\n", expectedConfVersion, foundConfVersion))
		return true
	}
	return false
}

func (r *NVMeshMgmtReconciler) updateConfAndRestartMgmt(cr *nvmeshv1.NVMesh, expected *v1.ConfigMap, conf *v1.ConfigMap) error {
	log := r.Log.WithValues("method", "updateConfAndRestartMgmt")

	log.Info("Updating ConfigMap\n")

	err := r.Client.Update(context.TODO(), expected)
	if err != nil {
		log.Error(err, "Error while updating object")
		return err
	}

	return r.restartManagement(cr.GetNamespace())
}

func (r *NVMeshMgmtReconciler) restartManagement(namespace string) error {
	log := r.Log.WithValues("method", "restartManagement")

	log.Info("restarting Managements\n")
	var ss appsv1.StatefulSet

	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: mgmtStatefulSetName, Namespace: namespace}, &ss)
	if err != nil {
		log.Error(err, "Error while getting object")
		return err
	}

	var originalValue int32 = *ss.Spec.Replicas
	var newValue int32 = 0
	ss.Spec.Replicas = &newValue

	err = r.Client.Update(context.TODO(), &ss)
	if err != nil {
		log.Error(err, "Error while updating object")
		return err
	}

	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: mgmtStatefulSetName, Namespace: namespace}, &ss)
	if err != nil {
		log.Error(err, "Error while getting object")
		return err
	}

	ss.Spec.Replicas = &originalValue
	updateAttempts := 5
	var updated bool = false
	for updated == false && updateAttempts > 0 {
		updateAttempts = updateAttempts - 1
		err = r.Client.Update(context.TODO(), &ss)
		if err == nil {
			updated = true
		}
	}

	return err
}
