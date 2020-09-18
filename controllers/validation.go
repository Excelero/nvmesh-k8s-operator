package controllers

import nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"

func (r *NVMeshReconciler) IsValid(cr *nvmeshv1.NVMesh) error {
	// NOTE: it is best to apply most of the validation using the OpenAPI with kubebuilder annotations on the NVMesh type
	//TODO: add CustomResource Validation logic
	return nil
}
