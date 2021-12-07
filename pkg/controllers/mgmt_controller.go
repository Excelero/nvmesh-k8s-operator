package controllers

import (
	"context"
	"encoding/json"
	goerrors "errors"
	"time"

	errors "github.com/pkg/errors"

	"fmt"
	"strconv"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/pkg/api/v1"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	mongoclient "excelero.com/nvmesh-k8s-operator/pkg/mongoclient"
	reflectutils "excelero.com/nvmesh-k8s-operator/pkg/reflectutils"
	mongotopology "go.mongodb.org/mongo-driver/x/mongo/driver/topology"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	reconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	mgmtAssetsLocation                = "resources/management/"
	mongoDBAssetsLocation             = "resources/mongodb"
	mgmtStatefulSetName               = "nvmesh-management"
	mgmtImageName                     = "nvmesh-management"
	mongoInstanceImageName            = "nvmesh-mongo-instance"
	mgmtGuiServiceName                = "nvmesh-management-gui"
	mgmtProtocol                      = "https"
	mgmtInitDbJobName                 = "mgmt-init-db"
	recursive                         = true
	nonRecursive                      = false
	SettingsKeyAutoFromatDrives       = "hidden.autoFormatDrive"
	SettingsKeyAutoEvictMissingDrives = "hidden.autoEvictMissingDrive"
)

//NVMeshMgmtReconciler - Reconciler for NVMesh-Management
type NVMeshMgmtReconciler struct {
	NVMeshBaseReconciler
}

//Reconcile - Reconciles for NVMesh-Management
func (r *NVMeshMgmtReconciler) Reconcile(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) (reconcile.Result, error) {
	var err error
	defaultRequeue := Requeue(time.Second)

	if !cr.Spec.Management.Disabled && !cr.Spec.Management.MongoDB.External {
		err = r.deployMongoDB(cr, nvmeshr)
	} else {
		err = r.removeMongoDB(cr, nvmeshr)
	}

	if err != nil {
		return defaultRequeue, err
	}

	if err != nil {
		return defaultRequeue, err
	}

	if cr.Spec.Management.Disabled {
		err = nvmeshr.removeObjectsFromDir(cr, r, mgmtAssetsLocation, recursive)
	} else {
		err = r.handleDBManipulations(cr)
		if err == mongo.ErrNoDocuments {
			// First run - Init DB
			r.Log.Info("No globalSettings document found in MongoDB - Running initDB")
			err = r.runMgmtInitDBJob(cr)
			return Requeue(time.Second * 3), err
		} else if err != nil {
			_, isServerSelectionError := err.(mongotopology.ServerSelectionError)
			if isServerSelectionError {
				// If failed to connect to mongo return immediately and requeue
				r.Log.Info("Failed to connect to MongoDB.. Please make sure that you have a valid PersistentVolume created for Mongo and that MongoDB Pod is running")
				return Requeue(time.Second * 5), nil
			} else {
				// return any other unexpected error
				return defaultRequeue, err
			}
		}
		err = nvmeshr.createObjectsFromDir(cr, r, mgmtAssetsLocation, recursive)
	}

	return DoNotRequeue(), err
}

func (r *NVMeshMgmtReconciler) handleDBManipulations(cr *nvmeshv1.NVMesh) error {
	log := r.Log.WithName("handleDBManipulations")

	filter := bson.D{}
	projection := bson.D{{"hidden", 1}}
	var err error

	type HiddenSettings struct {
		AutoEvictMissingDrive bool `bson:"autoEvictMissingDrive"`
		AutoFormatDrive       bool `bson:"autoFormatDrive"`
		IsElectDisabled       bool `bson:"isElectDisabled"`
	}

	type FindResult struct {
		ID     primitive.ObjectID `bson:"_id"` // omitempty to protect against zeroed _id insertion
		Hidden HiddenSettings     `bson:"hidden"`
	}

	var result FindResult
	mongoURI := getMongoURI(cr)

	// Used for development when we don't have access to the Pod's ClusterIP where mongo is listening
	if r.Options.Development {
		mongoURI = "mongodb://localhost:27017"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return errors.Wrap(err, "Failed to connect to MongoDB")
	}

	defer func() {
		if asyncErr := client.Disconnect(context.TODO()); asyncErr != nil {
			log.Error(asyncErr, "Error disconnecting from MongoDB")
		}
	}()

	err = mongoclient.FindOne(client, "globalSettings", filter, projection, &result)

	if err != nil {
		log.V(VerboseLogging).Info(fmt.Sprintf("Mongo FindOne failed: %s", err))
		return err
	}

	// We can now definitely delete any initDBJob that is left
	r.deleteJob(cr.GetNamespace(), mgmtInitDbJobName)

	if cr.Spec.Management.DisableAutoFormatDrives != !result.Hidden.AutoFormatDrive || cr.Spec.Management.DisableAutoEvictMissingDrives != !result.Hidden.AutoEvictMissingDrive {
		log.Info(fmt.Sprintf("Updating AutoFormatDrives=%t and AutoEvictMissingDrives=%t in the DB", !cr.Spec.Management.DisableAutoFormatDrives, !cr.Spec.Management.DisableAutoEvictMissingDrives))

		update := bson.D{{"$set", bson.D{
			{"hidden.autoEvictMissingDrive", !cr.Spec.Management.DisableAutoEvictMissingDrives},
			{"hidden.autoFormatDrive", !cr.Spec.Management.DisableAutoFormatDrives}}}}
		err = mongoclient.UpdateOne(client, "globalSettings", filter, &update)
		if err != nil {
			return errors.Wrap(err, "Failed to update autoEvictDrives & autoFormatDrives in MongoDB")
		}

		// we attempt to restart the management server but we ingore errors since it is possible that the it was not deployed yet
		_ = r.restartManagement(cr.GetNamespace())
	}

	return nil
}

func (r *NVMeshMgmtReconciler) deployMongoDB(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	return nvmeshr.createObjectsFromDir(cr, r, mongoDBAssetsLocation, nonRecursive)
}

func (r *NVMeshMgmtReconciler) removeMongoDB(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	return nvmeshr.removeObjectsFromDir(cr, r, mongoDBAssetsLocation, nonRecursive)
}

func (r *NVMeshMgmtReconciler) deployManagement(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	return nvmeshr.createObjectsFromDir(cr, r, mgmtAssetsLocation, recursive)
}

func (r *NVMeshMgmtReconciler) removeManagement(cr *nvmeshv1.NVMesh, nvmeshr *NVMeshReconciler) error {
	return nvmeshr.removeObjectsFromDir(cr, r, mgmtAssetsLocation, recursive)
}

func (r *NVMeshMgmtReconciler) runMgmtInitDBJob(cr *nvmeshv1.NVMesh) error {
	imageName := getMgmtImageFromResource(cr)
	job := r.getNewJob(cr, mgmtInitDbJobName, imageName)
	backoffLimit := int32(1)
	job.Spec.BackoffLimit = &backoffLimit
	container := &job.Spec.Template.Spec.Containers[0]

	container.Command = []string{"mongo"}
	mongoConnString := getMongoConnectionString(cr) + "/management"
	container.Args = []string{mongoConnString, "/opt/NVMesh/management/initDB.js"}

	err := r.Client.Create(context.TODO(), job)
	if err == nil {
		r.Log.Info(fmt.Sprintf("Job %s created", mgmtInitDbJobName))
	} else if !k8serrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "Failed to create iniDB job")
	}

	return nil
}

//InitObject Initializes  Management objects
func (r *NVMeshMgmtReconciler) InitObject(cr *nvmeshv1.NVMesh, obj client.Object) error {
	name := obj.GetName()
	switch o := (obj).(type) {
	case *appsv1.StatefulSet:
		switch name {
		case "nvmesh-management":
			return r.initMgmtStatefulSet(cr, o)
		case "mongo":
			return r.initMongoStatefulSet(cr, o)
		}
	case *v1.ConfigMap:
		switch name {
		case "nvmesh-mgmt-config":
			return r.initConfigMap(cr, o)
		}
	case *v1.Service:
		switch name {
		case "nvmesh-management-gui":
			return r.initMgmtGuiService(cr, o)
		}
	default:
		//o is unknown for us
		//log.Info(fmt.Sprintf("Object type %s not handled", o))
	}

	return nil
}

// ShouldUpdateObject Manages Management object updates
func (r *NVMeshMgmtReconciler) ShouldUpdateObject(cr *nvmeshv1.NVMesh, exp client.Object, obj client.Object) bool {
	name := obj.GetName()
	switch o := (obj).(type) {
	case *appsv1.StatefulSet:
		switch name {
		case "nvmesh-management":
			expectedStatefulSet := (exp).(*appsv1.StatefulSet)
			return r.shouldUpdateManagementStatefulSet(cr, expectedStatefulSet, o)
		case "mongo":
			expectedStatefulSet := (exp).(*appsv1.StatefulSet)
			return r.shouldUpdateMongoStatefulSet(cr, expectedStatefulSet, o)
		}
	case *v1.ConfigMap:
		switch name {
		case "nvmesh-mgmt-config":
			var expectedConf *v1.ConfigMap = (exp).(*v1.ConfigMap)
			shouldUpdateConf := r.shouldUpdateConfigMap(cr, expectedConf, o)
			if shouldUpdateConf == true {
				err := r.updateConfAndRestartMgmt(cr, expectedConf, o)
				if err != nil {
					r.Log.Info(fmt.Sprintf("Failed to Update Management Config. Error: %s", err))
				}
				return false
			}
		}
	case *v1.Service:
		switch name {
		case "nvmesh-management-gui":
			expectedService := (exp).(*v1.Service)
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

func getMongoURI(cr *nvmeshv1.NVMesh) string {
	return fmt.Sprintf("mongodb://%s", getMongoConnectionString(cr))
}

func (r *NVMeshMgmtReconciler) initConfigMap(cr *nvmeshv1.NVMesh, o *v1.ConfigMap) error {
	o.Data["configVersion"] = cr.Spec.Management.Version

	var mongoConnectionString string
	if cr.Spec.Management.MongoDB.External {
		mongoConnectionString = cr.Spec.Management.MongoDB.Address
	} else {
		mongoConnectionString = getMongoConnectionString(cr)
	}

	mongoConnection := map[string]string{
		"hosts": mongoConnectionString,
	}

	useSSL := strconv.FormatBool(!cr.Spec.Management.NoSSL)

	conf := make(map[string]interface{})
	conf["loggingLevel"] = "DEBUG"
	conf["statisticsCores"] = 5
	conf["useSSL"] = useSSL
	conf["mongoConnection"] = mongoConnection
	conf["nvmeshMetadataMongoConnection"] = mongoConnection
	conf["statisticsMongoConnection"] = mongoConnection

	conf["exceleroEmail"] = "customer.stats+OpenShift@excelero.com"
	conf["SMTP"] = r.getSMTPConfig(cr)

	jsonString, err := json.MarshalIndent(conf, "", "    ")
	if err != nil {
		return err
	}
	o.Data["config"] = string(jsonString)

	return nil
}

func (r *NVMeshMgmtReconciler) getSMTPConfig(cr *nvmeshv1.NVMesh) map[string]interface{} {
	smtpJson := []byte(`{
		"host": "smtp.gmail.com",
		"port": 587,
		"secure": true,
		"authRequired": true,
		"username": "app@excelero.com",
		"password": "Tom@2021",
		"useDefault": true
	}`)

	var smtpConf map[string]interface{}
	json.Unmarshal(smtpJson, &smtpConf)
	return smtpConf
}

func (r *NVMeshMgmtReconciler) initMgmtStatefulSet(cr *nvmeshv1.NVMesh, o *appsv1.StatefulSet) error {

	if cr.Spec.Management.Version == "" {
		return goerrors.New("Missing Management Version (NVMesh.Spec.Management.Version)")
	}

	o.Spec.Template.Spec.Containers[0].Image = getMgmtImageFromResource(cr)
	o.Spec.Replicas = &cr.Spec.Management.Replicas
	r.addKeepRunningAfterFailureEnvVar(cr, &o.Spec.Template.Spec.Containers[0])

	overrideVolumeClaimFields(&o.Spec.VolumeClaimTemplates[0].Spec, &cr.Spec.Management.BackupsVolumeClaim)

	return nil
}

func overrideVolumeClaimFields(target *v1.PersistentVolumeClaimSpec, source *v1.PersistentVolumeClaimSpec) {
	if source.StorageClassName != nil {
		target.StorageClassName = source.StorageClassName
	}

	if source.Selector != nil {
		target.Selector = source.Selector
	}

	if source.Resources.Requests != nil {
		target.Resources.Requests = source.Resources.Requests
	}

	if source.Resources.Limits != nil {
		target.Resources.Limits = source.Resources.Limits
	}
}

func (r *NVMeshMgmtReconciler) initMongoStatefulSet(cr *nvmeshv1.NVMesh, o *appsv1.StatefulSet) error {
	o.Spec.Template.Spec.Containers[0].Image = r.getCoreFullImageName(cr, mongoInstanceImageName)

	overrideVolumeClaimFields(&o.Spec.VolumeClaimTemplates[0].Spec, &cr.Spec.Management.MongoDB.DataVolumeClaim)

	return nil
}

func (r *NVMeshMgmtReconciler) initMgmtGuiService(cr *nvmeshv1.NVMesh, svc *v1.Service) error {
	if cr.Spec.Management.ExternalIPs != nil {
		svc.Spec.ExternalIPs = cr.Spec.Management.ExternalIPs
	}

	return nil
}

func getMgmtImageFromResource(cr *nvmeshv1.NVMesh) string {
	imageRegistry := cr.Spec.Management.ImageRegistry
	return imageRegistry + "/" + mgmtImageName + ":" + cr.Spec.Management.Version
}

func (r *NVMeshMgmtReconciler) shouldUpdateManagementStatefulSet(cr *nvmeshv1.NVMesh, expected *appsv1.StatefulSet, ss *appsv1.StatefulSet) bool {
	log := r.Log.WithName("shouldUpdateManagementStatefulSet")

	fields := []string{
		"Spec.Template.Spec.Containers[0].Image",
		"Spec.Replicas",
	}

	err, result := reflectutils.CompareFieldsInTwoObjects(expected, ss, fields)

	if err != nil {
		log.Error(err, "Error comparing Management StatefulSet")
	}

	if !result.Equals {
		log.Info(fmt.Sprintf("Management StatefulSet field %s needs to be updated expected: %+v found: %+v\n", result.Path, result.Value1, result.Value2))
		return true
	}

	return false
}

func (r *NVMeshMgmtReconciler) shouldUpdateMongoStatefulSet(cr *nvmeshv1.NVMesh, expected *appsv1.StatefulSet, ss *appsv1.StatefulSet) bool {
	log := r.Log.WithName("shouldUpdateMongoStatefulSet")

	fields := []string{
		"Spec.Template.Spec.Containers[0].Image",
	}

	err, result := reflectutils.CompareFieldsInTwoObjects(expected, ss, fields)

	if err != nil {
		log.Error(err, "Error comparing mongo StatefulSet")
	}

	if !result.Equals {
		log.Info(fmt.Sprintf("mongo StatefulSet field %s needs to be updated expected: %+v found: %+v\n", result.Path, result.Value1, result.Value2))
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
	log := r.Log.WithName("shouldUpdateConfigMap")

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
	log := r.Log.WithName("updateConfAndRestartMgmt")

	log.Info("Updating ConfigMap\n")

	err := r.Client.Update(context.TODO(), expected)
	if err != nil {
		return err
	}

	return r.restartManagement(cr.GetNamespace())
}

func (r *NVMeshMgmtReconciler) restartManagement(namespace string) error {
	return r.restartStatefulSet(namespace, mgmtStatefulSetName)
}
