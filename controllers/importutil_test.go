package controllers

import (
	"reflect"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	testScheme  = runtime.NewScheme()
	Codecs      = serializer.NewCodecFactory(testScheme)
	testDecoder = Codecs.UniversalDeserializer()
)

func TestFailureToReadFile(t *testing.T) {
	RegisterFailHandler(Fail)
	defer GinkgoRecover()

	_, err := YamlFileToObjects("file/does/not/exists.yaml", testDecoder)
	Expect(err).NotTo(BeNil())
}

func TestReadingYamlWFileWithMultipleDocuments(t *testing.T) {
	RegisterFailHandler(Fail)
	defer GinkgoRecover()

	objs, err := YamlFileToObjects("../test/samples/multiple_yaml_docs_with_errors.yaml", testDecoder)
	Expect(err).ToNot(BeNil())
	Expect(len(objs)).To(Equal(2))
}

func TestReadingYamlFile(t *testing.T) {
	RegisterFailHandler(Fail)
	defer GinkgoRecover()

	objs, err := YamlFileToObjects("../test/samples/service_account.yaml", testDecoder)
	Expect(err).To(BeNil())
	objectTypeString := reflect.TypeOf(objs[0]).String()
	Expect(objectTypeString).To(Equal("*v1.ServiceAccount"))
}

func TestFailureReadingNonYamlFile(t *testing.T) {
	RegisterFailHandler(Fail)
	defer GinkgoRecover()

	_, err := YamlFileToObjects("../Makefile", testDecoder)
	Expect(err).NotTo(BeNil())
}
