package controllers

import (
	"context"
	"math"
	"time"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *NVMeshReconciler) DoNotRequeue(cr *nvmeshv1.NVMesh) (ctrl.Result, error) {
	log := r.Log.WithValues("method", "DoNotRequeue")

	if cr != nil {
		cr.Status.ReconcileStatus = nvmeshv1.ReconcileStatus{
			LastUpdate: metav1.Now(),
			Status:     "Success",
		}

		err := r.Client.Status().Update(context.Background(), cr)

		if err != nil {
			log.Error(err, "unable to update status")
		}
	}

	return reconcile.Result{}, nil
}

func (r *NVMeshReconciler) RequeueWithError(cr *nvmeshv1.NVMesh, issue error) (reconcile.Result, error) {
	log := r.Log.WithValues("method", "RequeueWithError")
	var retryInterval time.Duration

	r.EventManager.Warning(cr, "ProcessingError", issue.Error())

	lastUpdate := cr.Status.ReconcileStatus.LastUpdate.Time
	lastStatus := cr.Status.ReconcileStatus.Status

	newStatus := nvmeshv1.ReconcileStatus{
		LastUpdate: metav1.Now(),
		Reason:     issue.Error(),
		Status:     "Failure",
	}

	cr.Status.ReconcileStatus = newStatus
	err := r.Client.Status().Update(context.Background(), cr)

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
