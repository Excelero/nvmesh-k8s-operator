package controllers

import (
	goerrors "errors"
	"fmt"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
)

func ValidationError(cr *nvmeshv1.NVMesh, errorMessage string, additionlaDetails string) error {
	return goerrors.New(fmt.Sprintf("Validation failed for NVMesh %s. %s %s", cr.GetName(), errorMessage, additionlaDetails))
}

func (r *NVMeshReconciler) IsValid(cr *nvmeshv1.NVMesh) error {
	// NOTE: it is best to apply most of the validation using the OpenAPI with kubebuilder annotations on the NVMesh type

	// If External mongo is deployed, mongo connection address is expected
	if cr.Spec.Management.MongoDB.External && cr.Spec.Management.MongoDB.Address == "" {
		return ValidationError(cr,
			"Missing MongoDB address for externally deployed MongoDB cluster.",
			"spec.management.mongoDB.external=true but spec.management.mongoDB.address is not specified. When MongoDB is deployed manually the mongoConnection address must be specified in MongoDB.Address. i.e: \"mongo-svc.default.svc.cluster.local:27017\".",
		)
	}

	if cr.Spec.Management.MongoDB.External && cr.Spec.Management.MongoDB.UseOperator {
		return ValidationError(cr,
			"Cannot use both mongoDB.external: true AND mongoDB.useOperator: true",
			"If you have a MongoDB cluster already deployed, Please use MongoDB.external: true and supply the mongo connection string in MongoDB.address. MongoDB.useOperator: true will cause the NVMesh Operator to deploy A MongoDB operator for you.",
		)
	}

	return nil
}
