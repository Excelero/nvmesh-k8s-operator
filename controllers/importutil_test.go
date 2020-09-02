package controllers

import (
	"reflect"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFailureToReadFile(t *testing.T) {
	RegisterFailHandler(Fail)
	defer GinkgoRecover()

	_, err := YamlFileToObjects("file/does/not/exists.yaml")
	Expect(err).NotTo(BeNil())
}

func TestReadingYamlWFileWithMultipleDocuments(t *testing.T) {
	RegisterFailHandler(Fail)
	defer GinkgoRecover()

	objs, err := YamlFileToObjects("../test/samples/multiple_yaml_docs_with_errors.yaml")
	Expect(err).ToNot(BeNil())
	Expect(len(objs)).To(Equal(2))
}

func TestReadingYamlFile(t *testing.T) {
	RegisterFailHandler(Fail)
	defer GinkgoRecover()

	objs, err := YamlFileToObjects("../test/samples/service_account.yaml")
	Expect(err).To(BeNil())
	objectTypeString := reflect.TypeOf(objs[0]).String()
	Expect(objectTypeString).To(Equal("*v1.ServiceAccount"))
}

func TestFailureReadingNonYamlFile(t *testing.T) {
	RegisterFailHandler(Fail)
	defer GinkgoRecover()

	_, err := YamlFileToObjects("../Makefile")
	Expect(err).NotTo(BeNil())
}
