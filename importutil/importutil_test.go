package importutil

import (
	"reflect"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func testReadingYamlFile(t *testing.T) {
	obj := YamlFileToObject("../test/samples/service_account.yaml")

	objectTypeString := reflect.TypeOf(obj).String()
	Expect(objectTypeString).To(Equal("*v1.ServiceAccount"))

}

func testReadingYamlFromUrl(t *testing.T) {
	obj := YamlURLToObject("https://gist.githubusercontent.com/matthewpalmer/8f04c26705c6b6b8f56e5c397c61d1e8/raw/3921bdcf1eb16ac7a0cdb065dc15e19625896043/pod.yaml")

	objectTypeString := reflect.TypeOf(obj).String()
	Expect(objectTypeString).To(Equal("*v1.ServiceAccount"))

}
func TestImportUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	testReadingYamlFile(t)
	testReadingYamlFromUrl(t)
}
