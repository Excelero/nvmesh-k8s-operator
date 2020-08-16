package controllers

import (
	"fmt"
	"testing"

	nvmeshv1alpha1 "excelero.com/nvmesh-k8s-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
)

var defaultCSIImageName string

func init() {
	defaultCSIImageName = "excelero/nvmesh-csi-driver:v1.1.0"
}

func TestNewCsiDaemonSetDefaultCR(t *testing.T) {
	RegisterFailHandler(Fail)
	cr := nvmeshv1alpha1.NVMesh{}
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
	cr := nvmeshv1alpha1.NVMesh{}
	nd := ctrlStatefulSet{}
	obj, err := nd.newObject(&cr)
	Expect(err).To(BeNil())
	Expect(obj).NotTo(BeNil())

	ss := (*obj).(*appsv1.StatefulSet)

	Expect(ss.Spec.Template.Spec.Containers[0].Image).To(Equal(defaultCSIImageName))
}
