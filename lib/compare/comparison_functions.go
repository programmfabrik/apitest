package compare

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/programmfabrik/fylr-apitest/lib/util"
)

type ComparisonContext struct {
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
}

func fillComparisonContext(in util.JsonObject) (out *ComparisonContext, err error) {
	out = &ComparisonContext{}

	for k, v := range in {
		switch k {
		case "order_matters":
			tV, ok := v.(bool)
			if !ok {
				err = fmt.Errorf("order_matters is no bool")
				return

			}
			out.orderMatters = tV
		case "no_extra":
			tV, ok := v.(bool)
			if !ok {
				err = fmt.Errorf("no_extra is no bool")
				return
			}
			out.noExtra = tV
		case "element_no_extra":
			tV, ok := v.(bool)
			if !ok {
				err = fmt.Errorf("element_no_extra is no bool")
				return

			}
			out.elementNoExtra = tV
		case "must_exist":
			tV, ok := v.(bool)
			if !ok {
				err = fmt.Errorf("must_exist is no bool")
				return

			}
			out.mustExist = tV
		case "element_count":
			tV, err := getAsInt64(v)
			if err != nil {
				return out, fmt.Errorf("element_count is no int64: %s", err)

			}
			out.elementCount = &tV
		case "is_string":
			tV, ok := v.(bool)
			if !ok {
				err = fmt.Errorf("is_string is no bool")
				return

			}
			out.isString = tV
		case "is_number":
			tV, ok := v.(bool)
			if !ok {
				err = fmt.Errorf("is_number is no bool")
				return

			}
			out.isNumber = tV
		case "is_array":
			tV, ok := v.(bool)
			if !ok {
				err = fmt.Errorf("is_array is no bool")
				return

			}
			out.isArray = tV
		case "must_not_exist":
			tV, ok := v.(bool)
			if !ok {
				err = fmt.Errorf("must_not_exist is no bool")
				return

			}
			out.mustNotExist = tV
		case "is_object":
			tV, ok := v.(bool)
			if !ok {
				err = fmt.Errorf("is_object is no bool")
				return

			}
			out.isObject = tV
		case "is_bool":
			tV, ok := v.(bool)
			if !ok {
				err = fmt.Errorf("is_bool is no bool")
				return

			}
			out.isBool = tV

		}
	}

	return
}

// ObjectComparison offerst the compare feature to other packages, with the standard behavior
// noExtra=false
func ObjectComparison(left, right util.JsonObject) (res CompareResult, err error) {
	return objectComparison(left, right, false)
}

// objectComparsion checks if two objects are equal
// hereby we also check our control structures and the noExtra parameter. If noExtra is true it is not allowed to have
// elements than set
func objectComparison(left, right util.JsonObject, noExtra bool) (res CompareResult, err error) {
	res.Equal = true
	keyRegex := regexp.MustCompile(`(?P<Key>.*?):control`)

	takenInRight := make(map[string]bool, 0)
	takenInLeft := make(map[string]bool, 0)

	//Iterate over normal fields
	for ck, cv := range left {
		if takenInLeft[ck] {
			continue
		}
		var rv, lv util.GenericJson
		var rOK, lOK bool
		control := &ComparisonContext{}
		var k string

		//Check which type of key we have
		if strings.HasSuffix(ck, ":control") {
			//We have a control key
			k = keyRegex.FindStringSubmatch(ck)[1]
			if !takenInRight[k] {
				rv, rOK = right[k]
			}
			lv, lOK = left[k]

			takenInLeft[k] = true
			takenInRight[k] = true

			cvObj, ok := cv.(util.JsonObject)
			if ok {

				control, err = fillComparisonContext(cvObj)
				if err != nil {
					return res, err
				}
			}

			delete(left, ck)
		} else {
			//Normal key
			k = ck
			if !takenInRight[k] {
				rv, rOK = right[k]
			}
			lv = cv
			lOK = true

			takenInLeft[k] = true
			takenInRight[k] = true

			leftObj, ok := left[k+":control"].(util.JsonObject)
			if ok {
				iControl, err := fillComparisonContext(leftObj)
				if err != nil {
					return res, err
				}
				if iControl != nil {
					control = iControl

					delete(left, k+":control")
				}
			}
		}

		//If a value is present, it must exist
		if lOK {
			control.mustExist = true
		}

		//Check for the given key functions
		err := keyChecks(k, rv, rOK, *control)
		if err != nil {
			res.Failures = append(res.Failures, CompareFailure{Key: k, Message: err.Error()})
			res.Equal = false

			//There is no use in checking the equality of the value if the preconditions do not work
			continue
		}

		//If we have a left value, check if it is the same as right
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
				res.Failures = append(res.Failures, CompareFailure{Key: "", Message: "extra elements found in object"})
				res.Equal = false
				return
			}
		}
	}

	return res, nil
}

// ArrayComparison offerst the compare feature to other packages, with the standard behavior
// noExtra=false, orderMatter=false
func ArrayComparison(left, right util.JsonArray) (res CompareResult, err error) {
	return arrayComparison(left, right, false, false, ComparisonContext{})
}

// arrayComparison makes a simple array comparison by either running trough both arrays with the same key (orderMaters)
// or taking a value from the left array and search it in the right one
func arrayComparison(left, right util.JsonArray, noExtra, orderMaters bool, control ComparisonContext) (res CompareResult, err error) {
	res.Equal = true

	if len(left) > len(right) {
		res.Equal = false

		leftJson, err := json.MarshalIndent(left, "", " ")
		if err != nil {
			return CompareResult{}, errors.Wrap(err, "Could not marshal expected array")
		}
		rightJson, err := json.MarshalIndent(right, "", " ")
		if err != nil {
			return CompareResult{}, errors.Wrap(err, "Could not marshal actual array")
		}

		res.Failures = append(res.Failures, CompareFailure{"", fmt.Sprintf("[arrayComparison] len(expected response) > len(actual response) \nExpected response:\n%s\nActual response:\n%s\n", string(leftJson), string(rightJson))})
		return res, nil
	}

	takenInRight := make(map[int]bool, 0)

	for lk, lv := range left {
		if orderMaters {
			tmp, err := JsonEqual(lv, right[lk], control)
			if err != nil {
				return CompareResult{}, err
			}
			if tmp.Equal != true {
				for _, v := range tmp.Failures {
					key := fmt.Sprintf("[%d].%s", lk, v.Key)
					if v.Key == "" {
						key = fmt.Sprintf("[%d]", lk)
					}
					res.Failures = append(res.Failures, CompareFailure{key, fmt.Sprintf("%s", v.Message)})
				}
				res.Equal = false
			}
			takenInRight[lk] = true
		} else {
			found := false
			allTmpFailures := make([]CompareFailure, 0)
			for rk, rv := range right {
				if takenInRight[rk] {
					continue
				}

				tmp, err := JsonEqual(lv, rv, control)
				if err != nil {
					return CompareResult{}, err
				}

				if tmp.Equal == true {
					//Found an element fitting
					found = true
					takenInRight[rk] = true

					break
				}

				allTmpFailures = append(allTmpFailures, tmp.Failures...)
			}

			if found != true {
				for _, v := range allTmpFailures {
					key := fmt.Sprintf("[%d].%s", lk, v.Key)
					if v.Key == "" {
						key = fmt.Sprintf("[%d]", lk)
					}
					res.Failures = append(res.Failures, CompareFailure{key, fmt.Sprintf("%s", v.Message)})
				}
				res.Equal = false
			}
		}

	}

	if noExtra {
		for k := range right {
			if !takenInRight[k] {
				res.Failures = append(res.Failures, CompareFailure{Key: "", Message: "extra elements found in array"})
				res.Equal = false
				return
			}
		}
	}

	return
}

func ObjectEqualWithControl(left, right util.JsonObject, control ComparisonContext) (res CompareResult, err error) {
	if control.noExtra == true {
		return objectComparison(left, right, true)
	}

	return objectComparison(left, right, false)

}

func ArrayEqualWithControl(left, right util.JsonArray, control ComparisonContext) (res CompareResult, err error) {
	emptyControl := ComparisonContext{}

	if control.elementNoExtra == true {
		emptyControl.noExtra = true
	}

	if control.orderMatters == true {
		if control.noExtra == true {
			//No extra with order
			return arrayComparison(left, right, true, true, emptyControl)
		} else {
			//with extra with order
			return arrayComparison(left, right, false, true, emptyControl)
		}
	} else {
		if control.noExtra == true {
			//No extra, no order
			return arrayComparison(left, right, true, false, emptyControl)
		} else {
			//with extra, no order
			return arrayComparison(left, right, false, false, emptyControl)
		}
	}
}

func keyChecks(lk string, right util.GenericJson, rOK bool, control ComparisonContext) (err error) {
	if control.isString == true {
		if right == nil {
			return fmt.Errorf("actual response[%s] == nil but should exists", lk)
		}
		jsonType := getJsonType(right)
		if jsonType != "String" {
			return fmt.Errorf("actual response[%s] should be 'String' but is '%s'", lk, jsonType)
		}
	} else if control.isNumber == true {
		if right == nil {
			return fmt.Errorf("actual response[%s] == nil but should exists", lk)
		}
		jsonType := getJsonType(right)
		if jsonType != "Number" {
			return fmt.Errorf("actual response[%s] should be 'Number' but is '%s'", lk, jsonType)
		}
	} else if control.isBool == true {
		if right == nil {
			return fmt.Errorf("actual response[%s] == nil but should exists", lk)
		}
		jsonType := getJsonType(right)
		if jsonType != "Bool" {
			return fmt.Errorf("actual response[%s] should be 'Bool' but is '%s'", lk, jsonType)
		}
	} else if control.isArray == true {
		if right == nil {
			return fmt.Errorf("actual response[%s] == nil but should exists", lk)
		}
		jsonType := getJsonType(right)
		if jsonType != "Array" {
			return fmt.Errorf("actual response[%s] should be 'Array' but is '%s'", lk, jsonType)
		}
	} else if control.isObject == true {
		if right == nil {
			return fmt.Errorf("actual response[%s] == nil but should exists", lk)
		}
		jsonType := getJsonType(right)
		if jsonType != "Object" {
			return fmt.Errorf("actual response[%s] should be 'Object' but is '%s'", lk, jsonType)
		}
	}

	//Check if exists
	if rOK == false && control.mustExist == true {
		return fmt.Errorf("actual response[%s] was not found, but should exists", lk)
	}

	if rOK == true && control.mustNotExist == true {
		return fmt.Errorf("actual response[%s] was found, but should NOT exist", lk)
	}

	//Check for array length
	if leftLen := control.elementCount; leftLen != nil {
		jsonType := getJsonType(right)
		if jsonType != "Array" {
			return fmt.Errorf("actual response[%s] should be 'Array' but is '%s'", lk, jsonType)
		}

		rightArray := right.(util.JsonArray)
		rightLen := int64(len(rightArray))
		if rightLen != *leftLen {
			return fmt.Errorf("length of the actual response array '%d' != '%d' expected length", rightLen, *leftLen)
		}
	}

	return nil
}

func getJsonType(value util.GenericJson) string {
	switch value.(type) {
	case util.JsonObject:
		return "Object"
	case util.JsonArray:
		return "Array"
	case util.JsonString:
		return "String"
	case util.JsonNumber:
		return "Number"
	case util.JsonBool:
		return "Bool"
	default:
		return "No JSON Type"
	}
}

func getAsInt64(value util.GenericJson) (v int64, err error) {
	switch t := value.(type) {
	case int64:
		return t, nil
	case int:
		return int64(t), nil
	case float32, float64:
		return strconv.ParseInt(fmt.Sprintf("%.0f", t), 10, 64)
	case json.Number:
		return t.Int64()
	default:
		return 0, fmt.Errorf("'%v' has no valid json number type", value)
	}
}
