package unittest

import (
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"testing"

	nvmeshv1alpha1 "excelero.com/nvmesh-k8s-operator/nvmesh-operator-go/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/rbac/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestNVMeshType(t *testing.T) {
	o := nvmeshv1alpha1.NVMesh{}
	fmt.Printf("%s\n", strconv.FormatBool(o.Spec.CSI.Deploy))
}

//TODO: move to utils
func yamlFileToObject(filename string) runtime.Object {
	bytes, err := ioutil.ReadFile("service_account.yaml")
	if err != nil {
		panic(err.Error())
	}

	yamlString := string(bytes)
	obj := yamlStringToObject(yamlString)
	return obj
}

//TODO: move to utils
func yamlStringToObject(yaml string) runtime.Object {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, groupVersionKind, err := decode([]byte(yaml), nil, nil)

	if err != nil {
		log.Fatal(fmt.Sprintf("Error while decoding YAML object. Err was: %s", err))
	}

	fmt.Printf("ApiVersion=%s", groupVersionKind)

	return obj
}

//TODO: move to utils
func printYamlObjectType(obj runtime.Object) {
	// now use switch over the type of the object
	// and match each type-case
	switch o := obj.(type) {
	case *v1.Pod:
		// o is a pod
	case *v1beta1.Role:
		// o is the actual role Object with all fields etc
	case *v1beta1.RoleBinding:
	case *v1beta1.ClusterRole:
	case *v1beta1.ClusterRoleBinding:
	case *v1.ServiceAccount:
		fmt.Printf("\nServiceAccount %s Found\n", o.ObjectMeta.Name)
	default:
		//o is unknown for us
		fmt.Printf("o=%s", o)
	}
}

func TestDeserialize(t *testing.T) {
	obj := yamlFileToObject("service_account.yaml")
	printYamlObjectType(obj)
}
