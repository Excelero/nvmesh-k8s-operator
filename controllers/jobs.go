package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
	errors "github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	registryCredSecretName = "excelero-registry-cred"
)

func (r *NVMeshBaseReconciler) waitForJobToFinish(cr *nvmeshv1.NVMesh, jobName string) (ctrl.Result, error) {
	return r.waitForJobToFinishWithoutCR(cr.GetNamespace(), jobName, cr.Spec.Debug.DebugJobs)
}

func (r *NVMeshBaseReconciler) waitForJobToFinishWithoutCR(namespace string, jobName string, debug bool) (ctrl.Result, error) {
	job := &batchv1.Job{}

	objKey := client.ObjectKey{Name: jobName, Namespace: namespace}
	err := r.Client.Get(context.TODO(), objKey, job)

	if err != nil {
		return DoNotRequeue(), err
	}

	completed, err := r.isSingleJobCompleted(job)

	if err != nil {
		return DoNotRequeue(), err
	}

	// no error
	if !completed {
		r.Log.Info(fmt.Sprintf("Waiting for %s to finish", job.ObjectMeta.GetName()))
		r.monitorJob(jobName, namespace, debug)
		return Requeue(time.Second), nil
	}

	return DoNotRequeue(), nil
}

func (r *NVMeshBaseReconciler) isSingleJobCompleted(job *batchv1.Job) (completed bool, err error) {
	s := job.Status

	if s.Failed > 0 {
		return true, fmt.Errorf("Job %s failed", job.GetName())
	} else if s.Succeeded == *job.Spec.Completions {
		return true, nil
	} else if s.Active > 0 {
		return false, nil
	} else if len(s.Conditions) > 0 && s.Conditions[0].Type == "Failed" {
		return true, fmt.Errorf(s.Conditions[0].Message)
	}

	return false, nil
}

func (r *NVMeshBaseReconciler) getJobPods(namespace string, jobName string) (*corev1.PodList, error) {
	podList := &corev1.PodList{}
	matchLabels := client.MatchingLabels{"job-name": jobName}
	err := r.Client.List(context.TODO(), podList, matchLabels)
	return podList, err
}

func (r *NVMeshBaseReconciler) deleteJob(namespace string, jobName string) error {
	job := &batchv1.Job{}
	job.SetName(jobName)
	job.SetNamespace(namespace)

	err := r.Client.Delete(context.TODO(), job)
	if err != nil && !k8serrors.IsNotFound(err) {
		return errors.Wrap(err, fmt.Sprintf("Failed to delete job %s", jobName))
	}

	// Find all Pods related to this job and delete them as well
	podList, err := r.getJobPods(namespace, jobName)
	if err != nil && !k8serrors.IsNotFound(err) {
		return errors.Wrap(err, fmt.Sprintf("Failed to find pods from job %s for deletion", jobName))
	}

	for _, pod := range podList.Items {
		err = r.Client.Delete(context.TODO(), &pod)
		if err != nil && !k8serrors.IsNotFound(err) {
			return errors.Wrap(err, fmt.Sprintf("Failed to delete pod %s", pod.GetName()))
		}
	}

	return nil
}

func matchNode(nodeName string) map[string]string {
	return map[string]string{"kubernetes.io/hostname": nodeName}
}

func (r *NVMeshBaseReconciler) getNewJob(cr *nvmeshv1.NVMesh, jobName string, image string) *batchv1.Job {
	backOffLimit := int32(3)
	completions := int32(1)

	labels := r.getOperatorLabels(cr)
	labels["job-name"] = jobName

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: cr.GetNamespace(),
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backOffLimit,
			Completions:  &completions,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: v1.PodSpec{
					ServiceAccountName: nvmeshClusterServiceAccountName,
					ImagePullSecrets:   []v1.LocalObjectReference{{Name: registryCredSecretName}},
					RestartPolicy:      corev1.RestartPolicyOnFailure,
					Containers: []v1.Container{
						{
							Name:            jobName,
							Image:           image,
							ImagePullPolicy: r.getImagePullPolicy(cr),
						},
					},
				},
			},
		},
	}
}

func (r *NVMeshBaseReconciler) addHostPathMount(podSpec *corev1.PodSpec, containerIndex int, path string) {
	volName := strings.ToLower(strings.ReplaceAll(path[1:], "/", "-"))
	vol := v1.Volume{
		Name: volName,
		VolumeSource: v1.VolumeSource{
			HostPath: &v1.HostPathVolumeSource{
				Path: path,
			},
		}}
	podSpec.Volumes = append(podSpec.Volumes, vol)

	mount := v1.VolumeMount{
		Name:      volName,
		MountPath: path,
	}

	podSpec.Containers[containerIndex].VolumeMounts = append(podSpec.Containers[containerIndex].VolumeMounts, mount)
}

func (r *NVMeshBaseReconciler) addConfigMapMount(podSpec *corev1.PodSpec, containerIndex int, configMapName string, path string) {
	volName := "config-map-vol-" + configMapName
	vol := v1.Volume{
		Name: volName,
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: configMapName,
				},
			},
		}}
	podSpec.Volumes = append(podSpec.Volumes, vol)

	mount := v1.VolumeMount{
		Name:      volName,
		MountPath: path,
	}

	podSpec.Containers[containerIndex].VolumeMounts = append(podSpec.Containers[containerIndex].VolumeMounts, mount)
}

func (r *NVMeshBaseReconciler) printJob(jobName string, namespace string) {
	jobJSONBytes, err := r.getJobAsJSON(jobName, namespace)

	if err != nil {
		r.Log.Info(fmt.Sprintf("Error printing json %s", err))
	} else {
		r.Log.Info(fmt.Sprintf("DEBUG: Job %s:\n%s", jobName, string(jobJSONBytes)))
	}
}

func (r *NVMeshBaseReconciler) getJobAsJSON(jobName string, namespace string) ([]byte, error) {
	jobKey := client.ObjectKey{Name: jobName, Namespace: namespace}
	job := &batchv1.Job{}
	err := r.Client.Get(context.TODO(), jobKey, job)
	if err != nil {
		r.Log.Info(fmt.Sprintf("DEBUG: Failed to get job %s. Error: %s", jobName, err))
	}
	bytes, err := json.MarshalIndent(job, "", "    ")

	return bytes, err
}

func (r *NVMeshBaseReconciler) monitorJob(jobName string, namespace string, debugJobs bool) {
	log := r.Log.WithName("debug_cluster")
	// 1. check the job pods and it's status
	podList, err := r.getJobPods(namespace, jobName)
	if err != nil && !k8serrors.IsNotFound(err) {
		log.Info(fmt.Sprintf("Warning: Failed to list pods from job %s", jobName))
	}

	log.Info(fmt.Sprintf("Found %d pods for job %s", len(podList.Items), jobName))

	if len(podList.Items) == 0 {
		log.Info(fmt.Sprintf("Found no pods from job %s", jobName))
	}

	if debugJobs {
		log.Info("debugJobs -->")

		for _, pod := range podList.Items {
			log.Info(fmt.Sprintf("DEBUG: JOB PODS - pod: %s status: %+v", pod.GetName(), pod.Status.ContainerStatuses))
		}

		r.printJob(jobName, namespace)

		r.printAllPodsStatuses(namespace)

		// If a job doesn't finish it is possible a container has ErrImagePull
		// which could be caused by missing or wrong secrets
		r.verifyNVMeshSecretsExist(namespace)
		log.Info("debugJobs <--")
	}
}
