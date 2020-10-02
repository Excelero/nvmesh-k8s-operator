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
	MgmtAssetsLocation             = "resources/management/"
	MongoDBOperatorAssetsLocation  = "resources/mongodb-operator"
	MongoDBCustomResourceLocation  = "resources/mongodb-operator/custom-resource"
	MongoDBUnManagedAssetsLocation = "resources/mongodb-unmanaged"
	MgmtStatefulSetName            = "nvmesh-management"
	MgmtImageName                  = "nvmesh-management"
	MgmtGuiServiceName             = "nvmesh-management-gui"
	MgmtProtocol                   = "https"
)

type NVMeshMgmtReconciler struct {
	NVMeshBaseReconciler
}

func (r *NVMeshMgmtReconciler) Reconcile(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	var err error
	recursive := true
	nonRecursive := false

	if !cr.Spec.Management.Disabled && cr.Spec.Management.MongoDB.UseOperator {
		// Deploy MongoDB Operator
		err = nvmeshr.CreateObjectsFromDir(cr, r, MongoDBOperatorAssetsLocation, nonRecursive)
		if err != nil {
			return err
		}
	} else {
		// Remove MongoDB Operator
		err = nvmeshr.RemoveObjectsFromDir(cr, r, MongoDBOperatorAssetsLocation, nonRecursive)
		if err != nil {
			return err
		}
	}

	if !cr.Spec.Management.Disabled && !cr.Spec.Management.MongoDB.External {
		// Deploy MongoDB Without Operator
		err = nvmeshr.CreateObjectsFromDir(cr, r, MongoDBUnManagedAssetsLocation, nonRecursive)
		if err != nil {
			return err
		}
	} else {
		// Remove MongoDB Without Operator
		err = nvmeshr.RemoveObjectsFromDir(cr, r, MongoDBUnManagedAssetsLocation, nonRecursive)
		if err != nil {
			return err
		}
	}

	// Reconcile MongoDB custom resource using the unstructured client
	shouldDeployMongo := !cr.Spec.Management.Disabled && !cr.Spec.Management.MongoDB.External
	err = nvmeshr.ReconcileUnstructuredObjects(cr, MongoDBCustomResourceLocation, shouldDeployMongo, updateMongoDBObjects)
	if err != nil {
		return err
	}

	if cr.Spec.Management.Disabled {
		err = nvmeshr.RemoveObjectsFromDir(cr, r, MgmtAssetsLocation, recursive)
	} else {
		err = nvmeshr.CreateObjectsFromDir(cr, r, MgmtAssetsLocation, recursive)
	}

	return err
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

func (r *NVMeshMgmtReconciler) InitObject(cr *nvmeshv1.NVMesh, obj *runtime.Object) error {
	name, _ := GetRunetimeObjectNameAndKind(obj)
	switch o := (*obj).(type) {
	case *appsv1.StatefulSet:
		switch name {
		case "nvmesh-management":
			return r.initiateMgmtStatefulSet(cr, o)
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

func (r *NVMeshMgmtReconciler) ShouldUpdateObject(cr *nvmeshv1.NVMesh, exp *runtime.Object, obj *runtime.Object) bool {
	name, _ := GetRunetimeObjectNameAndKind(obj)
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

func (r *NVMeshMgmtReconciler) initiateConfigMap(cr *nvmeshv1.NVMesh, o *v1.ConfigMap) error {
	o.Data["configVersion"] = cr.Spec.Management.Version

	loggingLevel := "DEBUG"
	useSSL := strconv.FormatBool(!cr.Spec.Management.NoSSL)

	var mongoConnectionString string
	if cr.Spec.Management.MongoDB.External {
		mongoConnectionString = cr.Spec.Management.MongoDB.Address
	} else {
		mongoConnectionString = "mongo-svc.default.svc.cluster.local:27017"
	}

	statisticsCores := 5

	jsonTemplate := `{
		"loggingLevel": "%s",
		"useSSL": %s,
		"mongoConnection": {
		  "hosts": "%s"
		},
		"statisticsMongoConnection": {
		  "hosts": "%s"
		},
		"statisticsCores": %d
	  }`

	o.Data["config"] = fmt.Sprintf(jsonTemplate, loggingLevel, useSSL, mongoConnectionString, mongoConnectionString, statisticsCores)
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

func (r *NVMeshMgmtReconciler) initiateMgmtGuiService(cr *nvmeshv1.NVMesh, svc *v1.Service) error {
	if cr.Spec.Management.ExternalIPs != nil {
		svc.Spec.ExternalIPs = cr.Spec.Management.ExternalIPs
	}

	return nil
}

func getMgmtImageFromResource(cr *nvmeshv1.NVMesh) string {
	imageRegistry := cr.Spec.Management.ImageRegistry
	return imageRegistry + "/" + MgmtImageName + ":" + cr.Spec.Management.Version
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

	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: MgmtStatefulSetName, Namespace: namespace}, &ss)
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

	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: MgmtStatefulSetName, Namespace: namespace}, &ss)
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
