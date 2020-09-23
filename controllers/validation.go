package controllers

import (
	goerrors "errors"
	"fmt"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
)

func (r *NVMeshReconciler) IsValid(cr *nvmeshv1.NVMesh) error {
	// NOTE: it is best to apply most of the validation using the OpenAPI with kubebuilder annotations on the NVMesh type

	// If External mongo is deployed, mongo connection address is expected
	if cr.Spec.Management.MongoDB.External && cr.Spec.Management.MongoDB.Address == "" {
		errMsg := "Missing MongoDB address for externally deployed MongoDB cluster."
		additionalDetails := "spec.management.mongoDB.external=true but spec.management.mongoDB.address is not specified. When MongoDB is deployed manually the mongoConnection address must be specified in MongoDB.Address. i.e: \"mongo-svc.default.svc.cluster.local:27017\"."
		return goerrors.New(fmt.Sprintf("Validation failed for NVMesh %s. %s %s", cr.GetName(), errMsg, additionalDetails))
	}

	return nil
}
