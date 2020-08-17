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

func TestNewMgmtStatefulSet(t *testing.T) {
	RegisterFailHandler(Fail)
	defer GinkgoRecover()

	cr := nvmeshv1alpha1.NVMesh{
		Spec: nvmeshv1alpha1.NVMeshSpec{
			Management: nvmeshv1alpha1.NVMeshManagement{
				Version: "2.0.3-4",
			},
		},
	}
	mgmt := mgmtStatefulset{}
	obj, err := mgmt.newObject(&cr)
	Expect(err).To(BeNil())
	Expect(obj).NotTo(BeNil())

	ss := (*obj).(*appsv1.StatefulSet)

	foundImage := ss.Spec.Template.Spec.Containers[0].Image
	Expect(foundImage).To(Equal(mgmtDefaultImageName))
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

	mgmtr := NVMeshMgmtReconciler{
		Scheme: e.Scheme,
		Log:    log.Log,
		Client: e.Client,
	}

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
	err = mgmtr.Reconcile(cr)
	Expect(err).To(BeNil())

	By("Reconciling Second Attempt")
	err = mgmtr.Reconcile(cr)
	Expect(err).To(BeNil())
}
