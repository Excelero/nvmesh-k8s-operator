package controllers

import (
	"context"
	goerrors "errors"
	"fmt"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
	storagev1 "k8s.io/api/storage/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterFinalizerName = "cluster.finalizers.nvmesh.excelero.com"
)

func (r *NVMeshReconciler) AddFinalizer(nvmeshCluster *nvmeshv1.NVMesh) error {
	//log := r.Log.WithValues("NVMesh Cluster Finalizer", nvmeshCluster.GetNamespace())

	// examine DeletionTimestamp to determine if object is under deletion
	if nvmeshCluster.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !containsString(nvmeshCluster.ObjectMeta.Finalizers, clusterFinalizerName) {
			nvmeshCluster.ObjectMeta.Finalizers = append(nvmeshCluster.ObjectMeta.Finalizers, clusterFinalizerName)
			err := r.Update(context.Background(), nvmeshCluster)
			return err
		}

		return nil
	} else {
		// The object is being deleted
		if containsString(nvmeshCluster.ObjectMeta.Finalizers, clusterFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteExternalResources(nvmeshCluster); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return err
			}

			// remove our finalizer from the list and update it.
			nvmeshCluster.ObjectMeta.Finalizers = removeString(nvmeshCluster.ObjectMeta.Finalizers, clusterFinalizerName)
			if err := r.Update(context.Background(), nvmeshCluster); err != nil {
				return err
			}
		}

		// Stop reconciliation as the item is being deleted
		return nil
	}
}

func (r *NVMeshReconciler) deleteExternalResources(cr *nvmeshv1.NVMesh) error {
	// delete any external resources associated with the Cluster
	// Ensure that delete implementation is idempotent and safe to invoke
	// multiple types for same object.

	// TODO: check if any NVMesh volume is attached
	// TODO: Try to refine the listing using ListOptions, can we list only NVMesh attachments, can we list only attached attachments ?
	attachmentsList := &storagev1.VolumeAttachmentList{}
	listOps := client.ListOptions{}
	err := r.Client.List(context.TODO(), attachmentsList, &listOps)
	if err != nil {
		// TODO: Wrap error with meaningful message
		return err
	}

	// the list returns Thousands of old attachment items that have attached: false
	fmt.Printf("%d", attachmentsList.Size())

	nvmeshAttachments := make([]storagev1.VolumeAttachment, 0)
	if attachmentsList.Size() > 0 {
		for _, attachment := range attachmentsList.Items {
			// TODO: find if it's an attachment of an NVMesh Volume
			if attachment.Status.Attached == true && attachment.Spec.Attacher == "nvmesh-csi.excelero.com" {
				nvmeshAttachments = append(nvmeshAttachments, attachment)
			}

			fmt.Printf("found attachment of %s on node %s", *attachment.Spec.Source.PersistentVolumeName, attachment.Spec.NodeName)
		}
	}

	if len(nvmeshAttachments) > 0 {
		errMsg := "Cannot delete NVMesh cluster while volumes are attached, Please remove any consumer pods to cause the volumes to detach. The following volumes are still attached: "
		for _, a := range nvmeshAttachments {
			errMsg = fmt.Sprintf("%s pvc: %s on node: %s ", errMsg, *a.Spec.Source.PersistentVolumeName, a.Spec.NodeName)
		}
		return goerrors.New(errMsg)
	}
	// TODO: Should we check if there are any PersistentVolume of type NVMesh ?
	// TODO: Should we check the Management api for attachments ?
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

