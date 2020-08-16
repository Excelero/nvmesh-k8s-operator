package importutil

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubectl/pkg/scheme"
)

// func YamlURLToObject(url string) runtime.Object {
// 	resp, err := http.Get(url)
// 	if err != nil {
// 		panic(err.Error())
// 	}

// 	yamlString := make([]byte, resp.ContentLength)
// 	_, err = resp.Body.Read(yamlString)
// 	if err != nil {
// 		panic(err.Error())
// 	}

// 	obj := YamlStringToObject(string(yamlString))
// 	return obj
// }

func YamlFileToObject(filename string) (runtime.Object, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		workdir, _ := os.Getwd()
		log.Println(fmt.Sprintf("Error while trying to read from YAML file. filename: %s, working dir: %s, Err was: %s", filename, workdir, err))
		return nil, err
	}

	yamlString := string(bytes)
	obj, err := YamlStringToObject(yamlString)
	return obj, err
}

func YamlStringToObject(yaml string) (runtime.Object, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(yaml), nil, nil)

	if err != nil {
		log.Println(fmt.Sprintf("Error while decoding YAML object. Err was: %s", err))
	}

	return obj, err
}
