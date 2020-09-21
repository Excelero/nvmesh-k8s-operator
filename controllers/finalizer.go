package controllers

import (
	"context"
	goerrors "errors"
	"fmt"
	"strings"

	errors "github.com/pkg/errors"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterFinalizerName = "cluster.finalizers.nvmesh.excelero.com"
	nvmeshCsiDriverName  = "nvmesh-csi.excelero.com"
)

func isBeingDeleted(nvmeshCluster *nvmeshv1.NVMesh) bool {
	// examine DeletionTimestamp to determine if object is under deletion
	return !nvmeshCluster.ObjectMeta.DeletionTimestamp.IsZero()
}

func hasFinalizer(nvmeshCluster *nvmeshv1.NVMesh, finName string) bool {
	return containsString(nvmeshCluster.ObjectMeta.Finalizers, finName)
}

func (r *NVMeshReconciler) addFinalizerAndUpdate(nvmeshCluster *nvmeshv1.NVMesh, finName string) error {
	nvmeshCluster.ObjectMeta.Finalizers = append(nvmeshCluster.ObjectMeta.Finalizers, finName)
	return r.Update(context.TODO(), nvmeshCluster)
}

func (r *NVMeshReconciler) removeFinalizerAndUpdate(nvmeshCluster *nvmeshv1.NVMesh, finName string) error {
	// remove our finalizer from the list and update it.
	nvmeshCluster.ObjectMeta.Finalizers = removeString(nvmeshCluster.ObjectMeta.Finalizers, clusterFinalizerName)
	return r.Update(context.TODO(), nvmeshCluster)
}

func (r *NVMeshReconciler) HandleFinalizer(nvmeshCluster *nvmeshv1.NVMesh) error {
	//log := r.Log.WithValues("NVMesh Cluster Finalizer", nvmeshCluster.GetNamespace())

	if isBeingDeleted(nvmeshCluster) {
		// The object is being deleted
		if hasFinalizer(nvmeshCluster, clusterFinalizerName) {
			if err := r.verifyNoExternalDependenciesExist(nvmeshCluster); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return err
			}
			if err := r.removeFinalizerAndUpdate(nvmeshCluster, clusterFinalizerName); err != nil {
				return err
			}
		}

		// Stop reconciliation as the item is being deleted
		return nil
	} else {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !hasFinalizer(nvmeshCluster, clusterFinalizerName) {
			if err := r.addFinalizerAndUpdate(nvmeshCluster, clusterFinalizerName); err != nil {
				return err
			}
		}

		return nil
	}
}

func (r *NVMeshReconciler) verifyNoExternalDependenciesExist(cr *nvmeshv1.NVMesh) error {
	log := r.Log.WithValues("method", "verifyNoExternalDependenciesExist", "component", "Finalizer")

	// delete any external resources associated with the Cluster
	// Ensure that delete implementation is idempotent and safe to invoke
	// multiple types for same object.

	err := r.verifyNoVolumeAttachments(cr)
	if err != nil {
		if cr.Spec.Operator.IgnoreVolumeAttachmentOnDelete == true {
			log.Info(fmt.Sprintf("WARNING: IgnoreVolumeAttachmentOnDelete is true, ignoring volumeAttachments on delete of NVMesh %s. %s", cr.GetName(), err))
		} else {
			return errors.Wrap(err, fmt.Sprintf("Cannot delete NVMesh cluster while volumes are attached. The following volumes are still attached: "))

		}
	}

	err = r.verifyNoPersistentVolumes(cr)
	if err != nil {
		if cr.Spec.Operator.IgnoreVolumeAttachmentOnDelete == true {
			log.Info(fmt.Sprintf("WARNING: IgnoreVolumeAttachmentOnDelete is true, ignoring NVMesh PersistentVolumes on delete of NVMesh %s. %s", cr.GetName(), err))
		} else {
			return errors.Wrap(err, "Cannot delete NVMesh cluster while NVMesh PersistentVolumes are provisioned")
		}
	}

	// TODO: Should we check the Management api for attachments ?
	// TODO: Should we perform any procedure as an uninstall here? (remove cache dirs from all nodes..)
	return nil
}

func (r *NVMeshReconciler) verifyNoVolumeAttachments(cr *nvmeshv1.NVMesh) error {
	attachmentsList := &storagev1.VolumeAttachmentList{}
	listOps := client.ListOptions{}
	err := r.Client.List(context.TODO(), attachmentsList, &listOps)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to list VolumeAttachments"))
	}

	nvmeshAttachments := make([]storagev1.VolumeAttachment, 0)
	if len(attachmentsList.Items) > 0 {
		for _, attachment := range attachmentsList.Items {
			// TODO: find if it's an attachment of an NVMesh Volume
			if attachment.Status.Attached == true && attachment.Spec.Attacher == nvmeshCsiDriverName {
				nvmeshAttachments = append(nvmeshAttachments, attachment)
			}

			fmt.Printf("found attachment of %s on node %s", *attachment.Spec.Source.PersistentVolumeName, attachment.Spec.NodeName)
		}
	}

	if len(nvmeshAttachments) > 0 {
		errMsg := "Found the following attachments"
		for _, a := range nvmeshAttachments {
			errMsg = fmt.Sprintf("%s pvc: %s on node: %s ", errMsg, *a.Spec.Source.PersistentVolumeName, a.Spec.NodeName)
		}
		return goerrors.New(errMsg)
	}

	return nil
}

func (r *NVMeshReconciler) verifyNoPersistentVolumes(cr *nvmeshv1.NVMesh) error {
	pvList := &corev1.PersistentVolumeList{}
	listOps := client.ListOptions{}
	err := r.Client.List(context.TODO(), pvList, &listOps)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to list PersistentVolumes"))

	}

	nvmeshPVs := make([]string, 0)
	if len(pvList.Items) > 0 {
		for _, pv := range pvList.Items {
			if pv.Spec.CSI != nil && pv.Spec.CSI.Driver == nvmeshCsiDriverName {
				nvmeshPVs = append(nvmeshPVs, pv.GetName())
			}
		}
	}

	if len(nvmeshPVs) > 0 {
		errMsg := fmt.Sprintf("Found the following NVMesh volumes: %s", strings.Join(nvmeshPVs, ", "))
		return goerrors.New(errMsg)
	}

	return nil
}

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
