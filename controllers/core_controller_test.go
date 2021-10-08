package controllers

import (
	"context"
	"testing"

	nvmeshv1 "excelero.com/nvmesh-k8s-operator/api/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestCoreReconciler(t *testing.T) {
	RegisterFailHandler(Fail)
	defer GinkgoRecover()

	cr := &nvmeshv1.NVMesh{
		Spec: nvmeshv1.NVMeshSpec{
			Core: nvmeshv1.NVMeshCore{
				Version: "2.0.3-dev",
			},
		},
	}

	cr.SetNamespace("nvmesh")
	cr.SetName("cluster1")

	var err error

	By("bootstrapping test environment")
	e, err := NewTestEnv()
	Expect(err).To(BeNil())

	nvmeshr := NVMeshReconciler{
		NVMeshBaseReconciler: NVMeshBaseReconciler{
			Scheme: e.Scheme,
			Log:    logf.Log,
			Client: e.Client,
		},
	}

	corer := NVMeshCoreReconciler(nvmeshr)

	//Start
	userspaceDS := appsv1.DaemonSet{}
	userspaceDS.SetNamespace(TestingNamespace)
	userspaceDS.SetName(coreUserspaceDaemonSetName)

	err = e.Client.Delete(context.TODO(), &userspaceDS)
	if err != nil {
		if errors.IsNotFound(err) {
			By("No Core Userspace Daemonset to delete")
		} else {
			panic("Could not prepare environment - Failed to delete Core Userspace Daemonset")
		}
	}

	By("Reconciling First Attempt")
	err = corer.Reconcile(cr, &nvmeshr)
	Expect(err).To(BeNil())

	By("Reconciling Second Attempt")
	err = corer.Reconcile(cr, &nvmeshr)
	Expect(err).To(BeNil())

	By("Test Core Reconciler finished")
}
