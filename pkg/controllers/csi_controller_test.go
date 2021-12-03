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

var defaultCSIImageName string

func init() {
	defaultCSIImageName = "excelero/nvmesh-csi-driver:v1.1.0"
}

func TestCsiReconciler(t *testing.T) {
	RegisterFailHandler(Fail)
	defer GinkgoRecover()

	cr := &nvmeshv1.NVMesh{
		Spec: nvmeshv1.NVMeshSpec{
			CSI: nvmeshv1.NVMeshCSI{
				Version: "v1.1.0",
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

	csir := NVMeshCSIReconciler(nvmeshr)

	//Start
	csiSS := appsv1.StatefulSet{}
	csiSS.SetNamespace(TestingNamespace)
	csiSS.SetName(csiStatefulSetName)

	err = e.Client.Delete(context.TODO(), &csiSS)
	if err != nil {
		if errors.IsNotFound(err) {
			By("No CSI StatefulSet to delete")
		} else {
			panic("Could not prepare environment - Failed to delete CSI StatefulSet")
		}
	}

	csiDS := appsv1.DaemonSet{}
	csiDS.SetNamespace(TestingNamespace)
	csiDS.SetName(csiDaemonSetName)

	err = e.Client.Delete(context.TODO(), &csiDS)
	if err != nil {
		if errors.IsNotFound(err) {
			By("No CSI DaemonSet to delete")
		} else {
			panic("Could not prepare environment - Failed to delete CSI DaemonSet")
		}
	}

	By("Reconciling First Attempt")
	_, err = csir.Reconcile(cr, &nvmeshr)
	Expect(err).To(BeNil())

	By("Reconciling Second Attempt")
	_, err = csir.Reconcile(cr, &nvmeshr)
	Expect(err).To(BeNil())

	By("Test CSI Reconciler finished")
}

func TestCsiReconcileGenericObject(t *testing.T) {
	RegisterFailHandler(Fail)
	defer GinkgoRecover()

	cr := &nvmeshv1.NVMesh{
		Spec: nvmeshv1.NVMeshSpec{
			CSI: nvmeshv1.NVMeshCSI{
				Version: "csi-test",
			},
		},
	}

	cr.SetNamespace("nvmesh")
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

	csir := NVMeshCSIReconciler(r)
	//Start
	csiSS := appsv1.StatefulSet{}
	csiSS.SetNamespace(TestingNamespace)
	csiSS.SetName(csiStatefulSetName)

	err = e.Client.Delete(context.TODO(), &csiSS)
	if err != nil {
		if errors.IsNotFound(err) {
			By("No CSI StatefulSet to delete")
		} else {
			panic("Could not prepare environment - Failed to delete CSI StatefulSet")
		}
	}

	csiDS := appsv1.DaemonSet{}
	csiDS.SetNamespace(TestingNamespace)
	csiDS.SetName(csiDaemonSetName)

	err = e.Client.Delete(context.TODO(), &csiDS)
	if err != nil {
		if errors.IsNotFound(err) {
			By("No CSI DaemonSet to delete")
		} else {
			panic("Could not prepare environment - Failed to delete CSI DaemonSet")
		}
	}

	By("Make sure exists First Attempt")
	err = r.reconcileYamlObjectsFromFile(cr, csiAssetsLocation+"statefulset_controller.yaml", &csir, false)
	Expect(err).To(BeNil())

	By("Make sure exists Second Attempt")
	err = r.reconcileYamlObjectsFromFile(cr, csiAssetsLocation+"statefulset_controller.yaml", &csir, false)
	Expect(err).To(BeNil())

	By("Make sure *removed* First Attempt")
	err = r.reconcileYamlObjectsFromFile(cr, csiAssetsLocation+"statefulset_controller.yaml", &csir, true)
	Expect(err).To(BeNil())

	By("Make sure *removed* Second Attempt")
	err = r.reconcileYamlObjectsFromFile(cr, csiAssetsLocation+"statefulset_controller.yaml", &csir, true)
	Expect(err).To(BeNil())
}
