package reflectutils

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

func getProp(obj reflect.Value, fieldName string) (error, *reflect.Value) {
	if _, ok := obj.Type().FieldByName(fieldName); !ok {
		return fmt.Errorf("Object does not have a field %s", fieldName), nil
	}

	f := obj.FieldByName(fieldName)
	return nil, &f
}

func getPropAtPath(obj interface{}, path string) (error, *reflect.Value) {
	pathParts := strings.Split(path, ".")

	var err error
	var index *int
	currentPath := ""
	ValueOfObj := reflect.ValueOf(obj)
	walkObj := &ValueOfObj
	for _, field := range pathParts {
		currentPath = currentPath + "." + field
		isArray := false
		// Get index and remove array notation from field name
		re := regexp.MustCompile(`\[\d*\]`)
		match := re.FindString(field)

		if match != "" {
			indexVal, err := strconv.ParseInt(match[1:len(match)-1], 0, 32)
			tmpInt := int(indexVal)
			index = &tmpInt
			if err != nil {
				return err, nil
			}
			isArray = true
			field = field[:len(field)-len(match)]
		}

		if walkObj.Kind() == reflect.Ptr {
			tmpPtrValue := walkObj.Elem()
			walkObj = &tmpPtrValue
		}
		err, walkObj = getProp(*walkObj, field)
		if err != nil {
			return fmt.Errorf("Object %+v does not have field %s in the path %s", obj, field, currentPath), nil
		}

		if isArray {
			f := walkObj.Index(*index)
			walkObj = &f
		}
	}

	if walkObj.Kind() == reflect.Ptr {
		tmpPtrValue := walkObj.Elem()
		walkObj = &tmpPtrValue
	}

	return nil, walkObj
}

type CompareFieldsResult struct {
	Equals bool
	Path   string
	Value1 interface{}
	Value2 interface{}
}

func CompareObjectsAtJsonPath(obj1 interface{}, obj2 interface{}, path string) (error, CompareFieldsResult) {
	err1, value1 := getPropAtPath(obj1, path)
	if err1 != nil {
		return err1, CompareFieldsResult{}
	}

	err2, value2 := getPropAtPath(obj2, path)
	if err2 != nil {
		return err2, CompareFieldsResult{}
	}

	if value1.Type() != value2.Type() || value1.Interface() != value2.Interface() {
		return nil, CompareFieldsResult{Equals: false, Path: path, Value1: value1.Interface(), Value2: value2.Interface()}
	}
	return nil, CompareFieldsResult{Equals: true}
}

func CompareFieldsInTwoObjects(obj1 interface{}, obj2 interface{}, fieldPaths []string) (error, CompareFieldsResult) {
	for _, path := range fieldPaths {
		err, result := CompareObjectsAtJsonPath(obj1, obj2, path)
		if err != nil || !result.Equals {
			return err, result
		}
	}

	return nil, CompareFieldsResult{Equals: true}
}
