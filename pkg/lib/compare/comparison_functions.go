package compare

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"strconv"
	"strings"

	"github.com/programmfabrik/apitest/pkg/lib/jsutil"
	"github.com/programmfabrik/golib"
)

type ComparisonContext struct {
	depth          int64
	orderMatters   bool
	noExtra        bool
	elementNoExtra bool
	mustExist      bool
	elementCount   *int64
	isString       bool
	isNumber       bool
	isArray        bool
	mustNotExist   bool
	isObject       bool
	isBool         bool
	numberGT       *jsutil.Number
	numberGE       *jsutil.Number
	numberLT       *jsutil.Number
	numberLE       *jsutil.Number
	regexMatch     *jsutil.String
	regexMatchNot  *jsutil.String
	startsWith     *jsutil.String
	endsWith       *jsutil.String
	notEqualNull   bool
	notEqual       *any
}

var controlKeyRegex = regexp.MustCompile(`(?P<Key>.*?):control`)

func fillComparisonContext(in jsutil.Object) (out *ComparisonContext, err error) {
	out = &ComparisonContext{}

	for k, v := range in {
		switch k {
		case "depth":
			tV, err := getAsInt64(v)
			if err != nil {
				return out, fmt.Errorf("depth is no int64: %w", err)

			}
			out.depth = tV
		case "order_matters":
			tV, ok := v.(bool)
			if !ok {
				err = errors.New("order_matters is no bool")
				return

			}
			out.orderMatters = tV
		case "no_extra":
			tV, ok := v.(bool)
			if !ok {
				err = errors.New("no_extra is no bool")
				return
			}
			out.noExtra = tV
		case "element_no_extra":
			tV, ok := v.(bool)
			if !ok {
				err = errors.New("element_no_extra is no bool")
				return

			}
			out.elementNoExtra = tV
		case "must_exist":
			tV, ok := v.(bool)
			if !ok {
				err = errors.New("must_exist is no bool")
				return

			}
			out.mustExist = tV
		case "element_count":
			tV, err := getAsInt64(v)
			if err != nil {
				return out, fmt.Errorf("element_count is no int64: %w", err)

			}
			out.elementCount = &tV
		case "is_string":
			tV, ok := v.(bool)
			if !ok {
				err = errors.New("is_string is no bool")
				return

			}
			out.isString = tV
		case "match":
			tV, ok := v.(string)
			if !ok {
				err = errors.New("match is no string")
				return
			}
			out.regexMatch = &tV
		case "not_match":
			tV, ok := v.(string)
			if !ok {
				err = errors.New("not_match is no string")
				return

			}
			out.regexMatchNot = &tV
		case "starts_with":
			tV, ok := v.(string)
			if !ok || tV == "" {
				err = errors.New("starts_with must be a string with length > 0")
				return

			}
			out.startsWith = &tV
		case "ends_with":
			tV, ok := v.(string)
			if !ok || tV == "" {
				err = errors.New("ends_with must be a string with length > 0")
				return

			}
			out.endsWith = &tV
		case "is_number":
			tV, ok := v.(bool)
			if !ok {
				err = errors.New("is_number is no bool")
				return

			}
			out.isNumber = tV
		case "is_array":
			tV, ok := v.(bool)
			if !ok {
				err = errors.New("is_array is no bool")
				return

			}
			out.isArray = tV
		case "must_not_exist":
			tV, ok := v.(bool)
			if !ok {
				err = errors.New("must_not_exist is no bool")
				return

			}
			out.mustNotExist = tV
		case "is_object":
			tV, ok := v.(bool)
			if !ok {
				err = errors.New("is_object is no bool")
				return

			}
			out.isObject = tV
		case "is_bool":
			tV, ok := v.(bool)
			if !ok {
				err = errors.New("is_bool is no bool")
				return

			}
			out.isBool = tV
		case "number_gt":
			// Number must be bigger
			tV, ok := v.(jsutil.Number)
			if !ok {
				err = errors.New("number_gt is no number")
				return

			}
			out.numberGT = &tV
			out.isNumber = true
		case "number_ge":
			// Number must be equal or bigger
			tV, ok := v.(jsutil.Number)
			if !ok {
				err = errors.New("number_gt is no number")
				return

			}
			out.numberGE = &tV
			out.isNumber = true
		case "number_lt":
			// Number must be smaller
			tV, ok := v.(jsutil.Number)
			if !ok {
				err = errors.New("number_lt is no number")
				return

			}
			out.numberLT = &tV
			out.isNumber = true
		case "number_le":
			// Number must be equal or smaller
			tV, ok := v.(jsutil.Number)
			if !ok {
				err = errors.New("number_le is no number")
				return
			}
			out.numberLE = &tV
			out.isNumber = true
		case "not_equal":
			if v == nil {
				out.notEqualNull = true
			} else {
				// only allow the not_equal for types string, number, bool
				switch getJsonType(v) {
				case "String", "Number", "Bool", "Array", "JsonNumber":
					out.notEqual = &v
				default:
					err = fmt.Errorf("not_equal has invalid type %s", getJsonType(v))
					return
				}
			}

		default:
			err = fmt.Errorf("unknown key in control: %s", k)
			return
		}
	}

	return
}

// objectComparsion checks if two objects are equal
// hereby we also check our control structures and the noExtra parameter. If noExtra is true it is not allowed to have
// elements than set
func objectComparison(left, right jsutil.Object, noExtra bool) (res CompareResult, err error) {
	var (
		rv, lv   any
		rOK, lOK bool
		k        string
	)

	res.Equal = true

	takenInRight := make(map[string]bool)
	takenInLeft := make(map[string]bool)

	// Iterate over normal fields
	leftCopy := maps.Clone(left)
	for ck, cv := range leftCopy {
		if takenInLeft[ck] {
			continue
		}
		control := &ComparisonContext{}

		// Check which type of key we have
		if strings.HasSuffix(ck, ":control") {
			// We have a control key
			k = controlKeyRegex.FindStringSubmatch(ck)[1]
			if !takenInRight[k] {
				rv, rOK = right[k]
			}
			lv, lOK = leftCopy[k]

			takenInLeft[k] = true
			takenInRight[k] = true

			cvObj, ok := cv.(jsutil.Object)
			if ok {
				control, err = fillComparisonContext(cvObj)
				if err != nil {
					return res, err
				}
			} else {
				return res, fmt.Errorf("%s:control must be an object", k)
			}
			delete(leftCopy, ck)
		} else {
			// Normal key
			k = ck
			if !takenInRight[k] {
				rv, rOK = right[k]
			}
			lv = cv
			lOK = true

			takenInLeft[k] = true
			takenInRight[k] = true

			leftObj, ok := leftCopy[k+":control"].(jsutil.Object)
			if ok {
				iControl, err := fillComparisonContext(leftObj)
				if err != nil {
					return res, err
				}
				if iControl != nil {
					control = iControl
					delete(leftCopy, k+":control")
				}
			}
		}

		// If a value is present, it must exist
		if lOK {
			control.mustExist = true
		}

		// Check for the given key functions
		err := keyChecks(rv, rOK, *control)
		if err != nil {
			res.Failures = append(res.Failures, compareFailure{Key: k, Message: err.Error()})
			res.Equal = false
			// There is no use in checking the equality of the value if the preconditions do not work
			continue
		}

		// If we have a left value, check if it is the same as right
		if lOK {
			tmp, err := JsonEqual(lv, rv, *control)
			if err != nil {
				return CompareResult{}, err
			}

			for ik, iv := range tmp.Failures {
				if iv.Key == "" {
					tmp.Failures[ik].Key = k
				} else {
					if []rune(iv.Key)[0] == '[' {
						tmp.Failures[ik].Key = k + iv.Key
					} else {
						tmp.Failures[ik].Key = k + "." + iv.Key
					}
				}
			}
			res.Failures = append(res.Failures, tmp.Failures...)
			res.Equal = res.Equal && tmp.Equal
		}
	}

	if noExtra {
		for k := range right {
			if !takenInRight[k] {
				res.Failures = append(res.Failures, compareFailure{Key: "", Message: "extra elements found in object"})
				res.Equal = false
				return
			}
		}
	}
	return res, nil
}

// ArrayComparison offerst the compare feature to other packages, with the standard behavior
// noExtra=false, orderMatter=false
func ArrayComparison(left, right jsutil.Array) (res CompareResult, err error) {
	return arrayComparison(left, right, ComparisonContext{}, ComparisonContext{})
}

// arrayComparison makes a simple array comparison by either running trough both arrays with the same key (orderMaters)
// or taking a value from the left array and search it in the right one
func arrayComparison(left, right jsutil.Array, currControl ComparisonContext, nextControl ComparisonContext) (res CompareResult, err error) {
	res.Equal = true

	if len(left) > len(right) {
		res.Equal = false

		leftJson, err := golib.JsonBytesIndent(left, "", "  ")
		if err != nil {
			return CompareResult{}, fmt.Errorf("could not marshal expected array: %w", err)
		}
		rightJson, err := golib.JsonBytesIndent(right, "", "  ")
		if err != nil {
			return CompareResult{}, fmt.Errorf("could not marshal actual array: %w", err)
		}

		res.Failures = append(res.Failures, compareFailure{"", fmt.Sprintf("[arrayComparison] length of expected response (%d) > length of actual response (%d)\nExpected response:\n%s\nActual response:\n%s\n", len(left), len(right), string(leftJson), string(rightJson))})
		return res, nil
	}

	takenInRight := make(map[int]bool)
	lastPositionFromLeftInRight := -1

	for lk, lv := range left {
		if currControl.orderMatters {
			for rk, rv := range right {
				if rk <= lastPositionFromLeftInRight {
					continue
				}
				tmp, err := JsonEqual(lv, rv, nextControl)
				if err != nil {
					return CompareResult{}, err
				}
				if tmp.Equal {
					takenInRight[lk] = true
					lastPositionFromLeftInRight = rk
					break
				}
			}
			if !takenInRight[lk] {
				key := fmt.Sprintf("[%d]", lk)
				elStr := fmt.Sprintf("%v", lv)
				elBytes, err := jsutil.Marshal(lv)
				if err == nil {
					elStr = string(elBytes)
				}
				res.Failures = append(res.Failures, compareFailure{key, fmt.Sprintf("element %s not found in array in proper order", elStr)})
				res.Equal = false
			}
		} else {
			found := false
			allTmpFailures := make([]compareFailure, 0)

			for rk, rv := range right {
				if takenInRight[rk] {
					continue
				}

				// We need to check the left interface against the right one multiple times
				// JsonEqual modifies such interface (it deletes it afterwards)
				// Therefore we need a copy of it for this case

				tmp, err := JsonEqual(lv, rv, nextControl)
				if err != nil {
					return CompareResult{}, err
				}

				if tmp.Equal {
					// Found an element fitting
					found = true
					takenInRight[rk] = true
					break
				}

				allTmpFailures = append(allTmpFailures, tmp.Failures...)
			}

			if !found {
				for _, v := range allTmpFailures {
					key := fmt.Sprintf("[%d].%s", lk, v.Key)
					if v.Key == "" {
						key = fmt.Sprintf("[%d]", lk)
					}
					res.Failures = append(res.Failures, compareFailure{key, v.Message})
				}
				res.Equal = false
			}
		}

	}

	if currControl.noExtra {
		for k := range right {
			if !takenInRight[k] {
				res.Failures = append(res.Failures, compareFailure{Key: "", Message: "extra elements found in array"})
				res.Equal = false
				return
			}
		}
	}

	return res, nil
}

func ObjectEqualWithControl(left, right jsutil.Object, control ComparisonContext) (res CompareResult, err error) {
	return objectComparison(left, right, control.noExtra)
}

func ArrayEqualWithControl(left, right jsutil.Array, control ComparisonContext) (res CompareResult, err error) {
	nextControl := ComparisonContext{
		noExtra: control.elementNoExtra,
		depth:   -9999,
	}
	if control.depth >= -1 {
		if control.depth > 0 {
			nextControl.depth = control.depth - 1
		} else if control.depth < 0 {
			nextControl.depth = control.depth
		}
	}
	if nextControl.depth >= -1 {
		nextControl.noExtra = nextControl.noExtra || control.noExtra
		nextControl.orderMatters = control.orderMatters
	}
	return arrayComparison(left, right, control, nextControl)
}

func keyChecks(right any, rOK bool, control ComparisonContext) (err error) {
	if control.isString {
		if right == nil {
			return errors.New("== nil but should exist")
		}
		jsonType := getJsonType(right)
		if jsonType != "String" {
			return fmt.Errorf("should be 'String' but is '%s'", jsonType)
		}
	} else if control.isNumber {
		if right == nil {
			return errors.New("== nil but should exist")
		}
		jsonType := getJsonType(right)
		if jsonType != "JsonNumber" && jsonType != "Number" {
			return fmt.Errorf("should be 'JsonNumber' or 'Number' but is '%s'", jsonType)
		}
	} else if control.isBool {
		if right == nil {
			return errors.New("== nil but should exist")
		}
		jsonType := getJsonType(right)
		if jsonType != "Bool" {
			return fmt.Errorf("should be 'Bool' but is '%s'", jsonType)
		}
	} else if control.isArray {
		if right == nil {
			return errors.New("== nil but should exist")
		}
		jsonType := getJsonType(right)
		if jsonType != "Array" {
			return fmt.Errorf("should be 'Array' but is '%s'", jsonType)
		}
	} else if control.isObject {
		if right == nil {
			return errors.New("== nil but should exist")
		}
		jsonType := getJsonType(right)
		if jsonType != "Object" {
			return fmt.Errorf("should be 'Object' but is '%s'", jsonType)
		}
	}

	// Check if exists
	if !rOK && control.mustExist {
		return errors.New("was not found, but should exist")
	}

	if rOK && control.mustNotExist {
		return errors.New("was found, but should NOT exist")
	}

	// Check for array length
	if control.elementCount != nil {
		jsonType := getJsonType(right)
		if jsonType != "Array" {
			return fmt.Errorf("should be 'Array' but is '%s'", jsonType)
		}

		rightArray := right.(jsutil.Array)
		rightLen := int64(len(rightArray))
		if rightLen != *control.elementCount {
			return fmt.Errorf("length of the actual response array %d != %d expected length", rightLen, *control.elementCount)
		}
	}

	// Check for number range
	if control.numberGE != nil {
		rightNumber := right.(jsutil.Number)
		if !(rightNumber >= *control.numberGE) {
			return fmt.Errorf("actual number %s is not equal or greater than %s", rightNumber, *control.numberGE)
		}
	}
	if control.numberGT != nil {
		rightNumber := right.(jsutil.Number)
		if !(rightNumber > *control.numberGT) {
			return fmt.Errorf("actual number %s is not greater than %s", rightNumber, *control.numberGT)
		}
	}
	if control.numberLE != nil {
		rightNumber := right.(jsutil.Number)
		if !(rightNumber <= *control.numberLE) {
			return fmt.Errorf("actual number %s is not equal or less than %s", rightNumber, *control.numberLE)
		}
	}
	if control.numberLT != nil {
		rightNumber := right.(jsutil.Number)
		if !(rightNumber < *control.numberLT) {
			return fmt.Errorf("actual number %s is not less than %s", rightNumber, *control.numberLT)
		}
	}

	var (
		matchS    string
		doesMatch bool
	)

	// Check if string matches regex
	if control.regexMatch != nil {
		jsonType := getJsonType(right)
		if jsonType != "String" {
			matchS = fmt.Sprintf("%v", right)
		} else {
			matchS = right.(jsutil.String)
		}
		doesMatch, err = regexp.Match(*control.regexMatch, []byte(matchS))
		if err != nil {
			return fmt.Errorf("could not match regex %q: %w", *control.regexMatch, err)
		}
		if !doesMatch {
			return fmt.Errorf("%T %q does not match regex %q", right, matchS, *control.regexMatch)
		}
	}

	// Check if string does not match regex
	if control.regexMatchNot != nil {
		jsonType := getJsonType(right)
		if jsonType != "String" {
			matchS = fmt.Sprintf("%v", right)
		} else {
			matchS = right.(jsutil.String)
		}
		doesMatch, err = regexp.Match(*control.regexMatchNot, []byte(matchS))
		if err != nil {
			return fmt.Errorf("could not match regex %q: %w", *control.regexMatchNot, err)
		}
		if doesMatch {
			return fmt.Errorf("matches regex %q but should not match", *control.regexMatchNot)
		}
	}

	// Check if string starts or ends with another string
	if control.startsWith != nil {
		jsonType := getJsonType(right)
		if jsonType != "String" {
			return fmt.Errorf("should be 'String' for starts_with but is '%s'", jsonType)
		}

		if !strings.HasPrefix(right.(jsutil.String), *control.startsWith) {
			return fmt.Errorf("does not start with '%s'", *control.startsWith)
		}
	}
	if control.endsWith != nil {
		jsonType := getJsonType(right)
		if jsonType != "String" {
			return fmt.Errorf("should be 'String' for ends_with but is '%s'", jsonType)
		}

		if !strings.HasSuffix(right.(jsutil.String), *control.endsWith) {
			return fmt.Errorf("does not end with '%s'", *control.endsWith)
		}
	}

	if control.notEqualNull {
		if right == nil {
			return errors.New("is null")
		}
	}
	if control.notEqual != nil {
		controlJsonType := getJsonType(*control.notEqual)
		jsonType := getJsonType(right)
		// only compare value if type is equal and a low level json type (string, number, bool) or array
		// different type is always not_equal
		if jsonType == controlJsonType {
			switch jsonType {
			case "Array":
				leftMar, err := jsutil.Marshal((*control.notEqual).(jsutil.Array))
				if err != nil {
					return fmt.Errorf("could not marshal left part: %w", err)
				}
				rightMar, err := jsutil.Marshal(right.(jsutil.Array))
				if err != nil {
					return fmt.Errorf("could not marshal right part: %w", err)
				}
				if string(leftMar) == string(rightMar) {
					return fmt.Errorf("is equal to %s %s, should not be equal", jsonType, string(leftMar))
				}
			case "String":
				if (*control.notEqual).(jsutil.String) == right.(jsutil.String) {
					return fmt.Errorf("is equal to %s '%s', should not be equal", jsonType, (*control.notEqual).(jsutil.String))
				}
			case "Number":
				if *control.notEqual == right {
					return fmt.Errorf("is equal to %s %v, should not be equal", jsonType, *control.notEqual)
				}
			case "JsonNumber":
				if jsutil.NumberEqual((*control.notEqual).(jsutil.Number), right.(jsutil.Number)) {
					return fmt.Errorf("expected %v, got %v", right, *control.notEqual)
				}
			case "Bool":
				if (*control.notEqual).(jsutil.Bool) == right.(jsutil.Bool) {
					return fmt.Errorf("is equal to %s %v, should not be equal", jsonType, (*control.notEqual).(jsutil.Bool))
				}
			}
		}
	}

	return nil
}

func getJsonType(value any) string {
	switch value.(type) {
	case jsutil.Object:
		return "Object"
	case jsutil.Array:
		return "Array"
	case jsutil.String:
		return "String"
	case int, float64:
		return "Number"
	case jsutil.Number:
		return "JsonNumber"
	case jsutil.Bool:
		return "Bool"
	default:
		return "No JSON Type: " + fmt.Sprintf("%v[%T]", value, value)
	}
}

func getAsInt64(value any) (n int64, err error) {
	switch t := value.(type) {
	case int64:
		return t, nil
	case int:
		return int64(t), nil
	case float32, float64:
		return strconv.ParseInt(fmt.Sprintf("%.0f", t), 10, 64)
	case jsutil.Number:
		return t.Int64()
	default:
		return 0, fmt.Errorf("'%v' has no valid json number type", value)
	}
}
