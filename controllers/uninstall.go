package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	errors "github.com/pkg/errors"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	uninstallJobNamePrefix          = "n-uninstall-"
	clearDbJobName                  = "nvmesh-clear-db-job"
	uninstallJobImageName           = "nvmesh-uninstall-job"
	nvmeshClusterServiceAccountName = "nvmesh-cluster"
)

var uninstallAction = nvmeshv1.ClusterAction{Name: "uninstall"}

func (r *NVMeshReconciler) uninstallCluster(nvmeshCluster *nvmeshv1.NVMesh) (ctrl.Result, error) {
	log := r.Log.WithValues("method", "uninstallCluster", "component", "Finalizer")

	if nvmeshCluster.Spec.Operator.SkipUninstall {
		// Skip uninstall procedure
		log.Info("Spec.Operator.SkipUninstall: true - Skipping Uninstall")
		return DoNotRequeue(), nil
	}

	var result ctrl.Result
	var err error

	var stages []Task = []Task{
		{
			"removeAllWorkloads",
			func(cr *nvmeshv1.NVMesh) (ctrl.Result, error) {
				return r.removeAllWorkloadsExceptMongo(cr)
			},
		},
		{
			"waitForWorkloadsToFinish",
			func(cr *nvmeshv1.NVMesh) (ctrl.Result, error) {
				return r.waitForWorkloadsToFinish(cr)
			},
		},
		{
			"clearDB",
			func(cr *nvmeshv1.NVMesh) (ctrl.Result, error) {
				return DoNotRequeue(), r.runClearDbJob(nvmeshCluster)
			},
		},
		{
			"waitForClearDBToFinish",
			func(cr *nvmeshv1.NVMesh) (ctrl.Result, error) {
				return r.waitForClearDBToFinish(cr)
			},
		},
		{
			"removeMongo",
			func(cr *nvmeshv1.NVMesh) (ctrl.Result, error) {
				return DoNotRequeue(), r.removeMongo(cr)
			},
		},
		{
			"uninstallClusterNodes",
			func(cr *nvmeshv1.NVMesh) (ctrl.Result, error) {
				return r.uninstallClusterNodes(cr)
			},
		},
		{
			"deleteClusterServiceAccount",
			func(cr *nvmeshv1.NVMesh) (ctrl.Result, error) {
				return r.deleteClusterServiceAccount(cr)
			},
		},
	}

	for _, stage := range stages {
		stageName := stage.Name
		if !r.isTaskFinished(nvmeshCluster, uninstallAction, stageName) {
			log.Info(fmt.Sprintf("Uninstall stage: %s", stageName))
			r.setTaskStarted(nvmeshCluster, uninstallAction, stageName)
			result, err = stage.Run(nvmeshCluster)

			if result.Requeue {
				log.Info(fmt.Sprintf("Uninstall stage %s not finished, will retry", stageName))
				return result, nil
			}

			if err != nil {
				return DoNotRequeue(), err
			}
			r.setTaskFinished(nvmeshCluster, uninstallAction, stageName)
			log.Info(fmt.Sprintf("Uninstall stage %s done", stageName))
		}
	}

	r.setActionComplete(nvmeshCluster, uninstallAction)

	nvmeshCluster.Spec.CSI.Disabled = true
	nvmeshCluster.Spec.Management.Disabled = true
	nvmeshCluster.Spec.Core.Disabled = true

	return DoNotRequeue(), nil
}

func (r *NVMeshReconciler) removeMongo(cr *nvmeshv1.NVMesh) error {
	mgmt := NVMeshMgmtReconciler(*r)
	if err := mgmt.removeMongoDBOperator(cr, r); err != nil {
		return err
	}

	if err := mgmt.removeMongoCustomResource(cr, r); err != nil {
		return err
	}

	if err := mgmt.removeMongoDBWithoutOperator(cr, r); err != nil {
		return err
	}

	return nil
}

func (r *NVMeshReconciler) removeAllWorkloadsExceptMongo(cr *nvmeshv1.NVMesh) (ctrl.Result, error) {
	core := NVMeshCoreReconciler(*r)
	if err := core.removeCore(cr, r); err != nil {
		return DoNotRequeue(), err
	}

	csi := NVMeshCSIReconciler(*r)
	if err := csi.removeCSI(cr, r); err != nil {
		return DoNotRequeue(), err
	}

	mgmt := NVMeshMgmtReconciler(*r)
	if err := mgmt.removeManagement(cr, r); err != nil {
		return DoNotRequeue(), err
	}

	return DoNotRequeue(), nil
}

func (r *NVMeshReconciler) getJobAsJSON(jobName string, namespace string) ([]byte, error) {
	jobKey := client.ObjectKey{Name: jobName, Namespace: namespace}
	job := &batchv1.Job{}
	err := r.Client.Get(context.TODO(), jobKey, job)
	if err != nil {
		r.Log.Info(fmt.Sprintf("DEBUG: Failed to get job %s. Error: %s", jobName, err))
	}
	bytes, err := json.MarshalIndent(job, "", "    ")

	return bytes, err
}

func (r *NVMeshReconciler) waitForClearDBToFinish(cr *nvmeshv1.NVMesh) (ctrl.Result, error) {
	log := r.Log.WithValues("method", "clearDB", "component", "Finalizer")
	result, err := r.waitForJobToFinish(cr, clearDbJobName)

	if result.Requeue {
		return result, nil
	}

	if err != nil {
		log.Info(fmt.Sprintf("WARNING: Unable to clear MongoDB Database for NVMesh Cluster %s on namespace %s", cr.GetName(), cr.GetNamespace()))
	}

	if err := r.deleteJob(cr.GetNamespace(), clearDbJobName); err != nil {
		return DoNotRequeue(), err
	}

	return DoNotRequeue(), nil
}

func (r *NVMeshReconciler) waitForWorkloadsToFinish(cr *nvmeshv1.NVMesh) (ctrl.Result, error) {
	log := r.Log.WithValues("method", "waitForWorkloadsToFinish", "component", "Finalizer")
	podList := &corev1.PodList{}

	require, err := labels.NewRequirement("nvmesh.excelero.com/component", selection.In, []string{"client", "target", "mcs-agent", "csi-node-driver"})
	if err != nil {
		return DoNotRequeue(), err
	}

	matchLabels := client.MatchingLabelsSelector{Selector: labels.NewSelector().Add(*require)}

	log.Info("checking if workload pods were removed")
	if err := r.Client.List(context.TODO(), podList, &matchLabels); err != nil {
		return DoNotRequeue(), err
	}

	if len(podList.Items) > 0 {
		r.Log.Info(fmt.Sprintf("Waiting for all workloads to finish. Found %d Pods", len(podList.Items)))
		return Requeue(time.Second), nil
	}

	return DoNotRequeue(), nil
}

func (r *NVMeshReconciler) runUninstallJobs(cr *nvmeshv1.NVMesh, nodeList []corev1.Node) error {
	var err error
	for _, node := range nodeList {
		nodeName := node.GetName()
		err := r.uninstallNode(cr, nodeName)
		if err != nil {
			r.Log.Info(fmt.Sprintf("Uninstall Failed on node %s. Error: %s", nodeName, err))
		}
	}

	return err
}

func (r *NVMeshReconciler) getUninstallJob(cr *nvmeshv1.NVMesh, nodeName string) *batchv1.Job {
	jobName := r.getUninstallJobName(nodeName)
	job := r.getNewJob(cr, jobName, r.getUninstallJobImageName(cr))

	podSpec := &job.Spec.Template.Spec
	r.addHostPathMount(podSpec, 0, "/opt")
	r.addHostPathMount(podSpec, 0, "/etc/opt")
	r.addHostPathMount(podSpec, 0, "/var/log")

	container := &podSpec.Containers[0]
	container.Env = []corev1.EnvVar{
		{Name: "KEEP_DOWNLOAD_CACHE", Value: "false"},
	}

	setContainerAsPrivileged(container)

	podSpec.NodeName = nodeName

	return job
}

func (r *NVMeshReconciler) getNodesWithLabel(key string, value string) (*corev1.NodeList, error) {
	nodeSelector := client.MatchingLabels{
		key: value,
	}

	nodeList := &corev1.NodeList{}
	err := r.Client.List(context.TODO(), nodeList, nodeSelector)
	if err != nil && k8serrors.IsNotFound(err) {
		return nodeList, nil
	}

	return nodeList, err
}

func (r *NVMeshReconciler) getAllMgmtLabelledNodes(cr *nvmeshv1.NVMesh) (*corev1.NodeList, error) {
	nodes, err := r.getNodesWithLabel(nvmeshMgmtLabelKey, "")
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

func (r *NVMeshReconciler) getAllNVMeshClusterNodes(cr *nvmeshv1.NVMesh) (map[string]corev1.Node, error) {

	clients, err := r.getNodesWithLabel(nvmeshClientLabelKey, "")
	if err != nil {
		return nil, err
	}
	targets, err := r.getNodesWithLabel(nvmeshTargetLabelKey, "")
	if err != nil {
		return nil, err
	}

	nodesSet := make(map[string]corev1.Node)

	for _, c := range clients.Items {
		nodesSet[c.GetName()] = c
	}

	for _, t := range targets.Items {
		if _, ok := nodesSet[t.GetName()]; !ok {
			nodesSet[t.GetName()] = t
		}
	}

	return nodesSet, err
}

func (r *NVMeshReconciler) getUninstallJobName(nodeName string) string {
	return uninstallJobNamePrefix + sanitizeString(nodeName)
}

func (r *NVMeshReconciler) waitForUninstallCompletion(cr *nvmeshv1.NVMesh, nodeList []corev1.Node) (ctrl.Result, error) {
	for _, node := range nodeList {
		jobName := r.getUninstallJobName(node.GetName())
		result, err := r.waitForJobToFinish(cr, jobName)
		if result.Requeue || err != nil {
			return result, err
		}

		r.Log.Info(fmt.Sprintf("job %s finished", jobName))
	}

	return DoNotRequeue(), nil
}

func (r *NVMeshReconciler) deleteUninstallJobs(cr *nvmeshv1.NVMesh, nodeList []corev1.Node) error {

	for _, node := range nodeList {
		jobName := r.getUninstallJobName(node.GetName())
		r.Log.Info(fmt.Sprintf("Deleting job %s\n", jobName))
		err := r.deleteJob(cr.GetNamespace(), jobName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *NVMeshReconciler) runClearDbJob(cr *nvmeshv1.NVMesh) error {
	mongoImage := r.getCoreFullImageName(cr, mongoInstanceImageName)
	job := r.getNewJob(cr, clearDbJobName, mongoImage)
	backoffLimit := int32(1)
	job.Spec.BackoffLimit = &backoffLimit
	container := &job.Spec.Template.Spec.Containers[0]

	container.Command = []string{"mongo"}
	mongoConnString := getMongoConnectionString(cr) + "/management"
	container.Args = []string{mongoConnString, "--eval", "db.dropDatabase()"}

	job.Spec.Template.Spec.ImagePullSecrets = r.getExceleroRegistryPullSecrets()

	err := r.Client.Create(context.TODO(), job)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "Failed to create Clear DB job")
	}

	return nil
}

func (r *NVMeshReconciler) nodeSetToList(nodeSet map[string]corev1.Node) []corev1.Node {
	var nodeList []corev1.Node
	for _, n := range nodeSet {
		nodeList = append(nodeList, n)
	}

	return nodeList
}

func (r *NVMeshReconciler) uninstallClusterNodes(nvmeshCluster *nvmeshv1.NVMesh) (ctrl.Result, error) {
	nodeSet, err := r.getAllNVMeshClusterNodes(nvmeshCluster)
	if err != nil {
		return DoNotRequeue(), errors.Wrap(err, fmt.Sprintf("Failed to list all of the nodes in NVMesh Cluster %s", nvmeshCluster.GetName()))
	}

	nodeList := r.nodeSetToList(nodeSet)

	r.Log.Info(fmt.Sprintf("Uninstalling %d nodes", len(nodeList)))
	for _, n := range nodeSet {
		fmt.Printf(n.GetName())
	}

	if err := r.runUninstallJobs(nvmeshCluster, nodeList); err != nil {
		return DoNotRequeue(), err
	}

	result, err := r.waitForUninstallCompletion(nvmeshCluster, nodeList)
	if err != nil || result.Requeue {
		return result, err
	}

	if err := r.deleteUninstallJobs(nvmeshCluster, nodeList); err != nil {
		return DoNotRequeue(), err
	}

	return DoNotRequeue(), nil
}

func (r *NVMeshReconciler) uninstallNode(cr *nvmeshv1.NVMesh, nodeName string) error {
	r.Log.Info(fmt.Sprintf("Running Uninstall Job on Node %s", nodeName))

	job := r.getUninstallJob(cr, nodeName)
	labels := map[string]string{"job-name": job.ObjectMeta.Name}
	job.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	job.Spec.Template.ObjectMeta.Labels = labels
	job.Spec.Template.Spec.NodeSelector = client.MatchingLabels{"kubernetes.io/hostname": nodeName}

	err := r.Client.Create(context.TODO(), job)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return errors.Wrap(err, fmt.Sprintf("Failed to run UninstallJob on node %s. Error: %s", nodeName, err))
	}

	return nil
}

func (r *NVMeshReconciler) deleteClusterServiceAccount(cr *nvmeshv1.NVMesh) (ctrl.Result, error) {
	sa := r.getClusterServiceAccount(cr)
	role, rb := r.getNVMeshClusterRoleAndRoleBinding(cr)
	saRuntime := runtime.Object(sa)
	roleRuntime := runtime.Object(role)
	rbRuntime := runtime.Object(rb)

	if err := r.makeSureObjectRemoved(cr, &saRuntime, nil); err != nil {
		return Requeue(time.Second), err
	}

	if err := r.makeSureObjectRemoved(cr, &roleRuntime, nil); err != nil {
		return Requeue(time.Second), err
	}

	if err := r.makeSureObjectRemoved(cr, &rbRuntime, nil); err != nil {
		return Requeue(time.Second), err
	}

	r.removeClusterServiceAccountFromSCC(cr)
	return ctrl.Result{}, nil
}

func (r *NVMeshReconciler) getUninstallJobImageName(cr *nvmeshv1.NVMesh) string {
	return r.getCoreFullImageName(cr, uninstallJobImageName)
}
