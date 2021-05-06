package controllers

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	nvmeshCoreAssestLocation   = "resources/nvmesh-core"
	coreUserspaceDaemonSetName = "nvmesh-mcs-agent"
	targetDriverDaemonSetName  = "nvmesh-target"
	clientDriverDaemonSetName  = "nvmesh-client"
	driverContainerImageName   = "nvmesh-driver-container"
	defaultFileServerAddress   = "https://repo.excelero.com/nvmesh/operator_binaries"
	envVarNVMeshVersion        = "NVMESH_VERSION"
)

//NVMeshCoreReconciler - Reconciles NVMesh Core Component
type NVMeshCoreReconciler struct {
	NVMeshBaseReconciler
}

//Reconcile NVMesh Core Component
func (r *NVMeshCoreReconciler) Reconcile(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	if !cr.Spec.Core.Disabled {
		return r.deployCore(cr, nvmeshr)
	}

	return r.removeCore(cr, nvmeshr)
}

func (r *NVMeshCoreReconciler) removeCore(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	return nvmeshr.removeObjectsFromDir(cr, r, nvmeshCoreAssestLocation, true)
}

func (r *NVMeshCoreReconciler) deployCore(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	return nvmeshr.createObjectsFromDir(cr, r, nvmeshCoreAssestLocation, true)
}

//InitObject Initialize objects in Core
func (r *NVMeshCoreReconciler) InitObject(cr *nvmeshv1.NVMesh, obj *runtime.Object) error {
	//name, kind := getRunetimeObjectNameAndKind(obj)
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

//ShouldUpdateObject Manages update objects in Core
func (r *NVMeshCoreReconciler) ShouldUpdateObject(cr *nvmeshv1.NVMesh, exp *runtime.Object, obj *runtime.Object) bool {
	name, _ := getRunetimeObjectNameAndKind(obj)
	switch o := (*obj).(type) {
	case *appsv1.DaemonSet:
		expDS := (*exp).(*appsv1.DaemonSet)
		switch name {
		case coreUserspaceDaemonSetName:
			fallthrough
		case targetDriverDaemonSetName:
			fallthrough
		case clientDriverDaemonSetName:
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

		for _, env := range c.Env {
			if env.Name == envVarNVMeshVersion {

				if env.Value != cr.Spec.Core.Version {
					log.Info(fmt.Sprintf("Core version requires update on DaemonSet %s Container %s expected: %s found: %s", ds.ObjectMeta.Name, c.Name, cr.Spec.Core.Version, env.Value))
					return true
				}
			}
		}
	}

	return false
}

func (r *NVMeshCoreReconciler) addVolumeAndMountToContainer(volumeName string, mountPath string, podSpec *v1.PodSpec, container *v1.Container) {
	volume := v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			HostPath: &v1.HostPathVolumeSource{
				Path: mountPath,
			},
		},
	}
	podSpec.Volumes = append(podSpec.Volumes, volume)

	mount := v1.VolumeMount{
		Name:      volumeName,
		MountPath: mountPath,
	}
	container.VolumeMounts = append(container.VolumeMounts, mount)
}

func (r *NVMeshCoreReconciler) addTomaIBLibMounts(podSpec *v1.PodSpec, tomaContainer *v1.Container) {
	volumeMounts := map[string]string{
		"etc-libibverbs": "/etc/libibverbs.d/",
	}

	for volumeName, volumePath := range volumeMounts {
		r.addVolumeAndMountToContainer(volumeName, volumePath, podSpec, tomaContainer)
	}
}

func (r *NVMeshCoreReconciler) initDaemonSets(cr *nvmeshv1.NVMesh, ds *appsv1.DaemonSet) error {
	var imageName string
	podSpec := &ds.Spec.Template.Spec

	for i, _ := range podSpec.Containers {
		container := &podSpec.Containers[i]
		switch container.Name {
		case "mcs":
			imageName = "nvmesh-mcs"
		case "agent":
			imageName = "nvmesh-mcs"
		case "toma":
			imageName = "nvmesh-toma"

			if !cr.Spec.Core.TCPOnly {
				r.addTomaIBLibMounts(podSpec, container)
			}
		case "tracer":
			imageName = "nvmesh-tracer"
		case "driver-container":
			imageName = "nvmesh-driver-container"
			if !cr.Spec.Core.TCPOnly {
				r.addVolumeAndMountToContainer("etc-infiniband", "/etc/infiniband", podSpec, container)
			}
		}

		podSpec.Containers[i].Image = cr.Spec.Core.ImageRegistry + "/" + imageName + ":" + cr.Spec.Core.ImageVersionTag
		podSpec.Containers[i].ImagePullPolicy = r.getImagePullPolicy(cr)
		r.setEnvVariableValues(cr, container)
	}

	return nil
}

func (r *NVMeshCoreReconciler) setEnvVariableValues(cr *nvmeshv1.NVMesh, container *corev1.Container) {
	for i, _ := range container.Env {
		env := &container.Env[i]
		if env.Name == "NVMESH_VERSION" {
			env.Value = cr.Spec.Core.Version
		}
	}

	r.addKeepRunningAfterFailureEnvVar(cr, container)
}

func (r *NVMeshCoreReconciler) configStringToDict(conf string) map[string]string {
	var configDict map[string]string = make(map[string]string, 0)

	lines := strings.Split(conf, "\n")
	for _, line := range lines {
		lineParts := strings.Split(line, "=")
		key := lineParts[0]
		value := strings.Join(lineParts[1:], "=")

		// add to map
		configDict[key] = value
	}

	return configDict
}

func (r *NVMeshCoreReconciler) configDictToString(configDict map[string]string) string {
	var lines []string = make([]string, 0)

	// get sorted list of keys
	sortedKeys := make([]string, 0, len(configDict))
	for k := range configDict {
		sortedKeys = append(sortedKeys, k)
	}

	sort.Strings(sortedKeys)

	// create conf string
	for _, key := range sortedKeys {
		value := configDict[key]
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

func nvmeshConfWrapWithQuotes(value string) string {
	return fmt.Sprintf("\"%s\"", value)
}

func (r *NVMeshCoreReconciler) initCoreConfigMap(cr *nvmeshv1.NVMesh, cm *v1.ConfigMap) error {
	fileServerOptions := nvmeshv1.OperatorFileServerSpec{}
	if cr.Spec.Operator.FileServer != nil {
		fileServerOptions = *cr.Spec.Operator.FileServer
	}

	if fileServerOptions.Address != "" {
		cm.Data["fileServer.address"] = fileServerOptions.Address
	} else {
		cm.Data["fileServer.address"] = defaultFileServerAddress
	}

	cm.Data["fileServer.skipCheckCertificate"] = strconv.FormatBool(fileServerOptions.SkipCheckCertificate)

	configDict := r.configStringToDict(cm.Data["nvmesh.conf"])

	managementServers := r.getMgmtServersConnectionString(cr)
	// Wrap value with double quotes
	configDict["MANAGEMENT_SERVERS"] = nvmeshConfWrapWithQuotes(managementServers)

	if cr.Spec.Core.TCPOnly {
		configDict["IPV4_ONLY"] = nvmeshConfWrapWithQuotes("Yes")
		configDict["TCP_ENABLED"] = nvmeshConfWrapWithQuotes("Yes")
		configDict["CONFIGURED_NICS"] = nvmeshConfWrapWithQuotes(cr.Spec.Core.ConfiguredNICs)
	}

	if cr.Spec.Core.AzureOptimized {
		configDict["CLOUD_OPTIMIZED"] = nvmeshConfWrapWithQuotes("Yes")

		// Reduces calls to S.M.A.R.T because of poor performance of NVMe SMART
		configDict["TOMA_CLOUD_MODE"] = nvmeshConfWrapWithQuotes("Yes")
		configDict["AGENT_CLOUD_MODE"] = nvmeshConfWrapWithQuotes("Yes")
	}

	cm.Data["nvmesh.conf"] = r.configDictToString(configDict)
	return nil
}
