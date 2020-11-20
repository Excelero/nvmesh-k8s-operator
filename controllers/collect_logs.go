package controllers

import (
	"context"
	"time"

	errors "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"fmt"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	collectLogsImageName     = "registry.excelero.com/nvmesh-logs-collector:" + CoreImageVersionTag
	collectDbJobName         = "collect-db"
	collectConfigMapsJobName = "collect-config-maps"
	collectLogsJobName       = "collect-logs"

	CollectDBStage         = "CollectDB"
	CollectConfigMapsStage = "CollectConfigMaps"
	CollectLogsStage       = "CollectLogs"
	WaitForJobsToFinish    = "WaitForJobsToFinish"
	DeleteJobsStage        = "DeleteJobs"

	mgmtConfigMapName   = "nvmesh-mgmt-config"
	nvmeshConfigMapName = "nvmesh-core-config"
	csiConfigMapName    = "nvmesh-csi-config"

	s3bucketSecretName = "s3-bucket-secrets"
	logsSavePath       = "/opt/nvmesh-operator/logs"
)

func (r *NVMeshReconciler) handleCollectLogs(cr *nvmeshv1.NVMesh, a nvmeshv1.ClusterAction) (bool, ctrl.Result, error) {
	// run db dump job
	if !r.isTaskFinished(cr, a, CollectDBStage) {
		r.setTaskStarted(cr, a, CollectDBStage)
		err := r.runCollectDBJob(cr, a)
		if err != nil {
			return false, DoNotRequeue(), err
		}
		r.setTaskFinished(cr, a, CollectDBStage)
	}

	// run collect config-maps job
	if !r.isTaskFinished(cr, a, CollectConfigMapsStage) {
		r.setTaskStarted(cr, a, CollectConfigMapsStage)
		err := r.runCollectConfigMapsJob(cr, a)
		if err != nil {
			return false, DoNotRequeue(), err
		}
		r.setTaskFinished(cr, a, CollectConfigMapsStage)
	}

	// get all Cluster Nodes (including mgmt labelled nodes) for logs collection
	nodeSet, err := r.getAllNVMeshClusterNodes(cr)
	if err != nil {
		return false, DoNotRequeue(), errors.Wrap(err, fmt.Sprintf("Failed to list all of the nodes in NVMesh Cluster %s", cr.GetName()))
	}

	// get all management labelled nodes, on these nodes it is possible that management pods were running.
	mgmtLabeledNodes, err := r.getAllMgmtLabelledNodes(cr)
	if err != nil {
		return false, DoNotRequeue(), errors.Wrap(err, fmt.Sprintf("Failed to list nodes with label %s", nvmeshMgmtLabelKey))
	}

	for _, n := range mgmtLabeledNodes.Items {
		nodeName := n.GetName()
		if _, ok := nodeSet[nodeName]; !ok {
			nodeSet[nodeName] = n
		}
	}

	nodeList := r.nodeSetToList(nodeSet)

	// create logs collector jobs
	if !r.isTaskFinished(cr, a, CollectLogsStage) {
		r.setTaskStarted(cr, a, CollectLogsStage)

		for _, node := range nodeList {
			err := r.createCollectLogsJob(cr, a, node.GetName())
			if err != nil {
				return false, DoNotRequeue(), err
			}
		}

		r.setTaskFinished(cr, a, CollectLogsStage)
	}

	if !r.isTaskFinished(cr, a, WaitForJobsToFinish) {
		r.setTaskStarted(cr, a, WaitForJobsToFinish)

		// wait for db dump job to finish
		res, err := r.waitForJobToFinish(cr.GetNamespace(), collectDbJobName)
		if err != nil {
			return false, res, err
		}

		if res.Requeue {
			res.RequeueAfter = time.Second * 3
			return false, res, err
		}

		// wait for collect config-maps job to finish
		res, err = r.waitForJobToFinish(cr.GetNamespace(), collectConfigMapsJobName)
		if err != nil {
			return false, res, err
		}

		if res.Requeue {
			res.RequeueAfter = time.Second * 3
			return false, res, err
		}

		// wait for logs collector jobs to finish
		for _, node := range nodeList {
			jobName := collectLogsJobName + "-" + sanitizeString(node.GetName())
			res, err = r.waitForJobToFinish(cr.GetNamespace(), jobName)
			if res.Requeue || err != nil {
				res.RequeueAfter = time.Second * 3
				return false, res, err
			}
		}

		r.setTaskFinished(cr, a, WaitForJobsToFinish)
	}

	if !r.isTaskFinished(cr, a, DeleteJobsStage) {
		r.setTaskStarted(cr, a, DeleteJobsStage)
		err := r.deleteCollectLogJobs(cr, nodeList)
		if err != nil && !k8serrors.IsNotFound(err) {
			return false, Requeue(time.Second), err
		}

		r.setTaskFinished(cr, a, DeleteJobsStage)
	}

	r.setActionComplete(cr, a)
	return true, DoNotRequeue(), nil
}

func (r *NVMeshReconciler) deleteCollectLogJobs(cr *nvmeshv1.NVMesh, nodeList []corev1.Node) error {

	err := r.deleteJob(cr.GetNamespace(), collectDbJobName)
	if err != nil {
		return err
	}

	err = r.deleteJob(cr.GetNamespace(), collectConfigMapsJobName)
	if err != nil {
		return err
	}

	for _, node := range nodeList {
		jobName := collectLogsJobName + "-" + sanitizeString(node.GetName())
		err = r.deleteJob(cr.GetNamespace(), jobName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *NVMeshReconciler) getNVMeshClusterName(cr *nvmeshv1.NVMesh) string {
	return cr.GetNamespace() + "_" + cr.GetName()
}

func (r *NVMeshReconciler) addClusterNameEnvVar(container *corev1.Container, cr *nvmeshv1.NVMesh) {
	if container.Env == nil {
		container.Env = []corev1.EnvVar{}
	}

	cluster_name_var := corev1.EnvVar{
		Name:  "CLUSTER_NAME",
		Value: r.getNVMeshClusterName(cr),
	}

	container.Env = append(container.Env, cluster_name_var)
}

func (r *NVMeshReconciler) addS3CredentialsEnvVar(container *corev1.Container, bucketName string) {
	if container.Env == nil {
		container.Env = []corev1.EnvVar{}
	}

	s3vars := []corev1.EnvVar{
		{
			Name: "S3_KEY_ID",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: s3bucketSecretName,
					},
					Key: "AWS_ACCESS_KEY_ID",
				},
			},
		},
		{
			Name: "S3_KEY_SECRET",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: s3bucketSecretName,
					},
					Key: "AWS_SECRET_ACCESS_KEY",
				},
			},
		},
		{
			Name:  "S3_BUCKET_NAME",
			Value: bucketName,
		},
	}

	container.Env = append(container.Env, s3vars...)
}

func setContainerAsPrivileged(container *corev1.Container) {
	if container.SecurityContext == nil {
		container.SecurityContext = &corev1.SecurityContext{}
	}

	isPrivileged := true
	container.SecurityContext.Privileged = &isPrivileged
}

func (r *NVMeshReconciler) runCollectDBJob(cr *nvmeshv1.NVMesh, action nvmeshv1.ClusterAction) error {
	job := r.getNewJob(cr, collectDbJobName, collectLogsImageName)
	grace := int64(1)
	podSpec := &job.Spec.Template.Spec

	podSpec.TerminationGracePeriodSeconds = &grace
	container := &podSpec.Containers[0]
	container.Command = []string{"sudo"}
	container.Args = []string{"-E", "/init.sh", "--db-dump"}
	container.Env = []corev1.EnvVar{
		{
			Name:  "MONGO_URI",
			Value: "mongodb://" + GetMongoConnectionString(cr) + "/management",
		},
	}

	podSpec.ServiceAccountName = nvmeshClusterServiceAccountName
	setContainerAsPrivileged(container)

	r.addClusterNameEnvVar(container, cr)

	if bucketName, ok := getActionArg(action, "upload-to-s3"); ok {
		container.Args = append(container.Args, "--upload-to-s3")
		r.addS3CredentialsEnvVar(container, bucketName)
	}

	if r.Options.Debug.CollectLogsJobsRunForever {
		container.Args = append(container.Args, "--debug")
	}

	r.addHostPathMount(podSpec, 0, logsSavePath)

	err := r.Client.Create(context.TODO(), job)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return errors.Wrap(err, fmt.Sprintf("Failed to create %s", job.GetName()))
	}

	return nil
}

func (r *NVMeshReconciler) runCollectConfigMapsJob(cr *nvmeshv1.NVMesh, action nvmeshv1.ClusterAction) error {
	job := r.getNewJob(cr, collectConfigMapsJobName, collectLogsImageName)
	grace := int64(1)
	podSpec := &job.Spec.Template.Spec

	podSpec.TerminationGracePeriodSeconds = &grace
	container := &podSpec.Containers[0]
	container.Command = []string{"/bin/bash"}
	container.Args = []string{"-c", "/init.sh --config-maps"}

	podSpec.ServiceAccountName = nvmeshClusterServiceAccountName
	setContainerAsPrivileged(container)

	r.addClusterNameEnvVar(container, cr)

	if bucketName, ok := getActionArg(action, "upload-to-s3"); ok {
		container.Args = append(container.Args, "--upload-to-s3")
		r.addS3CredentialsEnvVar(container, bucketName)
	}

	if r.Options.Debug.CollectLogsJobsRunForever {
		container.Args = append(container.Args, "--debug")
	}

	r.addHostPathMount(podSpec, 0, logsSavePath)

	r.addConfigMapMount(podSpec, 0, mgmtConfigMapName, "/config-maps/"+mgmtConfigMapName)
	r.addConfigMapMount(podSpec, 0, nvmeshConfigMapName, "/config-maps/"+nvmeshConfigMapName)
	r.addConfigMapMount(podSpec, 0, csiConfigMapName, "/config-maps/"+csiConfigMapName)

	err := r.Client.Create(context.TODO(), job)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return errors.Wrap(err, fmt.Sprintf("Failed to create %s", job.GetName()))
	}

	return nil
}

func (r *NVMeshReconciler) createCollectLogsJob(cr *nvmeshv1.NVMesh, action nvmeshv1.ClusterAction, nodeName string) error {
	jobName := collectLogsJobName + "-" + sanitizeString(nodeName)
	job := r.getNewJob(cr, jobName, collectLogsImageName)

	podSpec := &job.Spec.Template.Spec

	// use HostNetwork so that the hostname inside the container will match the hostname of the node
	podSpec.HostNetwork = true

	// use HostPID so that logs collector could check toma's process status
	podSpec.HostPID = true

	containerIndex := 0
	container := &podSpec.Containers[containerIndex]
	container.Command = []string{"sudo"}
	container.Args = []string{"-E", "/init.sh", "--node-logs"}

	podSpec.ServiceAccountName = nvmeshClusterServiceAccountName
	setContainerAsPrivileged(container)
	r.addClusterNameEnvVar(container, cr)

	if bucketName, ok := getActionArg(action, "upload-to-s3"); ok {
		container.Args = append(container.Args, "--upload-to-s3")
		r.addS3CredentialsEnvVar(container, bucketName)
	}

	if r.Options.Debug.CollectLogsJobsRunForever {
		container.Args = append(container.Args, "--debug")
	}

	mounts := []string{
		"/var/opt/NVMesh",
		"/var/log/NVMesh",
		"/opt/NVMesh",
		logsSavePath,
	}

	for _, mount := range mounts {
		r.addHostPathMount(podSpec, containerIndex, mount)
	}

	podSpec.NodeSelector = MatchNode(nodeName)

	err := r.Client.Create(context.TODO(), job)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return errors.Wrap(err, fmt.Sprintf("Failed to create %s on node %s", jobName, nodeName))
	}

	r.Log.Info(fmt.Sprintf("Created collect log job for node %s\n", nodeName))
	return nil
}
