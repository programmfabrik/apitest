package template

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strconv"

	"github.com/pkg/errors"
	"github.com/programmfabrik/apitest/pkg/lib/csv"
	"github.com/programmfabrik/apitest/pkg/lib/util"
	"github.com/tidwall/gjson"
)

func qjson(path string, json string) string {
	result := gjson.Get(json, path)
	return result.Raw
}

// N returns a slice of n 0-sized elements, suitable for ranging over. (github.com/bradfitz)
func N(n interface{}) ([]struct{}, error) {
	switch v := n.(type) {
	case float64:
		return make([]struct{}, int(v)), nil
	case int64:
		return make([]struct{}, v), nil
	case int:
		return make([]struct{}, v), nil
	}
	return nil, fmt.Errorf("N needs to receive a float64, int, int64. Got: %T", n)
}

// rowsToMap creates a new map, maps "key" column of each to the "value" column of that row. #51482
// if "value" is empty "", the whole row is mapped
func rowsToMap(keyCol, valCol string, rows []map[string]interface{}) (retMap map[string]interface{}, err error) {
	retMap = make(map[string]interface{})

	//If there is no keyCol, return empty map
	if keyCol == "" {
		return
	}

	for _, singlewRow := range rows {
		//Get typed map index
		if singlewRow[keyCol] == nil {
			continue
		}
		mapIndex, ok := singlewRow[keyCol].(string)
		if !ok {
			err = fmt.Errorf("'%v' must be string, as it functions as map index", singlewRow[keyCol])
		}

		if valCol != "" {
			//Normal row
			val := singlewRow[valCol]
			if val == nil {
				val = ""
			}
			retMap[mapIndex] = val
		} else {
			//Row with not valCol
			retMap[mapIndex] = singlewRow
		}
	}

	return
}

// functions copied from: https://github.com/hashicorp/consul-template/blob/de2ebf4/template_functions.go#L727-L901

// add returns the sum of a and b.
func add(b, a interface{}) (interface{}, error) {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	switch bv.Kind() {
	case reflect.String:
		i, err := strconv.Atoi(bv.String())
		if err != nil {
			return nil, err
		}
		b = i
		bv = reflect.ValueOf(b)
	}

	switch av.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return av.Int() + bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Int() + int64(bv.Uint()), nil
		case reflect.Float32, reflect.Float64:
			return float64(av.Int()) + bv.Float(), nil
		default:
			return nil, fmt.Errorf("add: unknown type for %q (%T)", bv, b)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return int64(av.Uint()) + bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Uint() + bv.Uint(), nil
		case reflect.Float32, reflect.Float64:
			return float64(av.Uint()) + bv.Float(), nil
		default:
			return nil, fmt.Errorf("add: unknown type for %q (%T)", bv, b)
		}
	case reflect.Float32, reflect.Float64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return av.Float() + float64(bv.Int()), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Float() + float64(bv.Uint()), nil
		case reflect.Float32, reflect.Float64:
			return av.Float() + bv.Float(), nil
		default:
			return nil, fmt.Errorf("add: unknown type for %q (%T)", bv, b)
		}
	default:
		return nil, fmt.Errorf("add: unknown type for %q (%T)", av, a)
	}
}

// subtract returns the difference of b from a.
func subtract(b, a interface{}) (interface{}, error) {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	switch av.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return av.Int() - bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Int() - int64(bv.Uint()), nil
		case reflect.Float32, reflect.Float64:
			return float64(av.Int()) - bv.Float(), nil
		default:
			return nil, fmt.Errorf("subtract: unknown type for %q (%T)", bv, b)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return int64(av.Uint()) - bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Uint() - bv.Uint(), nil
		case reflect.Float32, reflect.Float64:
			return float64(av.Uint()) - bv.Float(), nil
		default:
			return nil, fmt.Errorf("subtract: unknown type for %q (%T)", bv, b)
		}
	case reflect.Float32, reflect.Float64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return av.Float() - float64(bv.Int()), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Float() - float64(bv.Uint()), nil
		case reflect.Float32, reflect.Float64:
			return av.Float() - bv.Float(), nil
		default:
			return nil, fmt.Errorf("subtract: unknown type for %q (%T)", bv, b)
		}
	default:
		return nil, fmt.Errorf("subtract: unknown type for %q (%T)", av, a)
	}
}

// multiply returns the product of a and b.
func multiply(b, a interface{}) (interface{}, error) {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	switch av.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return av.Int() * bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Int() * int64(bv.Uint()), nil
		case reflect.Float32, reflect.Float64:
			return float64(av.Int()) * bv.Float(), nil
		default:
			return nil, fmt.Errorf("multiply: unknown type for %q (%T)", bv, b)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return int64(av.Uint()) * bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Uint() * bv.Uint(), nil
		case reflect.Float32, reflect.Float64:
			return float64(av.Uint()) * bv.Float(), nil
		default:
			return nil, fmt.Errorf("multiply: unknown type for %q (%T)", bv, b)
		}
	case reflect.Float32, reflect.Float64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return av.Float() * float64(bv.Int()), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Float() * float64(bv.Uint()), nil
		case reflect.Float32, reflect.Float64:
			return av.Float() * bv.Float(), nil
		default:
			return nil, fmt.Errorf("multiply: unknown type for %q (%T)", bv, b)
		}
	default:
		return nil, fmt.Errorf("multiply: unknown type for %q (%T)", av, a)
	}
}

// FROM https://github.com/hashicorp/consul-template/blob/de2ebf4/template_functions.go#L727-L901

// divide returns the division of b from a.
func divide(b, a interface{}) (interface{}, error) {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	switch av.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return av.Int() / bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Int() / int64(bv.Uint()), nil
		case reflect.Float32, reflect.Float64:
			return float64(av.Int()) / bv.Float(), nil
		default:
			return nil, fmt.Errorf("divide: unknown type for %q (%T)", bv, b)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return int64(av.Uint()) / bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Uint() / bv.Uint(), nil
		case reflect.Float32, reflect.Float64:
			return float64(av.Uint()) / bv.Float(), nil
		default:
			return nil, fmt.Errorf("divide: unknown type for %q (%T)", bv, b)
		}
	case reflect.Float32, reflect.Float64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return av.Float() / float64(bv.Int()), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Float() / float64(bv.Uint()), nil
		case reflect.Float32, reflect.Float64:
			return av.Float() / bv.Float(), nil
		default:
			return nil, fmt.Errorf("divide: unknown type for %q (%T)", bv, b)
		}
	default:
		return nil, fmt.Errorf("divide: unknown type for %q (%T)", av, a)
	}
}

func fileReadInternal(pathOrURL, rootDir string) ([]byte, error) {
	_, file, err := util.OpenFileOrUrl(pathOrURL, rootDir)
	if err != nil {
		return nil, errors.Wrapf(err, "fileReadInternal: %q", pathOrURL)
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, errors.Wrapf(err, "fileReadInternal: %q", pathOrURL)
	}
	return data, nil
}

// fileRender loads file from path and renders is as Go template passing
// the arguments as ".Param1", ".Param2" into the template.
func loadFileAndRender(rootDir string, loader *Loader) interface{} {
	return func(path string, params ...interface{}) (st string, err error) {
		data, err := fileReadInternal(path, rootDir)
		if err != nil {
			return "", err
		}
		tmplParams := map[string]interface{}{}
		for idx, param := range params {
			tmplParams["Param"+strconv.Itoa(idx+1)] = param
		}
		data, err = loader.Render(data, filepath.Dir(filepath.Join(rootDir, path)), tmplParams)
		if err != nil {
			return "", errors.Wrapf(err, "Render error in file %q", path)
		}
		return string(data), nil
	}
}

// fileRender loads file from path and renders is as Go template passing
// the arguments as ".Param1", ".Param2" into the template.
func loadFile(rootDir string, loader *Loader) interface{} {
	return func(path string, params ...interface{}) (st string, err error) {
		data, err := fileReadInternal(path, rootDir)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}

// loadFileCSV reads file and parses it in the CSV map. A delimiter can
// be specified. Defaults to ','
func loadFileCSV(rootDir string, loader *Loader) interface{} {
	return func(path string, delimiters ...rune) (m []map[string]interface{}, err error) {
		var delimiter rune
		switch len(delimiters) {
		case 0:
			delimiter = ','
		case 1:
			delimiter = delimiters[0]
		default:
			return nil, errors.New("loadFileCSV: only one or non delimiter parameter allowed")
		}
		fileBytes, err := fileReadInternal(path, rootDir)
		if err != nil {
			return nil, err
		}
		data, err := csv.CSVToMap(fileBytes, delimiter)
		if err != nil {
			return data, errors.Wrapf(err, "CSV map error in file %q", path)
		}
		return data, err
	}
}
