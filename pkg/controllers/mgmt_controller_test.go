package controllers

import (
	"context"
	"testing"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/pkg/api/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestManagementReconciler(t *testing.T) {
	RegisterFailHandler(Fail)
	defer GinkgoRecover()

	cr := &nvmeshv1.NVMesh{
		Spec: nvmeshv1.NVMeshSpec{
			Management: nvmeshv1.NVMeshManagement{
				Version: "2.0.3-test",
			},
		},
	}
	cr.SetNamespace(TestingNamespace)
	cr.SetName("cluster1")

	var err error

	By("bootstrapping test environment")
	e, err := NewTestEnv()
	Expect(err).To(BeNil())

	r := NVMeshReconciler{
		NVMeshBaseReconciler: NVMeshBaseReconciler{
			Scheme: e.Scheme,
			Log:    logf.Log,
			Client: e.Client,
		},
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

	By("TestManagementReconciler - Reconciling First Attempt")
	_, err = mgmtr.Reconcile(cr, &r)
	Expect(err).To(BeNil())

	By("TestManagementReconciler - Reconciling Second Attempt")
	_, err = mgmtr.Reconcile(cr, &r)
	Expect(err).To(BeNil())
}
