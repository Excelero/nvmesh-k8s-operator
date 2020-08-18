package controllers

import (
	"context"
	"testing"

	nvmeshv1alpha1 "excelero.com/nvmesh-k8s-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var mgmtDefaultImageName string

func init() {
	mgmtDefaultImageName = "docker.excelero.com/nvmesh-management:2.0.3-4"
}

func TestManagementReconciler(t *testing.T) {
	RegisterFailHandler(Fail)
	defer GinkgoRecover()

	cr := &nvmeshv1alpha1.NVMesh{
		Spec: nvmeshv1alpha1.NVMeshSpec{
			Management: nvmeshv1alpha1.NVMeshManagement{
				Version: "2.0.3-test",
			},
		},
	}

	var err error

	By("bootstrapping test environment")
	e, err := NewTestEnv()
	Expect(err).To(BeNil())

	r := NVMeshReconciler{
		Scheme: e.Scheme,
		Log:    log.Log,
		Client: e.Client,
	}

	mgmtr := NVMeshMgmtReconciler(r)

	// Start
	mgmt := appsv1.StatefulSet{}
	mgmt.SetNamespace(TestingNamespace)
	mgmt.SetName("nvmesh-management")

	err = e.Client.Delete(context.TODO(), &mgmt)
	if err != nil {
		if errors.IsNotFound(err) {
			By("No Managment StatefulSet to delete")
		} else {
			panic("Could not prepare environment - Failed to delete Management StatefulSet")
		}
	}

	By("Reconciling First Attempt")
	err = r.ReconcileComponent(cr, &mgmtr)
	Expect(err).To(BeNil())

	By("Reconciling Second Attempt")
	err = r.ReconcileComponent(cr, &mgmtr)
	Expect(err).To(BeNil())
}
