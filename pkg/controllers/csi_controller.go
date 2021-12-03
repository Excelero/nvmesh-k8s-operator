package controllers

import (
	goerrors "errors"
	"fmt"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/pkg/api/v1"
	reflectutils "excelero.com/nvmesh-k8s-operator/pkg/reflectutils"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	csiAssetsLocation        = "resources/csi/"
	csiDaemonSetName         = "nvmesh-csi-node-driver"
	csiStatefulSetName       = "nvmesh-csi-controller"
	csiDriverImageName       = "nvmesh-csi-driver"
	csiDriverDefaultRegistry = "excelero"
	csiServiceAccountName    = "nvmesh-csi"
)

//NVMeshCSIReconciler is a Reconciler for CSI
type NVMeshCSIReconciler struct {
	NVMeshBaseReconciler
}

//Reconcile Reconciles CSI
func (r *NVMeshCSIReconciler) Reconcile(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) (ctrl.Result, error) {
	if !cr.Spec.CSI.Disabled {
		return DoNotRequeue(), r.deployCSI(cr, nvmeshr)
	}

	return DoNotRequeue(), r.removeCSI(cr, nvmeshr)
}

func (r *NVMeshCSIReconciler) deployCSI(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	return nvmeshr.createObjectsFromDir(cr, r, csiAssetsLocation, true)
}

func (r *NVMeshCSIReconciler) removeCSI(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	return nvmeshr.removeObjectsFromDir(cr, r, csiAssetsLocation, true)
}

//InitObject Initializes CSI Objects
func (r *NVMeshCSIReconciler) InitObject(cr *nvmeshv1.NVMesh, obj client.Object) error {
	name := obj.GetName()

	switch o := (obj).(type) {
	case *appsv1.StatefulSet:
		switch name {
		case "nvmesh-csi-controller":
			err := r.initCSIControllerStatefulSet(cr, o)
			return err
		}
	case *appsv1.DaemonSet:
		switch name {
		case "nvmesh-csi-node-driver":
			return r.initCSINodeDriverDaemonSet(cr, o)
		}
	case *appsv1.Deployment:
	case *v1.ServiceAccount:
	case *v1.ConfigMap:
		return initCSIConfigMap(cr, o)
	case *rbac.RoleBinding:
		return r.initRoleBinding(cr, o)
	case *rbac.ClusterRoleBinding:
		return r.initClusterRoleBinding(cr, o)
	default:
		//o is unknown for us
		//log.Info(fmt.Sprintf("Object type %s not handled", o))
	}

	return nil
}

//ShouldUpdateObject Manages CIS object updates
func (r *NVMeshCSIReconciler) ShouldUpdateObject(cr *nvmeshv1.NVMesh, exp client.Object, found client.Object) bool {
	name := found.GetName()

	switch o := (found).(type) {
	case *appsv1.StatefulSet:
		switch name {
		case "nvmesh-csi-controller":
			expected := (exp).(*appsv1.StatefulSet)
			return r.shouldUpdateCSIControllerStatefulSet(cr, expected, o)
		}
	case *appsv1.DaemonSet:
		switch name {
		case "nvmesh-csi-node-driver":
			expected := (exp).(*appsv1.DaemonSet)
			return r.shouldUpdateCSINodeDriverDaemonSet(cr, expected, o)
		}

	case *appsv1.Deployment:
	case *v1.ServiceAccount:
	case *v1.ConfigMap:
	default:
		//o is unknown for us
		//log.Info(fmt.Sprintf("Object type %s not handled", o))
	}

	return false
}

func (r *NVMeshCSIReconciler) getCSIServiceAccountName(cr *nvmeshv1.NVMesh) string {
	return csiServiceAccountName
}

func (r *NVMeshCSIReconciler) initCSINodeDriverDaemonSet(cr *nvmeshv1.NVMesh, ds *appsv1.DaemonSet) error {
	if cr.Spec.CSI.Version == "" {
		return goerrors.New("Missing NVMesh CSI Driver Version (NVMesh.Spec.CSI.Version)")
	}

	ds.Spec.Template.Spec.ServiceAccountName = r.getCSIServiceAccountName(cr)

	ds.Spec.Template.Spec.Containers[0].Image = getCSIFullImageName(cr)
	ds.Spec.Template.Spec.Containers[0].ImagePullPolicy = r.getImagePullPolicy(cr)

	return nil
}

func (r *NVMeshCSIReconciler) initCSIControllerStatefulSet(cr *nvmeshv1.NVMesh, ss *appsv1.StatefulSet) error {
	if cr.Spec.CSI.Version == "" {
		return goerrors.New("Missing NVMesh CSI Driver Version (NVMesh.Spec.CSI.Version)")
	}

	ss.Spec.Template.Spec.ServiceAccountName = r.getCSIServiceAccountName(cr)

	ss.Spec.Template.Spec.Containers[0].Image = getCSIFullImageName(cr)
	ss.Spec.Template.Spec.Containers[0].ImagePullPolicy = r.getImagePullPolicy(cr)

	// set replicas from CustomResource
	ss.Spec.Replicas = &cr.Spec.CSI.ControllerReplicas

	return nil
}

func initCSIConfigMap(cr *nvmeshv1.NVMesh, conf *v1.ConfigMap) error {
	conf.Data["management.protocol"] = mgmtProtocol
	conf.Data["management.servers"] = mgmtGuiServiceName + "." + cr.GetNamespace() + ".svc.cluster.local:4000"
	return nil
}

func (r *NVMeshCSIReconciler) initClusterRoleBinding(cr *nvmeshv1.NVMesh, crb *rbac.ClusterRoleBinding) error {
	ns := cr.GetNamespace()
	for i := range crb.Subjects {
		crb.Subjects[i].Namespace = ns
	}

	crb.Subjects[0].Name = r.getCSIServiceAccountName(cr)
	return nil
}

func (r *NVMeshCSIReconciler) initRoleBinding(cr *nvmeshv1.NVMesh, rb *rbac.RoleBinding) error {
	ns := cr.GetNamespace()
	for i := range rb.Subjects {
		rb.Subjects[i].Namespace = ns
	}

	rb.Subjects[0].Name = r.getCSIServiceAccountName(cr)
	return nil
}

func getCSIFullImageName(cr *nvmeshv1.NVMesh) string {
	registry := csiDriverDefaultRegistry
	if cr.Spec.CSI.ImageRegistry != "" {
		registry = cr.Spec.CSI.ImageRegistry
	}

	version := cr.Spec.CSI.Version
	imageName := registry + "/" + csiDriverImageName + ":" + version

	return imageName
}

func (r *NVMeshCSIReconciler) shouldUpdateCSINodeDriverDaemonSet(cr *nvmeshv1.NVMesh, expected *appsv1.DaemonSet, ds *appsv1.DaemonSet) bool {
	log := r.Log.WithName("shouldUpdateCSINodeDriverDaemonSet")

	fields := []string{
		"Spec.Template.Spec.Containers[0].Image",
		"Spec.Template.Spec.Containers[0].ImagePullPolicy",
	}

	err, result := reflectutils.CompareFieldsInTwoObjects(expected, ds, fields)

	if err != nil {
		log.Error(err, "Error comparing CSI Node Driver DaemonSet")
	}

	if !result.Equals {
		log.Info(fmt.Sprintf("CSI Node Driver field %s needs to be updated expected: %+v found: %+v\n", result.Path, result.Value1, result.Value2))
		return true
	}

	return false
}

func (r *NVMeshCSIReconciler) shouldUpdateCSIControllerStatefulSet(cr *nvmeshv1.NVMesh, expected *appsv1.StatefulSet, ss *appsv1.StatefulSet) bool {
	log := r.Log.WithName("shouldUpdateCSIControllerStatefulSet")
	fields := []string{
		"Spec.Template.Spec.Containers[0].Image",
		"Spec.Replicas",
	}

	err, result := reflectutils.CompareFieldsInTwoObjects(expected, ss, fields)

	if err != nil {
		log.Error(err, "Error comparing CSI Controller StatefulSet")
	}

	if !result.Equals {
		log.Info(fmt.Sprintf("CSI Controller field %s needs to be updated expected: %+v found: %+v\n", result.Path, result.Value1, result.Value2))
		return true
	}

	return false
}
