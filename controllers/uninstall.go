package controllers

import (
	"context"
	goerrors "errors"
	"fmt"
	"time"

	errors "github.com/pkg/errors"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	uninstallJobFile   = "resources/uninstall/uninstall_job.yaml"
	clearDbJobFile     = "resources/uninstall/clear_db_job.yaml"
	uninstallJobPrefix = "nvmesh-uninstall-"
	clearDbJobName     = "nvmesh-clear-db-job"
)

var UninstallAction = nvmeshv1.ClusterAction{Name: "uninstall"}

type TaskFunc func(cr *nvmeshv1.NVMesh) (ctrl.Result, error)

type Task struct {
	Name string
	Run  TaskFunc
}

func (r *NVMeshReconciler) UninstallCluster(nvmeshCluster *nvmeshv1.NVMesh) (ctrl.Result, error) {
	log := r.Log.WithValues("method", "UninstallCluster", "component", "Finalizer")

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

func (r *NVMeshReconciler) waitForClearDBToFinish(cr *nvmeshv1.NVMesh) (ctrl.Result, error) {
	log := r.Log.WithValues("method", "clearDB", "component", "Finalizer")
	result, err := r.waitForJobToFinish(cr.GetNamespace(), clearDbJobName)

	if result.Requeue {
		return result, nil
	}

	if err != nil {
		log.Info("WARNING: Unable to clear MongoDB Database")
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

func (r *NVMeshReconciler) runUninstallJobs(cr *nvmeshv1.NVMesh, nodeList []*corev1.Node) error {

	// create uninstall job from a template file that will run a pod on each of the found nodes
	jobTemplate, err := r.getUninstallJob()
	if err != nil {
		return err
	}

	var firstError error
	for _, node := range nodeList {
		job := jobTemplate.DeepCopy()
		err := r.UninstallNode(node.GetName(), cr.GetNamespace(), job)
		if err != nil {
			r.Log.Info(fmt.Sprintf("Uninstall Failed on node %s", node.GetName()))
			if firstError == nil {
				firstError = err
			}
		}
	}

	return firstError
}

func (r *NVMeshReconciler) getUninstallJob() (*batchv1.Job, error) {
	decoder := r.getDecoder()
	objects, err := YamlFileToObjects(uninstallJobFile, decoder)
	if err != nil {
		return nil, errors.Wrap(err, "Error gettings UninstallJob template from file")
	}

	job := objects[0].(*batchv1.Job)

	labels := job.ObjectMeta.Labels
	labels["app.kubernetes.io/managed-by"] = "nvmesh-operator"
	job.Spec.Template.Spec.NodeSelector = client.MatchingLabels{nvmeshClientLabelKey: ""}

	return job, nil
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

func (r *NVMeshReconciler) getAllMgmtLabelledNodes(cr *nvmeshv1.NVMesh) ([]*corev1.Node, error) {
	nodes, err := r.getNodesWithLabel(nvmeshMgmtLabelKey, "")
	if err != nil {
		return nil, err
	}

	nodeList := []*corev1.Node{}
	for _, n := range nodes.Items {
		nodeList = append(nodeList, &n)
	}

	return nodeList, nil
}

func (r *NVMeshReconciler) getAllNVMeshClusterNodes(cr *nvmeshv1.NVMesh) ([]*corev1.Node, error) {

	clients, err := r.getNodesWithLabel(nvmeshClientLabelKey, "")
	if err != nil {
		return nil, err
	}
	targets, err := r.getNodesWithLabel(nvmeshTargetLabelKey, "")
	if err != nil {
		return nil, err
	}

	nodesDict := make(map[string]bool)
	nodesList := make([]*corev1.Node, 0)

	for i, c := range clients.Items {
		nodesDict[c.GetName()] = true
		nodesList = append(nodesList, &clients.Items[i])
	}

	for i, t := range targets.Items {
		if _, ok := nodesDict[t.GetName()]; !ok {
			nodesDict[t.GetName()] = true
			nodesList = append(nodesList, &targets.Items[i])
		}
	}

	return nodesList, err
}

func (r *NVMeshReconciler) waitForUninstallCompletion(namespace string, nodeList []*corev1.Node) (ctrl.Result, error) {
	for _, node := range nodeList {
		result, err := r.waitForJobToFinish(namespace, uninstallJobPrefix+node.GetName())
		if result.Requeue || err != nil {
			return result, err
		}
	}

	return DoNotRequeue(), nil
}

func (r *NVMeshReconciler) checkUninstallCompletion(cr *nvmeshv1.NVMesh, node *corev1.Node) (bool, error) {
	log := r.Log.WithValues("method", "checkUninstallCompletion", "component", "Finalizer")

	job := &batchv1.Job{}

	objKey := client.ObjectKey{Namespace: cr.GetNamespace(), Name: uninstallJobPrefix + node.GetName()}
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

func (r *NVMeshReconciler) deleteUninstallJobs(cr *nvmeshv1.NVMesh, nodeList []*corev1.Node) error {

	for _, node := range nodeList {
		jobName := uninstallJobPrefix + node.GetName()
		err := r.deleteJob(cr.GetNamespace(), jobName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *NVMeshReconciler) runClearDbJob(cr *nvmeshv1.NVMesh) error {
	decoder := r.getDecoder()
	objects, err := YamlFileToObjects(clearDbJobFile, decoder)
	if err != nil {
		return err
	}

	job := objects[0].(*batchv1.Job)

	job.ObjectMeta.Namespace = cr.GetNamespace()
	labels := job.ObjectMeta.Labels
	labels["app.kubernetes.io/managed-by"] = "nvmesh-operator"
	labels[nvmeshClusterNameLabelKey] = cr.GetName()
	container := &job.Spec.Template.Spec.Containers[0]
	container.Args[0] = GetMongoConnectionString(cr) + "/management"
	container.Image = GetMongoForNVMeshImageName()

	err = r.Client.Create(context.TODO(), job)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "Failed to create Clear DB job")
	}

	return nil
}

func (r *NVMeshReconciler) uninstallClusterNodes(nvmeshCluster *nvmeshv1.NVMesh) (ctrl.Result, error) {
	nodeList, err := r.getAllNVMeshClusterNodes(nvmeshCluster)
	if err != nil {
		return DoNotRequeue(), errors.Wrap(err, fmt.Sprintf("Failed to list all of the nodes in NVMesh Cluster %s", nvmeshCluster.GetName()))
	}

	r.Log.Info(fmt.Sprintf("Uninstalling %d nodes", len(nodeList)))
	for _, n := range nodeList {
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

func (r *NVMeshReconciler) UninstallNode(nodeName string, namespace string, job *batchv1.Job) error {
	r.Log.Info(fmt.Sprintf("Running Uninstall Job on Node %s", nodeName))
	var err error
	if job == nil {
		job, err = r.getUninstallJob()
		if err != nil {
			return err
		}
	}

	job.ObjectMeta.Name = uninstallJobPrefix + nodeName
	if namespace != "" {
		job.ObjectMeta.Namespace = namespace
	}

	job.Spec.Selector.MatchLabels["job-name"] = job.ObjectMeta.Name
	job.Spec.Template.ObjectMeta.Labels = job.Spec.Selector.MatchLabels
	job.Spec.Template.Spec.NodeSelector = client.MatchingLabels{"kubernetes.io/hostname": nodeName}

	err = r.Client.Create(context.TODO(), job)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return errors.Wrap(err, fmt.Sprintf("Failed to run UninstallJob on node %s", nodeName))
	}

	return nil
}
