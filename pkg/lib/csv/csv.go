package csv

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Get information
type info struct {
	name   string
	format string
}

func CSVToMap(inputCSV []byte, comma rune) ([]map[string]any, error) {
	if len(inputCSV) == 0 {
		return nil, fmt.Errorf("The given input csv was empty")
	}

	records, err := renderCSV(bytes.NewReader(inputCSV), comma)
	if err != nil {
		return nil, errors.Wrap(err, "CSVToMap.renderCSV")
	}

	records = removeEmptyRowsAndComments(records)

	infos, err := extractHeaderInformation(records[0], records[1])
	if err != nil {
		return nil, err
	}

	output := []map[string]any{}

	//Iterate over the records with skipping the first two lines (as they contain the infos)
	for _, v := range records[2:] {
		tmpRow := make(map[string]any, 0)

		for ki, vi := range v {
			if ki >= len(infos) || infos[ki].format == "SKIP_COLUMN" {
				continue
			}
			ptr := false
			frmt := infos[ki].format
			if strings.HasPrefix(frmt, "*") {
				frmt = frmt[1:]
				ptr = true
			}

			value, err := getTyped(vi, frmt)
			if err != nil {
				return nil, err
			}

			if ptr {
				if vi == "" || value == nil {
					tmpRow[infos[ki].name] = nil // use an untyped nil
				} else {
					tmpRow[infos[ki].name] = &value
				}
			} else {
				tmpRow[infos[ki].name] = value
			}
		}
		output = append(output, tmpRow)
	}

	return output, nil

}

func GenericCSVToMap(inputCSV []byte, comma rune) ([]map[string]any, error) {
	if len(inputCSV) == 0 {
		return nil, fmt.Errorf("The given input csv was empty")
	}

	records, err := renderCSV(bytes.NewReader(inputCSV), comma)
	if err != nil {
		return nil, err
	}

	infos := make([]info, 0)
	for _, v := range records[0] {
		infos = append(infos, info{name: strings.TrimSpace(v)})
	}

	output := []map[string]any{}

	//Iterate over the records with skipping the first two lines (as they contain the infos)
	for _, v := range records[1:] {
		tmpRow := make(map[string]any, 0)

		for ki, vi := range v {
			if ki >= len(infos) {
				continue
			}

			value, err := getTyped(vi, "string")
			if err != nil {
				return nil, err
			}

			tmpRow[infos[ki].name] = value

		}
		output = append(output, tmpRow)
	}

	return output, nil
}

func extractHeaderInformation(names, formats []string) ([]info, error) {
	infos := make([]info, 0)

	for k, v := range names {
		if k >= len(formats) {
			continue
		}

		if !isValidFormat(formats[k]) {
			if strings.TrimSpace(formats[k]) == "" {
				formats[k] = "SKIP_COLUMN"
			} else {
				return nil, fmt.Errorf("'%s' is no valid format", formats[k])
			}
		}
		if strings.TrimSpace(v) == "" {
			formats[k] = "SKIP_COLUMN"
		}

		infos = append(infos, info{format: formats[k], name: strings.TrimSpace(v)})
	}

	return infos, nil
}

func removeEmptyRowsAndComments(input [][]string) (output [][]string) {
	output = make([][]string, 0)
	for _, v := range input {
		empty := true
		for idx, vi := range v {
			if idx == 0 && strings.HasPrefix(vi, "#") {
				// this is a comment line
				break
			}
			if vi != "" {
				empty = false
				break
			}
		}
		if !empty {
			output = append(output, v)
		}
	}

	return output
}

func renderCSV(read io.Reader, comma rune) ([][]string, error) {
	reader := csv.NewReader(read)
	reader.Comma = comma
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	return records, nil
}

func isValidFormat(format string) bool {
	if strings.HasPrefix(format, "*") {
		format = format[1:]
	}
	validFormats := []string{"string", "int64", "int", "float64", "bool"}
	for _, v := range validFormats {
		if format == v || format == v+",array" || format == "json" {
			return true
		}
	}

	return false
}

// getTyped
func getTyped(value, format string) (any, error) {

	switch format {
	case "string":
		return value, nil
	case "int64":
		if value == "" {
			return int64(0), nil
		}
		return strconv.ParseInt(value, 10, 64)
	case "int":
		if value == "" {
			return int(0), nil
		}
		return strconv.Atoi(value)
	case "float64":
		if value == "" {
			return float64(0), nil
		}
		return strconv.ParseFloat(value, 64)
	case "bool":
		if value == "" {
			return false, nil
		}
		return strconv.ParseBool(value)
	case "string,array":
		if value == "" {
			return []string{}, nil
		}

		records, err := renderCSV(strings.NewReader(value), ',')
		if err != nil {

			return nil, err
		}

		//Check if we only have one row. If not return error
		if len(records) > 1 {
			return nil, fmt.Errorf("Only one row is allowed for type 'string,array': %s", value)
		}

		retArray := make([]string, 0)
		for _, v := range records[0] {
			// DEBUG: The previous code would trim values here.
			// Uncomment to debug your CSV...
			// if len(strings.TrimSpace(v)) != len(v) {
			// 	println(fmt.Sprintf("Trimming %s %v", v, records[0]))
			// }
			retArray = append(retArray, v)
		}
		return retArray, nil
	case "int64,array":
		if value == "" {
			return []int64{}, nil
		}

		records, err := renderCSV(strings.NewReader(value), ',')
		if err != nil {
			return nil, err
		}

		//Check if we only have one row. If not return error
		if len(records) > 1 {
			return nil, fmt.Errorf("Only one row is allowed for type 'int64,array'")
		}

		retArray := make([]int64, 0)
		for _, v := range records[0] {
			vi := int64(0)
			if v != "" {
				vi, err = strconv.ParseInt(strings.TrimSpace(v), 10, 64)
				if err != nil {
					return nil, err
				}
			}
			retArray = append(retArray, vi)
		}
		return retArray, nil
	case "float64,array":
		if value == "" {
			return []float64{}, nil
		}

		records, err := renderCSV(strings.NewReader(value), ',')
		if err != nil {
			return nil, err
		}

		//Check if we only have one row. If not return error
		if len(records) > 1 {

			return nil, fmt.Errorf("Only one row is allowed for type 'float64,array'")
		}
		retArray := make([]float64, 0)
		for _, v := range records[0] {
			vi := float64(0)
			if v != "" {
				vi, err = strconv.ParseFloat(strings.TrimSpace(v), 64)
				if err != nil {
					return nil, err
				}
			}
			retArray = append(retArray, vi)
		}
		return retArray, nil
	case "bool,array":
		if value == "" {
			return []bool{}, nil
		}

		records, err := renderCSV(strings.NewReader(value), ',')
		if err != nil {
			return nil, err
		}

		//Check if we only have one row. If not return error
		if len(records) > 1 {
			return nil, fmt.Errorf("Only one row is allowed for type 'bool,array'")
		}

		retArray := make([]bool, 0)
		for _, v := range records[0] {
			retArray = append(retArray, strings.TrimSpace(v) == "true")
		}
		return retArray, nil
	case "json":
		if value == "" {
			return nil, nil
		}
		var data any
		err := json.Unmarshal([]byte(value), &data)
		if err != nil {
			return nil, fmt.Errorf("file_csv: Error in JSON: \"%s\": %s", value, err)
		}
		return data, nil
	default:
		return nil, fmt.Errorf("Given format '%s' not supported for csv usage", format)
	}
}
