package controllers

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

type YamlDocumentParseError struct {
	Message             string
	YamlContent         string
	DocumentIndexInFile int
	Err                 error
}

func (e *YamlDocumentParseError) Error() string { return e.Message }
func (e *YamlDocumentParseError) Unwrap() error { return e.Err }

type YamlFileParseError struct {
	Message string
	Details interface{}
}

func (e *YamlFileParseError) Error() string { return e.Message }

func YamlFileToObjects(filename string, decoder runtime.Decoder) ([]runtime.Object, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		workdir, _ := os.Getwd()
		err = errors.Wrap(err, fmt.Sprintf("Error while trying to read from YAML file. filename: %s, working dir: %s", filename, workdir))
		return nil, err
	}

	yamlString := string(bytes)
	return YamlStringToObjects(yamlString, decoder)
}

func YamlStringToObjects(yamlString string, decoder runtime.Decoder) ([]runtime.Object, error) {
	yamlDocs := strings.Split(yamlString, "\n---\n")

	var objs []runtime.Object
	var failedDocs []YamlDocumentParseError

	for i, doc := range yamlDocs {
		obj, err := SingleYamlDocStringToObject(doc, decoder)
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

func SingleYamlDocStringToObject(yaml string, decoder runtime.Decoder) (runtime.Object, error) {
	obj, _, err := decoder.Decode([]byte(yaml), nil, nil)

	if err != nil {
		err = errors.Wrap(err, "Error while decoding YAML object")
	}

	return obj, err
}
