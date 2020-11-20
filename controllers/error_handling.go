package controllers

import (
	"context"
	"fmt"
	"math"
	"time"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
	"github.com/prometheus/common/log"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

//ManageSuccess - Handles Reconcile success result
func (r *NVMeshReconciler) ManageSuccess(cr *nvmeshv1.NVMesh, result ctrl.Result) (ctrl.Result, error) {
	//log := r.Log.WithValues("method", "DoNotRequeue")
	var generation int64 = -1

	if cr != nil {
		generation = cr.ObjectMeta.GetGeneration()

		cr.Status.ReconcileStatus = nvmeshv1.ReconcileStatus{
			LastUpdate: metav1.Now(),
			Status:     "Success",
		}

		r.setStatusOnCustomResource(cr)
		err := r.UpdateStatus(cr)

		if err != nil && !k8serrors.IsNotFound(err) {
			log.Error(err, "unable to update status")

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

//ManageError - Handles Reconcile errors, updates CR status, prints to log, and returns reconcile.Result
func (r *NVMeshReconciler) ManageError(cr *nvmeshv1.NVMesh, issue error) (reconcile.Result, error) {
	log := r.Log.WithValues("method", "ManageError")
	var retryInterval time.Duration

	log.Info(fmt.Sprintf("Reconcile cycle failed. %s", issue))

	r.EventManager.Warning(cr, "ProcessingError", issue.Error())

	lastUpdate := cr.Status.ReconcileStatus.LastUpdate.Time
	lastStatus := cr.Status.ReconcileStatus.Status

	newStatus := nvmeshv1.ReconcileStatus{
		LastUpdate: metav1.Now(),
		Reason:     issue.Error(),
		Status:     "Failure",
	}

	r.setStatusOnCustomResource(cr)
	cr.Status.ReconcileStatus = newStatus
	err := r.UpdateStatus(cr)

	if err != nil {
		log.Error(err, "unable to update status")

		return reconcile.Result{
			RequeueAfter: time.Second,
			Requeue:      true,
		}, nil
	}

	if lastUpdate.IsZero() || lastStatus == "Success" {
		retryInterval = time.Second
	} else {
		retryInterval = newStatus.LastUpdate.Sub(lastUpdate).Round(time.Second)
	}

	maxTime := float64(time.Hour.Nanoseconds() * 6)
	doubleLastTime := float64(retryInterval.Nanoseconds() * 2)
	nextReconcile := time.Duration(math.Min(doubleLastTime, maxTime))
	return Requeue(nextReconcile), nil
}

//UpdateStatus - Updates CR Status field
func (r *NVMeshReconciler) UpdateStatus(cr *nvmeshv1.NVMesh) error {
	err := r.Client.Status().Update(context.TODO(), cr)

	if err != nil {
		if k8serrors.IsNotFound(err) {
			return err
		}

		updated := &nvmeshv1.NVMesh{}
		err = r.Client.Get(context.TODO(), types.NamespacedName{Name: cr.GetName(), Namespace: cr.GetNamespace()}, updated)

		if err != nil {
			if k8serrors.IsNotFound(err) {
				// if the object was deleted, failing to update it's status is not an error
				return nil
			}
			return err
		}

		updated.Status = cr.Status
		return r.Client.Status().Update(context.TODO(), cr)
	}

	return nil
}
