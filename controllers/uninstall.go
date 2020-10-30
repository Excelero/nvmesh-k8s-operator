package controllers

import (
	"context"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"time"

	errors "github.com/pkg/errors"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	uninstallJobNamePrefix          = "nvmesh-uninstall-"
	clearDbJobName                  = "nvmesh-clear-db-job"
	uninstallJobImage               = "registry.excelero.com/nvmesh-uninstall-job:0.7.0-2"
	nvmeshClusterServiceAccountName = "nvmesh-cluster"
)

var UninstallAction = nvmeshv1.ClusterAction{Name: "uninstall"}

type TaskFunc func(cr *nvmeshv1.NVMesh) (ctrl.Result, error)

type Task struct {
	Name string
	Run  TaskFunc
}

func (r *NVMeshReconciler) UninstallCluster(nvmeshCluster *nvmeshv1.NVMesh) (ctrl.Result, error) {
	log := r.Log.WithValues("method", "UninstallCluster", "component", "Finalizer")

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
		if !r.isTaskFinished(nvmeshCluster, UninstallAction, stageName) {
			log.Info(fmt.Sprintf("Uninstall stage: %s", stageName))
			r.setTaskStarted(nvmeshCluster, UninstallAction, stageName)
			result, err = stage.Run(nvmeshCluster)

			if result.Requeue {
				log.Info(fmt.Sprintf("Uninstall stage %s not finished, will retry", stageName))
				return result, nil
			}

			if err != nil {
				return DoNotRequeue(), err
			}
			r.setTaskFinished(nvmeshCluster, UninstallAction, stageName)
			log.Info(fmt.Sprintf("Uninstall stage %s done", stageName))
		}
	}

	r.setActionComplete(nvmeshCluster, UninstallAction)

	nvmeshCluster.Spec.CSI.Disabled = true
	nvmeshCluster.Spec.Management.Disabled = true
	nvmeshCluster.Spec.Core.Disabled = true

	return DoNotRequeue(), nil
}

func (r *NVMeshReconciler) removeMongo(cr *nvmeshv1.NVMesh) error {
	mgmt := NVMeshMgmtReconciler(*r)
	if err := mgmt.RemoveMongoDBOperator(cr, r); err != nil {
		return err
	}

	if err := mgmt.RemoveMongoCustomResource(cr, r); err != nil {
		return err
	}

	if err := mgmt.RemoveMongoDBWithoutOperator(cr, r); err != nil {
		return err
	}

	return nil
}

func (r *NVMeshReconciler) removeAllWorkloadsExceptMongo(cr *nvmeshv1.NVMesh) (ctrl.Result, error) {
	core := NVMeshCoreReconciler(*r)
	if err := core.RemoveCore(cr, r); err != nil {
		return DoNotRequeue(), err
	}

	csi := NVMeshCSIReconciler(*r)
	if err := csi.RemoveCSI(cr, r); err != nil {
		return DoNotRequeue(), err
	}

	mgmt := NVMeshMgmtReconciler(*r)
	if err := mgmt.RemoveManagement(cr, r); err != nil {
		return DoNotRequeue(), err
	}

	return DoNotRequeue(), nil
}

func (r *NVMeshReconciler) debug_jobs(cr *nvmeshv1.NVMesh) {
	log := r.Log.WithValues("method", "debug_cluster")
	log.Info("debug_jobs -->")
	// 1. check the job pods and it's status
	podList, err := r.getJobPods(cr.GetNamespace(), clearDbJobName)
	if err != nil && !k8serrors.IsNotFound(err) {
		log.Info(fmt.Sprintf("DEBUG: Failed to find pods from job %s for deletion", clearDbJobName))
	}

	log.Info(fmt.Sprintf("found %d pods for job %s", len(podList.Items), clearDbJobName))

	if len(podList.Items) == 0 {
		log.Info(fmt.Sprintf("Found no pods from job %s", clearDbJobName))
	}

	for _, pod := range podList.Items {
		log.Info(fmt.Sprintf("DEBUG: JOB PODS - pod: %s status: %+v", pod.GetName(), pod.Status.ContainerStatuses))
	}

	// print the job
	if false {
		jobKey := client.ObjectKey{Name: clearDbJobName, Namespace: cr.GetNamespace()}
		job := &batchv1.Job{}
		err = r.Client.Get(context.TODO(), jobKey, job)
		if err != nil {
			log.Info(fmt.Sprintf("DEBUG: Failed to get job %s. Error: %s", clearDbJobName, err))
		}

		bytes, err := json.MarshalIndent(job, "", "    ")
		if err != nil {
			log.Info(fmt.Sprintf("Error printing json %s", err))
		} else {
			log.Info(fmt.Sprintf("DEBUG: Job %s:\n%s", clearDbJobName, string(bytes)))
		}
	}

	//2. list all pods and their statuses (in the current namespace)
	if false {
		allPodsList := &corev1.PodList{}
		err = r.Client.List(context.TODO(), allPodsList, &client.ListOptions{Namespace: cr.GetNamespace()})
		if err != nil {
			log.Info(fmt.Sprintf("DEBUG: Failed to find all pods: %s", err))
		}

		log.Info(fmt.Sprintf("DEBUG: all pods - found %d pods in namespace %s", len(allPodsList.Items), cr.GetNamespace()))

		for _, pod := range allPodsList.Items {
			if pod.Status.ContainerStatuses != nil {
				log.Info(fmt.Sprintf("DEBUG: all pods - pod: %s status: %+v", pod.GetName(), pod.Status.ContainerStatuses))
			}
		}
	}

	if false {
		// 3. check if the secrets appear
		// make sure RBAC rules allow this
		secret := &corev1.Secret{}
		secret_name := exceleroRegistrySecretName
		secretKey := client.ObjectKey{Name: exceleroRegistrySecretName, Namespace: cr.GetNamespace()}
		err = r.Client.Get(context.TODO(), secretKey, secret)
		if err != nil {
			log.Info(fmt.Sprintf("DEBUG: Failed to find secret: %s error: %s", secret_name, err))
		}

		log.Info(fmt.Sprintf("DEBUG: secret: %s found", secret_name))
	}

	log.Info("debug_jobs <--")
}

func (r *NVMeshReconciler) waitForClearDBToFinish(cr *nvmeshv1.NVMesh) (ctrl.Result, error) {
	log := r.Log.WithValues("method", "clearDB", "component", "Finalizer")
	result, err := r.waitForJobToFinish(cr.GetNamespace(), clearDbJobName)

	if result.Requeue {
		r.debug_jobs(cr)
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
		err := r.UninstallNode(cr, nodeName)
		if err != nil {
			r.Log.Info(fmt.Sprintf("Uninstall Failed on node %s. Error: %s", nodeName, err))
		}
	}

	return err
}

func (r *NVMeshReconciler) getUninstallJob(cr *nvmeshv1.NVMesh, nodeName string) *batchv1.Job {
	jobName := r.getUninstallJobName(nodeName)
	job := r.getNewJob(cr, jobName, uninstallJobImage)

	podSpec := &job.Spec.Template.Spec
	r.addHostPathMount(podSpec, 0, "/opt")
	r.addHostPathMount(podSpec, 0, "/etc/opt")
	r.addHostPathMount(podSpec, 0, "/var/log")

	container := &podSpec.Containers[0]
	container.Env = []corev1.EnvVar{
		{Name: "KEEP_DOWNLOAD_CACHE", Value: "false"},
	}

	podSpec.ServiceAccountName = nvmeshClusterServiceAccountName
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

func (r *NVMeshReconciler) waitForUninstallCompletion(namespace string, nodeList []corev1.Node) (ctrl.Result, error) {
	for _, node := range nodeList {
		jobName := r.getUninstallJobName(node.GetName())
		result, err := r.waitForJobToFinish(namespace, jobName)
		if result.Requeue || err != nil {
			return result, err
		}

		r.Log.Info(fmt.Sprintf("job %s finished", jobName))
	}

	return DoNotRequeue(), nil
}

func (r *NVMeshReconciler) checkUninstallCompletion(cr *nvmeshv1.NVMesh, node *corev1.Node) (bool, error) {
	log := r.Log.WithValues("method", "checkUninstallCompletion", "component", "Finalizer")

	job := &batchv1.Job{}

	objKey := client.ObjectKey{Namespace: cr.GetNamespace(), Name: uninstallJobNamePrefix + node.GetName()}
	err := r.Client.Get(context.TODO(), objKey, job)
	if err != nil {
		return false, errors.Wrap(err, "Failed to get nvmesh-uninstall job")
	}

	if job.Status.Failed+job.Status.Succeeded >= *job.Spec.Completions {
		// all jobs finished
		if job.Status.Succeeded == *job.Spec.Completions {
			return true, nil
		} else {
			// some jobs failed
			return false, goerrors.New(fmt.Sprintf("Uninstall failed on %d nodes. Check failed jobs for details", job.Status.Failed))
		}
	} else {
		// still some jobs running
		log.Info(fmt.Sprintf("Node Uninstall Jobs Status: out of %d total nodes, %d succeeded, %d failed and %d are still running\n", *job.Spec.Completions, job.Status.Succeeded, job.Status.Failed, job.Status.Active))
		return false, nil
	}
}

func (r *NVMeshReconciler) isSingleJobCompleted(job *batchv1.Job) (completed bool, err error) {
	s := job.Status
	if s.Active > 0 {
		return false, nil
	} else if s.Failed+s.Succeeded >= *job.Spec.Completions {
		return true, nil
	} else if s.Failed > 0 {
		return true, errors.New(fmt.Sprintf("Job %s failed", job.GetName()))
	} else if len(s.Conditions) > 0 && s.Conditions[0].Type == "Failed" {
		return true, errors.New(s.Conditions[0].Message)
	}

	return false, nil
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
	mongoImage := GetMongoForNVMeshImageName()
	job := r.getNewJob(cr, clearDbJobName, mongoImage)
	backoffLimit := int32(1)
	job.Spec.BackoffLimit = &backoffLimit
	container := &job.Spec.Template.Spec.Containers[0]

	container.Command = []string{"mongo"}
	mongoConnString := GetMongoConnectionString(cr) + "/management"
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

	result, err := r.waitForUninstallCompletion(nvmeshCluster.GetNamespace(), nodeList)
	if err != nil || result.Requeue {
		return result, err
	}

	if err := r.deleteUninstallJobs(nvmeshCluster, nodeList); err != nil {
		return DoNotRequeue(), err
	}

	return DoNotRequeue(), nil
}

func (r *NVMeshReconciler) UninstallNode(cr *nvmeshv1.NVMesh, nodeName string) error {
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
	r.Log.Info(fmt.Sprintf("Removing service account %s in namespace %s", sa.GetName(), sa.GetNamespace()))
	err := r.Client.Delete(context.TODO(), sa)
	if err != nil && !k8serrors.IsNotFound(err) {
		r.Log.Info(fmt.Sprintf("Warning: failed to delete Cluster Service Account %s in namespace %s", sa.GetName(), sa.GetNamespace()))
		return ctrl.Result{}, err
	}

	r.removeClusterServiceAccountFromSCC(cr)
	return ctrl.Result{}, nil
}
