package controllers

import (
	"context"
	goerrors "errors"
	"fmt"
	"strings"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MgmtAssetsLocation    = "config/samples/management/"
	MongoDBAssestLocation = "config/samples/mongodb"
	MgmtStatefulSetName   = "nvmesh-management"
	MgmtImageName         = "nvmesh-management"
	MgmtGuiServiceName    = "nvmesh-management-gui"
	MgmtProtocol          = "https"
)

type NVMeshMgmtReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *NVMeshMgmtReconciler) Reconcile(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	var err error

	if cr.Spec.Management.Deploy {
		err = nvmeshr.CreateObjectsFromDir(cr, r, MgmtAssetsLocation)
	} else {
		err = nvmeshr.RemoveObjectsFromDir(cr, r, MgmtAssetsLocation)
	}

	return err
}

func (r *NVMeshMgmtReconciler) InitObject(cr *nvmeshv1.NVMesh, obj *runtime.Object) error {
	name, _ := GetRunetimeObjectNameAndKind(obj)
	switch o := (*obj).(type) {
	case *appsv1.StatefulSet:
		switch name {
		case "nvmesh-management":
			return r.initiateMgmtStatefulSet(cr, o)
		}
	case *v1.Service:
		switch name {
		case "nvmesh-management-svc-0":
			return r.initiateServiceMcs(cr, o)
		}
	case *v1.ConfigMap:
		return r.initiateConfigMap(cr, o)
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
		var expectedConf *v1.ConfigMap = (*exp).(*v1.ConfigMap)
		shouldUpdateConf := r.shouldUpdateConfigMap(cr, expectedConf, o)
		if shouldUpdateConf == true {
			r.updateConfAndRestartMgmt(cr, expectedConf, o)
			return false
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
	useSSL := "true"
	mongoConnectionString := cr.Spec.Management.MongoAddress
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

func (r *NVMeshMgmtReconciler) initiateServiceMcs(cr *nvmeshv1.NVMesh, o *v1.Service) error {
	// TODO: we need to template the service and duplicate it as replica size times
	// Check the option of routing using <statefulset-instance>.<statefulset-name>.<ns>.svc.cluster.local:<port>
	return nil
}

func (r *NVMeshMgmtReconciler) initiateMgmtStatefulSet(cr *nvmeshv1.NVMesh, o *appsv1.StatefulSet) error {

	if cr.Spec.Management.Version == "" {
		return goerrors.New("Missing Management Version (NVMesh.Spec.Management.Version)")
	}

	o.Spec.Template.Spec.Containers[0].Image = getMgmtImageFromResource(cr)
	return nil
}

func getMgmtImageFromResource(cr *nvmeshv1.NVMesh) string {
	imageRegistry := cr.Spec.Management.ImageRegistry
	if imageRegistry != "" && !strings.HasSuffix(imageRegistry, "/") {
		imageRegistry = imageRegistry + "/"
	}

	return imageRegistry + MgmtImageName + ":" + cr.Spec.Management.Version
}

func (r *NVMeshMgmtReconciler) shouldUpdateStatefulSet(cr *nvmeshv1.NVMesh, expected *appsv1.StatefulSet, ss *appsv1.StatefulSet) bool {

	expectedVersion := expected.Spec.Template.Spec.Containers[0].Image
	foundVersion := ss.Spec.Template.Spec.Containers[0].Image
	if expectedVersion != foundVersion {
		fmt.Printf("found mgmt version missmatch - expected: %s found: %s\n", expectedVersion, foundVersion)
		return true
	}

	expectedReplicas := *expected.Spec.Replicas
	foundReplicas := *ss.Spec.Replicas
	if *(expected.Spec.Replicas) != *(ss.Spec.Replicas) {
		fmt.Printf("Management replica number needs to be updated expected: %d found: %d\n", expectedReplicas, foundReplicas)
		return true
	}

	return false
}

func (r *NVMeshMgmtReconciler) shouldUpdateConfigMap(cr *nvmeshv1.NVMesh, expected *v1.ConfigMap, conf *v1.ConfigMap) bool {
	expectedConfig := expected.Data["config"]
	foundConfig := conf.Data["config"]
	if expectedConfig != foundConfig {
		fmt.Printf("found mgmt config missmatch - expected: %s\n found: %s\n", expectedConfig, foundConfig)
		return true
	}

	expectedConfVersion := expected.Data["configVersion"]
	foundConfVersion := conf.Data["configVersion"]
	if expectedConfig != foundConfig {
		fmt.Printf("found mgmt config version missmatch - expected: %s found: %s\n", expectedConfVersion, foundConfVersion)
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
