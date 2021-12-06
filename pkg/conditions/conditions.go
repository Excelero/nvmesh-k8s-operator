/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package meta

import (
	"time"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/pkg/api/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SetStatusCondition sets the corresponding condition in conditions to newCondition.
// conditions must be non-nil.
// 1. if the condition of the specified type already exists (all fields of the existing condition are updated to
//    newCondition, LastTransitionTime is set to now if the new status differs from the old status)
// 2. if a condition of the specified type does not exist (LastTransitionTime is set to now() if unset, and newCondition is appended)
func SetStatusCondition(conditions *[]nvmeshv1.ClusterCondition, newCondition *nvmeshv1.ClusterCondition) {
	if conditions == nil {
		return
	}
	existingCondition := FindStatusCondition(*conditions, newCondition.Type)
	if existingCondition == nil {
		if newCondition.LastTransitionTime.IsZero() {
			newCondition.LastTransitionTime = metav1.NewTime(time.Now())
		}
		*conditions = append(*conditions, *newCondition)
		return
	}

	if existingCondition.Status != newCondition.Status {
		existingCondition.Status = newCondition.Status
		if !newCondition.LastTransitionTime.IsZero() {
			existingCondition.LastTransitionTime = newCondition.LastTransitionTime
		} else {
			existingCondition.LastTransitionTime = metav1.NewTime(time.Now())
		}
	}

	existingCondition.Reason = newCondition.Reason
	existingCondition.Message = newCondition.Message
}

// RemoveStatusCondition removes the corresponding conditionType from conditions.
// conditions must be non-nil.
func RemoveStatusCondition(conditions *[]nvmeshv1.ClusterCondition, conditionType nvmeshv1.ClusterConditionType) {
	if conditions == nil {
		return
	}
	newConditions := make([]nvmeshv1.ClusterCondition, 0, len(*conditions)-1)
	for _, condition := range *conditions {
		if condition.Type != conditionType {
			newConditions = append(newConditions, condition)
		}
	}

	*conditions = newConditions
}

// FindStatusCondition finds the conditionType in conditions.
func FindStatusCondition(conditions []nvmeshv1.ClusterCondition, conditionType nvmeshv1.ClusterConditionType) *nvmeshv1.ClusterCondition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}

	return nil
}

// IsStatusConditionTrue returns true when the conditionType is present and set to `nvmeshv1.ClusterConditionTrue`
func IsStatusConditionTrue(conditions []nvmeshv1.ClusterCondition, conditionType nvmeshv1.ClusterConditionType) bool {
	return IsStatusConditionPresentAndEqual(conditions, conditionType, nvmeshv1.ConditionTrue)
}

// IsStatusConditionFalse returns true when the conditionType is present and set to `nvmeshv1.ClusterConditionFalse`
func IsStatusConditionFalse(conditions []nvmeshv1.ClusterCondition, conditionType nvmeshv1.ClusterConditionType) bool {
	return IsStatusConditionPresentAndEqual(conditions, conditionType, nvmeshv1.ConditionFalse)
}

// IsStatusConditionPresentAndEqual returns true when conditionType is present and equal to status.
func IsStatusConditionPresentAndEqual(conditions []nvmeshv1.ClusterCondition, conditionType nvmeshv1.ClusterConditionType, status v1.ConditionStatus) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Status == status
		}
	}
	return false
}