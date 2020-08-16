package controllers

import (
	"path/filepath"
	"testing"

	nvmeshv1alpha1 "excelero.com/nvmesh-k8s-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
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
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	cfg, err = testEnv.Start()
	myScheme := runtime.NewScheme()
	c, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	mgmtr := NVMeshMgmtReconciler{
		Scheme: myScheme,
		Log:    logf.Log.Logger,
		Client: c,
	}

	err = mgmtr.Reconcile(cr)

	Expect(err).To(BeNil())
}
