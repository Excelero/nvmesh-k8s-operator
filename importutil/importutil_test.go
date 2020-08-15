package importutil

import (
	"reflect"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFailureToReadFile(t *testing.T) {
	RegisterFailHandler(Fail)
	_, err := YamlFileToObject("file/does/not/exists.yaml")
	Expect(err).NotTo(BeNil())
}

func TestReadingYamlFile(t *testing.T) {
	RegisterFailHandler(Fail)
	obj, err := YamlFileToObject("../test/samples/service_account.yaml")
	Expect(err).To(BeNil())
	objectTypeString := reflect.TypeOf(obj).String()
	Expect(objectTypeString).To(Equal("*v1.ServiceAccount"))
}

func TestFailureReadingNonYamlFile(t *testing.T) {
	RegisterFailHandler(Fail)
	_, err := YamlFileToObject("../Makefile")
	Expect(err).NotTo(BeNil())
}

// func TestReadingYamlFromUrl(t *testing.T) {
// 	RegisterFailHandler(Fail)
// 	t.Skip("SKIP - issue with ContentLength = -1")

// 	obj := YamlURLToObject("https://raw.githubusercontent.com/kubernetes/website/master/content/en/examples/application/deployment.yaml")

// 	objectTypeString := reflect.TypeOf(obj).String()
// 	Expect(objectTypeString).To(Equal("*v1.ServiceAccount"))
// }
