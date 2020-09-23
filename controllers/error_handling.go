package controllers

import (
	"context"
	"fmt"
	"math"
	"time"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
	"github.com/prometheus/common/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *NVMeshReconciler) ManageSuccess(cr *nvmeshv1.NVMesh, requeue bool) (ctrl.Result, error) {
	//log := r.Log.WithValues("method", "DoNotRequeue")
	var generation int64 = -1

	if cr != nil {
		generation = cr.ObjectMeta.GetGeneration()

		cr.Status.ReconcileStatus = nvmeshv1.ReconcileStatus{
			LastUpdate: metav1.Now(),
			Status:     "Success",
		}

		r.setStatusOnCustomResource(cr)
		err := r.Client.Status().Update(context.TODO(), cr)

		if err != nil {
			log.Info(fmt.Sprintf("unable to update status. %s", err))

			// If we failed to update the status let's requeue and have the next cycle update the status
			return reconcile.Result{
				RequeueAfter: time.Second,
				Requeue:      true,
			}, nil
		}

	}

	fmt.Printf("Reconcile Success. Cycle #: %d, Generation: %d\n", reconcileCycles, generation)
	if requeue {
		// trigger antoher reconcile cycle in one second
		return reconcile.Result{
			RequeueAfter: time.Second,
			Requeue:      true,
		}, nil
	} else {
		// Do not trigger another cycle
		return reconcile.Result{}, nil
	}
}

func (r *NVMeshReconciler) ManageError(cr *nvmeshv1.NVMesh, issue error) (reconcile.Result, error) {
	log := r.Log.WithValues("method", "ManageError")
	var retryInterval time.Duration

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
	err := r.Client.Status().Update(context.TODO(), cr)

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
	return reconcile.Result{
		RequeueAfter: nextReconcile,
		Requeue:      true,
	}, nil
}
