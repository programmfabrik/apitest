package template

import (
	"encoding/json"
	"fmt"
	"io"
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
func N(n any) ([]struct{}, error) {
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
func rowsToMap(keyCol, valCol string, rows []map[string]any) (retMap map[string]any, err error) {
	retMap = make(map[string]any)

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

// pivotRows turns rows into columns and columns into rows
func pivotRows(key, typ string, rows []map[string]any) (sheet []map[string]any, err error) {

	getStr := func(data any) (s string) {
		s, _ = data.(string)
		return s
	}

	for _, row := range rows {
		sheetKey := getStr(row[key])
		sheetType := getStr(row[typ])
		if sheetKey == "" || sheetType == "" {
			continue
		}
		switch sheetType {
		case "string", "int64", "float64", "number", "json":
			// supported
		default:
			return nil, fmt.Errorf("type %q not supported", sheetType)
		}

		for kI, vI := range row {
			rowI, _ := strconv.Atoi(getStr(kI))
			if rowI <= 0 {
				continue
			}
			// find row in sheet
			if len(sheet) < rowI {
				for i := len(sheet); i < rowI; i++ {
					sheet = append(sheet, map[string]any{})
				}
			}
			if vI == nil {
				continue
			}
			v := getStr(vI)
			sheetRow := sheet[rowI-1]
			switch sheetType {
			case "string":
				sheetRow[sheetKey] = v
			case "json":
				var i any
				err = json.Unmarshal([]byte(v), &i)
				if err == nil {
					sheetRow[sheetKey] = i
				}
			case "int64":
				sheetRow[sheetKey], _ = strconv.ParseInt(v, 10, 64)
			case "float64":
				sheetRow[sheetKey], _ = strconv.ParseFloat(v, 10)
			case "number":
				var number json.Number
				err = json.Unmarshal([]byte(v), &number)
				if err == nil {
					sheetRow[sheetKey] = number
				}
			}
			sheet[rowI-1] = sheetRow
		}
	}

	return sheet, nil
}

// functions copied from: https://github.com/hashicorp/consul-template/blob/de2ebf4/template_functions.go#L727-L901

// add returns the sum of a and b.
func add(b, a any) (any, error) {
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
func subtract(b, a any) (any, error) {
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
func multiply(b, a any) (any, error) {
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
func divide(b, a any) (any, error) {
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
	file, err := util.OpenFileOrUrl(pathOrURL, rootDir)
	if err != nil {
		return nil, fmt.Errorf("fileReadInternal: %q: %w", pathOrURL, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("fileReadInternal: %q: %w", pathOrURL, err)
	}
	return data, nil
}

// fileRender loads file from path and renders is as Go template passing
// the arguments as ".Param1", ".Param2" into the template.
func loadFileAndRender(rootDir string, loader *Loader) any {
	return func(path string, params ...any) (st string, err error) {
		data, err := fileReadInternal(path, rootDir)
		if err != nil {
			return "", err
		}
		tmplParams := map[string]any{}
		for idx, param := range params {
			tmplParams["Param"+strconv.Itoa(idx+1)] = param
		}
		data, err = loader.Render(data, filepath.Dir(filepath.Join(rootDir, path)), tmplParams)
		if err != nil {
			return "", fmt.Errorf("Render error in file %q: %w", path, err)
		}
		return string(data), nil
	}
}

// fileRender loads file from path and renders is as Go template passing
// the arguments as ".Param1", ".Param2" into the template.
func loadFile(rootDir string) any {
	return func(path string, params ...any) (st string, err error) {
		data, err := fileReadInternal(path, rootDir)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}

// loadFileCSV reads file and parses it in the CSV map. A delimiter can
// be specified. Defaults to ','
func loadFileCSV(rootDir string) any {
	return func(path string, delimiters ...rune) (m []map[string]any, err error) {
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
			return data, fmt.Errorf("CSV map error in file %q: %w", path, err)
		}
		return data, err
	}
}
