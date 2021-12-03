package controllers

import (
	"context"
	"fmt"
	"time"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/pkg/api/v1"
	"github.com/prometheus/common/log"

	conditions "excelero.com/nvmesh-k8s-operator/pkg/conditions"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

//ManageSuccess - Handles Reconcile success result
func (r *NVMeshReconciler) ManageSuccess(cr *nvmeshv1.NVMesh, result ctrl.Result) (ctrl.Result, error) {
	//log := r.Log.WithName("method", "DoNotRequeue")
	var generation int64 = -1

	if cr != nil {
		generation = cr.ObjectMeta.GetGeneration()

		isReadyCondition := nvmeshv1.ClusterCondition{
			Type:   nvmeshv1.Ready,
			Reason: "",
			Status: nvmeshv1.ConditionTrue,
		}
		conditions.SetStatusCondition(&cr.Status.Conditions, &isReadyCondition)

		r.populateStatusFields(cr)
		err := r.UpdateStatus(cr)

		if err != nil && !k8serrors.IsNotFound(err) {
			log.Error(err, "Failed to update status")

			// If we failed to update the status let's requeue and have the next cycle update the status
			return reconcile.Result{
				RequeueAfter: time.Second,
				Requeue:      true,
			}, nil
		}
	}

	fmt.Printf("Reconcile Success. Cycle #: %d, Generation: %d\n", reconcileCycles, generation)
	return result, nil
}

func (r *NVMeshReconciler) populateStatusFields(cr *nvmeshv1.NVMesh) {
	cr.Status.WebUIURL = r.getManagementGUIURL(cr)
}

//ManageError - Handles Reconcile errors, updates CR status, prints to log, and returns reconcile.Result
func (r *NVMeshReconciler) ManageError(cr *nvmeshv1.NVMesh, issue error, result ctrl.Result) (reconcile.Result, error) {
	log := r.Log.WithName("ManageError")

	log.Info(fmt.Sprintf("Reconcile cycle failed. %s", issue))

	r.EventManager.Warning(cr, "ProcessingError", issue.Error())

	newCondition := nvmeshv1.ClusterCondition{
		Type:   nvmeshv1.Ready,
		Reason: issue.Error(),
		Status: nvmeshv1.ConditionFalse,
	}

	conditions.SetStatusCondition(&cr.Status.Conditions, &newCondition)

	r.populateStatusFields(cr)
	err := r.UpdateStatus(cr)

	if err != nil && !k8serrors.IsNotFound(err) {
		log.Error(err, "Failed to update status")

		return reconcile.Result{
			RequeueAfter: time.Second,
			Requeue:      true,
		}, nil
	}

	return result, nil
}

//UpdateStatus - Updates CR Status field
func (r *NVMeshReconciler) UpdateStatus(cr *nvmeshv1.NVMesh) error {
	handle_err := func(err error) error {
		if k8serrors.IsNotFound(err) {
			// if the object was deleted, failing to update it's status is not an error
			return nil
		}
		return err
	}

	var updatedStatus nvmeshv1.NVMeshStatus = cr.Status
	r.Log.V(VerboseLogging).Info(fmt.Sprintf("Updating status to %+v", updatedStatus))
	firstTime := true
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if !firstTime {
			err := r.Client.Get(context.TODO(), types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, cr)
			if err != nil {
				return handle_err(err)
			}
		}

		firstTime = false

		cr.Status = updatedStatus
		err := r.Client.Status().Update(context.TODO(), cr)
		r.Log.V(VerboseLogging).Info(fmt.Sprintf("RetryOnConflict for Updating status to %+v, err=%+v", cr.Status, err))
		return handle_err(err)
	})

	return err
}
