package controllers

import (
	"context"
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

func (r *NVMeshReconciler) waitForJobToFinish(namespace string, jobName string) (ctrl.Result, error) {
	job := &batchv1.Job{}

	objKey := client.ObjectKey{Name: jobName, Namespace: namespace}
	err := r.Client.Get(context.TODO(), objKey, job)

	if err != nil {
		return DoNotRequeue(), err
	}

	completed, err := r.isSingleJobCompleted(job)

	if err != nil {
		return DoNotRequeue(), err
	} else {
		// no error
		if !completed {
			r.Log.Info(fmt.Sprintf("Waiting for %s to finish", job.ObjectMeta.GetName()))
			return Requeue(time.Second), nil
		} else {
			return DoNotRequeue(), nil
		}
	}
}

func (r *NVMeshReconciler) deleteJob(namespace string, jobName string) error {
	job := &batchv1.Job{}
	job.SetName(jobName)
	job.SetNamespace(namespace)

	err := r.Client.Delete(context.TODO(), job)
	if err != nil && !k8serrors.IsNotFound(err) {
		return errors.Wrap(err, fmt.Sprintf("Failed to delete job %s", jobName))
	}

	// Find all Pods related to this job and delete them as well
	podList := &corev1.PodList{}
	matchLabels := client.MatchingLabels{"job-name": jobName}
	err = r.Client.List(context.TODO(), podList, matchLabels)
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

func MatchNode(nodeName string) map[string]string {
	return map[string]string{"kubernetes.io/hostname": nodeName}
}

func (r *NVMeshReconciler) getNewJob(cr *nvmeshv1.NVMesh, jobName string, image string) *batchv1.Job {
	backOffLimit := int32(3)
	completions := int32(1)
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: cr.GetNamespace(),
			Labels:    r.GetOperatorLabels(cr),
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backOffLimit,
			Completions:  &completions,
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					ImagePullSecrets: []v1.LocalObjectReference{{Name: registryCredSecretName}},
					RestartPolicy:    corev1.RestartPolicyOnFailure,
					Containers: []v1.Container{
						{
							Name:            jobName,
							Image:           image,
							ImagePullPolicy: GetGlobalImagePullPolicy(),
						},
					},
				},
			},
		},
	}
}

func (r *NVMeshReconciler) addHostPathMount(podSpec *corev1.PodSpec, containerIndex int, path string) {
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

func (r *NVMeshReconciler) addConfigMapMount(podSpec *corev1.PodSpec, containerIndex int, configMapName string, path string) {
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
