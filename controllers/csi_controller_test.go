package controllers

import (
	"context"
	"fmt"
	"testing"

	nvmeshv1alpha1 "excelero.com/nvmesh-k8s-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var defaultCSIImageName string

func init() {
	defaultCSIImageName = "excelero/nvmesh-csi-driver:v1.1.0"
}

func TestCsiReconciler(t *testing.T) {
	RegisterFailHandler(Fail)
	defer GinkgoRecover()

	cr := &nvmeshv1alpha1.NVMesh{
		Spec: nvmeshv1alpha1.NVMeshSpec{
			CSI: nvmeshv1alpha1.NVMeshCSI{
				Version: "v1.1.0",
			},
		},
	}

	var err error

	By("bootstrapping test environment")
	e, err := NewTestEnv()
	Expect(err).To(BeNil())

	cfg, err = testEnv.Start()
	myScheme := runtime.NewScheme()
	c, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	csir := NVMeshCSIReconciler{
		Scheme: myScheme,
		Log:    logf.Log.Logger,
		Client: c,
	}

	//Start
	csiSS := appsv1.StatefulSet{}
	csiSS.SetNamespace(TestingNamespace)
	csiSS.SetName(CSIStatefulSetName)

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
	csiDS.SetName(CSIDaemonSetName)

	err = e.Client.Delete(context.TODO(), &csiDS)
	if err != nil {
		if errors.IsNotFound(err) {
			By("No CSI DaemonSet to delete")
		} else {
			panic("Could not prepare environment - Failed to delete CSI DaemonSet")
		}
	}

	By("Reconciling First Attempt")
	err = csir.Reconcile(cr)
	Expect(err).To(BeNil())

	By("Reconciling Second Attempt")
	err = csir.Reconcile(cr)
	Expect(err).To(BeNil())
}

func TestNewCsiDaemonSetDefaultCR(t *testing.T) {
	RegisterFailHandler(Fail)
	defer GinkgoRecover()

	cr := nvmeshv1alpha1.NVMesh{
		Spec: nvmeshv1alpha1.NVMeshSpec{
			CSI: nvmeshv1alpha1.NVMeshCSI{
				Version: "v1.1.0",
			},
		},
	}
	nd := nodeDriverDaemonSet{}
	obj, err := nd.newObject(&cr)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}

	Expect(err).To(BeNil())
	Expect(obj).NotTo(BeNil())

	daemonset := (*obj).(*appsv1.DaemonSet)

	Expect(daemonset.Spec.Template.Spec.Containers[0].Image).To(Equal(defaultCSIImageName))
}

func TestNewCsiDaemonSetWithImage(t *testing.T) {
	RegisterFailHandler(Fail)
	defer GinkgoRecover()

	imageName := "excelero/nvmesh-csi-driver:my-image"
	cr := nvmeshv1alpha1.NVMesh{
		Spec: nvmeshv1alpha1.NVMeshSpec{
			CSI: nvmeshv1alpha1.NVMeshCSI{
				Image: imageName,
			},
		},
	}
	nd := nodeDriverDaemonSet{}
	obj, err := nd.newObject(&cr)
	Expect(err).To(BeNil())
	Expect(obj).NotTo(BeNil())

	daemonset := (*obj).(*appsv1.DaemonSet)

	Expect(daemonset.Spec.Template.Spec.Containers[0].Image).To(Equal(imageName))
}

func TestNewCsiStatefulSet(t *testing.T) {
	RegisterFailHandler(Fail)
	defer GinkgoRecover()

	cr := nvmeshv1alpha1.NVMesh{}
	nd := ctrlStatefulSet{}
	obj, err := nd.newObject(&cr)
	Expect(err).To(BeNil())
	Expect(obj).NotTo(BeNil())

	ss := (*obj).(*appsv1.StatefulSet)

	Expect(ss.Spec.Template.Spec.Containers[0].Image).To(Equal(defaultCSIImageName))
}
