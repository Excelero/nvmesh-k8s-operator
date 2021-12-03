package controllers

import (
	"context"
	goerrors "errors"
	"fmt"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/pkg/api/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func validationError(cr *nvmeshv1.NVMesh, errorMessage string, additionlaDetails string) error {
	return goerrors.New(fmt.Sprintf("Validation failed for NVMesh %s. %s %s", cr.GetName(), errorMessage, additionlaDetails))
}

func (r *NVMeshBaseReconciler) isValid(cr *nvmeshv1.NVMesh) error {
	// NOTE: it is best to apply most of the validation using the OpenAPI with kubebuilder annotations on the NVMesh type

	// If External mongo is deployed, mongo connection address is expected
	if cr.Spec.Management.MongoDB.External && cr.Spec.Management.MongoDB.Address == "" {
		return validationError(cr,
			"Missing MongoDB address for externally deployed MongoDB cluster.",
			"spec.management.mongoDB.external=true but spec.management.mongoDB.address is not specified. When MongoDB is deployed manually the mongoConnection address must be specified in MongoDB.Address. i.e: \"mongo-svc.default.svc.cluster.local:27017\".",
		)
	}

	return nil
}

func (r *NVMeshBaseReconciler) verifySecretExists(secretName string, ns string) error {
	// check if a secret exist
	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{Name: exceleroRegistrySecretName, Namespace: ns}
	err := r.Client.Get(context.TODO(), secretKey, secret)

	if err != nil {
		if k8serrors.IsNotFound(err) {
			r.Log.Info(fmt.Sprintf("DEBUG: secret %s was not found in the namespace %s", secretName, ns))
		} else {
			r.Log.Info(fmt.Sprintf("DEBUG: Error while trying to get secret: %s in namespace %s. error: %s", secretName, ns, err))
		}
	}

	r.Log.Info(fmt.Sprintf("DEBUG: Secret: %s found", secretName))

	return err
}

func (r *NVMeshBaseReconciler) verifyNVMeshSecretsExist(namespace string) {
	// check if the secrets exist in the current namespace
	r.verifySecretExists(exceleroRegistrySecretName, namespace)
	r.verifySecretExists(fileServerSecretName, namespace)
}
