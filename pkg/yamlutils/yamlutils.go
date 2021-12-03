package yamlutils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
)

//YamlDocumentParseError - Error while trying to parse a YAML document
type YamlDocumentParseError struct {
	Message             string
	YamlContent         string
	DocumentIndexInFile int
	Err                 error
}

func (e *YamlDocumentParseError) Error() string { return e.Message }
func (e *YamlDocumentParseError) Unwrap() error { return e.Err }

//YamlFileParseError - Error while trying to parse a YAML file
type YamlFileParseError struct {
	Message string
	Details interface{}
}

func (e *YamlFileParseError) Error() string { return e.Message }

//YamlFileToString - Read a YAML file into string
func YamlFileToString(filename string) (string, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		workdir, _ := os.Getwd()
		err = errors.Wrap(err, fmt.Sprintf("Error while trying to read from YAML file. filename: %s, working dir: %s", filename, workdir))
		return "", err
	}

	yamlString := string(bytes)
	return yamlString, nil
}

func UnstructuredToString(obj unstructured.Unstructured) string {
	buf := bytes.NewBufferString("")
	enc := json.NewEncoder(buf)
	enc.SetIndent("", "    ")
	enc.Encode(obj)
	return buf.String()
}

//YamlFileToUnstructured - Read a YAML file to unstructured type
func YamlFileToUnstructured(filename string) (*unstructured.Unstructured, *schema.GroupVersionKind, error) {
	yamlString, err := YamlFileToString(filename)
	if err != nil {
		panic(err)
	}

	obj := &unstructured.Unstructured{}

	// decode YAML into unstructured.Unstructured
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, gvk, err := dec.Decode([]byte(yamlString), nil, obj)

	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Error while trying to read from YAML file. filename: %s, content: %s", filename, yamlString))
		return nil, nil, err
	}

	return obj, gvk, err
}

//YamlFileToObjects - Read a YAML file to typed objects
func YamlFileToObjects(filename string, decoder runtime.Decoder) ([]client.Object, error) {
	yamlString, err := YamlFileToString(filename)
	if err != nil {
		return nil, err
	}

	return yamlStringToObjects(yamlString, decoder)
}

func yamlStringToObjects(yamlString string, decoder runtime.Decoder) ([]client.Object, error) {
	yamlDocs := strings.Split(yamlString, "\n---\n")

	var objs []client.Object
	var failedDocs []YamlDocumentParseError

	for i, doc := range yamlDocs {
		obj, err := singleYamlDocStringToObject(doc, decoder)
		if err != nil {
			newError := YamlDocumentParseError{Message: "Failed to parse yaml document", YamlContent: doc, DocumentIndexInFile: i, Err: err}
			failedDocs = append(failedDocs, newError)
		} else {
			objs = append(objs, obj)
		}
	}

	if len(failedDocs) > 0 {
		msg := fmt.Sprintf("yaml with multiple documents had %d documents that failed to parse.", len(failedDocs))
		newYamlError := YamlFileParseError{Message: msg, Details: failedDocs}
		return objs, &newYamlError
	}
	return objs, nil
}

func singleYamlDocStringToObject(yaml string, decoder runtime.Decoder) (client.Object, error) {
	obj, _, err := decoder.Decode([]byte(yaml), nil, nil)

	if err != nil {
		err = errors.Wrap(err, "Error while decoding YAML object")
	}

	return (obj).(client.Object), err
}
