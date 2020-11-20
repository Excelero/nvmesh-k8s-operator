package controllers

import (
	"context"
	goerrors "errors"
	"strings"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"fmt"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	actionComplete = "ActionComplete"
	taskFinished   = "TaskFinished"
	taskStarted    = "TaskStarted"
)

// TaskFunc - Type to represent a function to invoke when task is started
type TaskFunc func(cr *nvmeshv1.NVMesh) (ctrl.Result, error)

//Task - represent a sub-task in a ClusterAction
type Task struct {
	Name string
	Run  TaskFunc
}

func (r *NVMeshReconciler) hasActions(cr *nvmeshv1.NVMesh) bool {
	return len(cr.Spec.Actions) > 0
}

func (r *NVMeshReconciler) handleActions(cr *nvmeshv1.NVMesh) (ctrl.Result, error) {
	pendingActions := cr.Spec.Actions

	if len(pendingActions) > 0 {
		for actionIndex, action := range pendingActions {
			shouldRemove, result, err := r.handleAction(action, cr)
			if shouldRemove {
				r.removeAction(actionIndex, cr)
				err := r.updateNVMeshClusterObject(cr)
				if err != nil {
					fmt.Printf("Failed to remove action %s from cluster %s\n", action.Name, cr.GetName())
				}
			}

			if result.Requeue || err != nil {
				if result.Requeue {
					r.UpdateStatus(cr)
				}
				return result, err
			}
		}
	}

	return DoNotRequeue(), nil
}

func (r *NVMeshReconciler) handleAction(action nvmeshv1.ClusterAction, cr *nvmeshv1.NVMesh) (bool, ctrl.Result, error) {
	switch action.Name {
	case "collect-logs":
		return r.handleCollectLogs(cr, action)
	default:
		return false, DoNotRequeue(), goerrors.New(fmt.Sprintf("Unknown Action %s", action.Name))
	}
}

func (r *NVMeshReconciler) removeAction(indexToRemove int, cr *nvmeshv1.NVMesh) {
	newList := make([]nvmeshv1.ClusterAction, len(cr.Spec.Actions)-1)

	for i, item := range cr.Spec.Actions {
		if i < indexToRemove {
			newList[i] = item
		}

		if i > indexToRemove {
			newList[i-1] = item
		}
	}

	cr.Spec.Actions = newList
}

func (r *NVMeshReconciler) removeFinishedActionStatuses(cr *nvmeshv1.NVMesh) ctrl.Result {
	// remove ActionStatus of actions that were finished and deleted
	var found bool
	var result ctrl.Result

	if cr.Status.ActionsStatus != nil {
		for actionName := range cr.Status.ActionsStatus {
			found = false
			for _, action := range cr.Spec.Actions {
				if action.Name == actionName {
					found = true
				}
			}

			if !found {
				delete(cr.Status.ActionsStatus, actionName)
				// when this function is finished immediately start another reconcile cycle
				result.Requeue = true
			}
		}
	}

	if result.Requeue {
		r.UpdateStatus(cr)
	}
	return result
}

func (r *NVMeshReconciler) updateNVMeshClusterObject(cr *nvmeshv1.NVMesh) error {
	err := r.Client.Update(context.TODO(), cr)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	return nil
}

func sanitizeString(s string) string {
	return strings.ReplaceAll(s, ".", "-")
}

func (r *NVMeshReconciler) getTaskStatus(cr *nvmeshv1.NVMesh, action nvmeshv1.ClusterAction, key string) string {
	if cr.Status.ActionsStatus == nil {
		cr.Status.ActionsStatus = make(map[string]nvmeshv1.ActionStatus)
	}

	actionStatus, ok := cr.Status.ActionsStatus[action.Name]
	if !ok {
		cr.Status.ActionsStatus[action.Name] = make(map[string]string)
		actionStatus = cr.Status.ActionsStatus[action.Name]
	}

	return actionStatus[key]
}

func (r *NVMeshReconciler) setTaskStatus(cr *nvmeshv1.NVMesh, action nvmeshv1.ClusterAction, key string, value string) {
	if cr.Status.ActionsStatus == nil {
		cr.Status.ActionsStatus = make(map[string]nvmeshv1.ActionStatus)
	}

	if _, ok := cr.Status.ActionsStatus[action.Name]; !ok {
		cr.Status.ActionsStatus[action.Name] = make(map[string]string)
	}

	cr.Status.ActionsStatus[action.Name][key] = value
}

func (r *NVMeshReconciler) isTaskFinished(cr *nvmeshv1.NVMesh, action nvmeshv1.ClusterAction, key string) bool {
	return r.getTaskStatus(cr, action, key) == taskFinished
}

func (r *NVMeshReconciler) setTaskStarted(cr *nvmeshv1.NVMesh, action nvmeshv1.ClusterAction, key string) {
	r.setTaskStatus(cr, action, key, taskStarted)
}

func (r *NVMeshReconciler) setTaskFinished(cr *nvmeshv1.NVMesh, action nvmeshv1.ClusterAction, key string) {
	r.setTaskStatus(cr, action, key, taskFinished)
}

func (r *NVMeshReconciler) setActionComplete(cr *nvmeshv1.NVMesh, action nvmeshv1.ClusterAction) {
	r.setTaskStatus(cr, action, actionComplete, taskFinished)
}

func getActionArg(action nvmeshv1.ClusterAction, key string) (string, bool) {
	if action.Args != nil {
		val, ok := action.Args[key]
		return val, ok
	}

	return "", false
}
