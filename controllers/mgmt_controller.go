package controllers

import (
	goerrors "errors"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MgmtAssetsLocation    = "config/samples/management/"
	MongoDBAssestLocation = "config/samples/mongodb"
	MgmtStatefulSetName   = "nvmesh-management"
	MgmtImageName         = "docker.excelero.com/nvmesh-management"
)

type NVMeshMgmtReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *NVMeshMgmtReconciler) Reconcile(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	var err error

	if cr.Spec.Management.DeployMongo {
		err = r.DeployMongoDB(cr, nvmeshr)
		if err != nil {
			return err
		}
	} else {
		err = r.RemoveMongoDB(cr, nvmeshr)
		if err != nil {
			return err
		}
	}

	if cr.Spec.Management.Deploy {
		err = nvmeshr.CreateObjectsFromDir(cr, r, MgmtAssetsLocation)
	} else {
		err = nvmeshr.RemoveObjectsFromDir(cr, r, MgmtAssetsLocation)
	}

	return err
}

func (r *NVMeshMgmtReconciler) DeployMongoDB(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	// First deploy the crd
	crdFile := MongoDBAssestLocation + "/mongodb.com_mongodb_crd.yaml"
	err := nvmeshr.ReconcileYamlObjectsFromFile(cr, crdFile, r, false)
	if err != nil {
		return err
	}

	// Then deploy all the rest
	err = nvmeshr.CreateObjectsFromDir(cr, r, MongoDBAssestLocation)
	return err
}

func (r *NVMeshMgmtReconciler) RemoveMongoDB(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	err := nvmeshr.RemoveObjectsFromDir(cr, r, MongoDBAssestLocation)
	return err
}

func (r *NVMeshMgmtReconciler) InitObject(cr *nvmeshv1.NVMesh, obj *runtime.Object) error {
	name, _ := GetRunetimeObjectNameAndKind(obj)
	switch o := (*obj).(type) {
	case *appsv1.StatefulSet:
		switch name {
		case "nvmesh-management":
			return initiateMgmtStatefulSet(cr, o)
		}
	case *v1.Service:
		switch name {
		case "nvmesh-management-svc-0":
			return initiateServiceMcs(cr, o)
		}
	case *v1.ConfigMap:
		// TODO: we will probably need to parse the json value edit it and encode as json again

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
			return shouldUpdateMgmtStatefulSet(cr, o)
		}
	default:
		//o is unknown for us
		//log.Info(fmt.Sprintf("Object type %s not handled", o))
	}

	return false
}

func initiateServiceMcs(cr *nvmeshv1.NVMesh, o *v1.Service) error {
	// TODO: we need to template the service and duplicate it as replica size times
	return nil
}

func initiateMgmtStatefulSet(cr *nvmeshv1.NVMesh, o *appsv1.StatefulSet) error {

	if cr.Spec.Management.Version == "" {
		return goerrors.New("Missing Management Version (NVMesh.Spec.Management.Version)")
	}

	o.Spec.Template.Spec.Containers[0].Image = getMgmtImageFromVersion(cr.Spec.Management.Version)

	//TODO: set still use configMap or set values directly into the daemonset ?
	return nil
}

func getMgmtImageFromVersion(version string) string {
	return MgmtImageName + ":" + version
}

func shouldUpdateMgmtStatefulSet(cr *nvmeshv1.NVMesh, ss *appsv1.StatefulSet) bool {

	if getMgmtImageFromVersion(cr.Spec.Management.Version) != ss.Spec.Template.Spec.Containers[0].Image {
		return true
	}

	return false
}
