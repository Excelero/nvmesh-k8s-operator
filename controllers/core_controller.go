package controllers

import (
	"fmt"
	"strconv"
	"strings"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	NVMeshCoreAssestLocation   = "resources/nvmesh-core"
	CoreUserspaceDaemonSetName = "nvmesh-mcs-agent"
	TargetDriverDaemonSetName  = "nvmesh-target-driver-container"
	ClientDriverDaemonSetName  = "nvmesh-client-driver-container"
	DriverContainerImageName   = "nvmesh-driver-container"
	FileServerAddress          = "https://repo.excelero.com/nvmesh/operator_binaries"
	CoreImageVersionTag        = "0.7.0-2"
)

type NVMeshCoreReconciler struct {
	NVMeshBaseReconciler
}

func (r *NVMeshCoreReconciler) Reconcile(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	if !cr.Spec.Core.Disabled {
		return r.DeployCore(cr, nvmeshr)
	} else {
		return r.RemoveCore(cr, nvmeshr)
	}
}

func (r *NVMeshCoreReconciler) RemoveCore(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	return nvmeshr.RemoveObjectsFromDir(cr, r, NVMeshCoreAssestLocation, true)
}

func (r *NVMeshCoreReconciler) DeployCore(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	return nvmeshr.CreateObjectsFromDir(cr, r, NVMeshCoreAssestLocation, true)
}

func (r *NVMeshCoreReconciler) InitObject(cr *nvmeshv1.NVMesh, obj *runtime.Object) error {
	//name, kind := GetRunetimeObjectNameAndKind(obj)
	switch o := (*obj).(type) {
	case *appsv1.DaemonSet:
		err := r.initDaemonSets(cr, o)
		return err
	case *v1.ConfigMap:
		err := r.initCoreConfigMap(cr, o)
		return err
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
	case *v1.ConfigMap:
		// shouldUpdateCoreConfigMap
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

func (r *NVMeshCoreReconciler) initDaemonSets(cr *nvmeshv1.NVMesh, ds *appsv1.DaemonSet) error {
	var imageName string
	for i, c := range ds.Spec.Template.Spec.Containers {
		switch c.Name {
		case "mcs":
			imageName = "nvmesh-mcs"
		case "agent":
			imageName = "nvmesh-mcs"
		case "toma":
			imageName = "nvmesh-toma"
		case "tracer":
			imageName = "nvmesh-tracer"
		case "driver-container":
			imageName = "nvmesh-driver-container"
		}

		ds.Spec.Template.Spec.Containers[i].Image = cr.Spec.Core.ImageRegistry + "/" + imageName + ":" + CoreImageVersionTag
		ds.Spec.Template.Spec.Containers[i].ImagePullPolicy = r.GetGlobalImagePullPolicy()
	}

	return nil
}

func (r *NVMeshCoreReconciler) configStringToDict(conf string) map[string]string {
	var configDict map[string]string = make(map[string]string, 0)

	lines := strings.Split(conf, "\n")
	for _, line := range lines {
		line_parts := strings.Split(line, "=")
		key := line_parts[0]
		value := strings.Join(line_parts[1:], "=")

		// add to map
		configDict[key] = value
	}

	return configDict
}

func (r *NVMeshCoreReconciler) configDictToString(configDict map[string]string) string {
	var lines []string = make([]string, 0)

	for key, value := range configDict {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	return strings.Join(lines, "\n")
}

func (r *NVMeshCoreReconciler) getMgmtServersConnectionString(cr *nvmeshv1.NVMesh) string {
	var servers []string

	replicas := int(cr.Spec.Management.Replicas)
	for i := 0; i < replicas; i++ {
		server := fmt.Sprintf("nvmesh-management-%d.nvmesh-management-ws.%s.svc.cluster.local:4001", i, cr.GetNamespace())
		servers = append(servers, server)
	}

	return strings.Join(servers, ",")
}

func (r *NVMeshCoreReconciler) initCoreConfigMap(cr *nvmeshv1.NVMesh, cm *v1.ConfigMap) error {
	if cr.Spec.Operator.FileServer.Address != "" {
		cm.Data["fileServer.address"] = cr.Spec.Operator.FileServer.Address
	} else {
		cm.Data["fileServer.address"] = FileServerAddress
	}

	cm.Data["fileServer.skipCheckCertificate"] = strconv.FormatBool(cr.Spec.Operator.FileServer.SkipCheckCertificate)
	cm.Data["nvmesh.version"] = cr.Spec.Core.Version

	configDict := r.configStringToDict(cm.Data["nvmesh.conf"])

	managementServers := r.getMgmtServersConnectionString(cr)
	// Wrap value with double quotes
	configDict["MANAGEMENT_SERVERS"] = fmt.Sprintf("\"%s\"", managementServers)

	cm.Data["nvmesh.conf"] = r.configDictToString(configDict)
	return nil
}
