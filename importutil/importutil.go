package importutil

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	v1 "k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubectl/pkg/scheme"
)

func YamlURLToObject(url string) runtime.Object {
	resp, err := http.Get(url)
	if err != nil {
		panic(err.Error())
	}

	yamlString := make([]byte, resp.ContentLength)
	_, err = resp.Body.Read(yamlString)
	if err != nil {
		panic(err.Error())
	}

	obj := YamlStringToObject(string(yamlString))
	return obj
}

func YamlFileToObject(filename string) runtime.Object {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err.Error())
	}

	yamlString := string(bytes)
	obj := YamlStringToObject(yamlString)
	return obj
}

func YamlStringToObject(yaml string) runtime.Object {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, groupVersionKind, err := decode([]byte(yaml), nil, nil)

	if err != nil {
		log.Fatal(fmt.Sprintf("Error while decoding YAML object. Err was: %s", err))
	}

	fmt.Printf("ApiVersion=%s", groupVersionKind)

	return obj
}

func PrintYamlObjectType(obj runtime.Object) {
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
