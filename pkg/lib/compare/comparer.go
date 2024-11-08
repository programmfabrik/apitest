package compare

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/programmfabrik/apitest/pkg/lib/util"
)

type CompareResult struct {
	Equal    bool
	Failures []CompareFailure
}

type CompareFailure struct {
	Key     string
	Message string
}

func (f CompareFailure) String() string {
	return fmt.Sprintf("[%s] %s", f.Key, f.Message)
}

func (f CompareFailure) Error() string {
	return f.String()
}

// jsonNumberEq is comparing ints, floats or strings of the number. It fails to
// compare different formats, 1e10 != 10000000000, although it is the same mathematical value.
func jsonNumberEq(numberExp, numberGot json.Number) (eq bool) {

	expInt, expIntErr := numberExp.Int64()
	gotInt, gotIntErr := numberGot.Int64()
	expFloat, expFloatErr := numberExp.Float64()
	gotFloat, gotFloatErr := numberGot.Float64()

	var cmp string
	_ = cmp

	if expIntErr == nil && gotIntErr == nil {
		cmp = "int"
	} else if expFloatErr == nil && gotFloatErr == nil {
		cmp = "float"
	} else {
		cmp = "string"
	}

	// if any of the interpretations is out of range, we compare by string
	for _, e := range []error{
		expIntErr, gotIntErr, expFloatErr, gotFloatErr,
	} {
		if e == nil {
			continue
		}
		if strings.Contains(e.Error(), "range") {
			cmp = "string"
			break
		}
	}

	switch cmp {
	case "int":
		eq = expInt == gotInt
	case "float":
		eq = expFloat == gotFloat
	case "string":
		eq = numberExp == numberGot
	}

	// golib.Pln("exp %q == got %q : %t %s", numberExp, numberGot, eq, cmp)
	return eq
}

func JsonEqual(left, right any, control ComparisonContext) (res CompareResult, err error) {
	//left may be nil, because we dont specify the content of the field
	if left == nil && right == nil {
		res := CompareResult{
			Equal: true,
		}
		return res, nil
	}
	if right == nil && left != nil {
		res := CompareResult{
			false,
			[]CompareFailure{
				{
					"$",
					"response == nil && expected response != nil",
				},
			},
		}
		return res, nil
	}

	switch typedLeft := left.(type) {
	case json.Number:
		typedRight, ok := right.(json.Number)
		if !ok {
			res := CompareResult{
				false,
				[]CompareFailure{
					{
						"$",
						fmt.Sprintf("expected json.Number, but got %T", right),
					},
				},
			}
			return res, nil
		}

		if jsonNumberEq(typedLeft, typedRight) {
			res = CompareResult{
				Equal: true,
			}
		} else {
			res = CompareResult{
				Equal: false,
				Failures: []CompareFailure{
					{
						"",
						fmt.Sprintf("Got '%s', expected '%s'", typedRight, typedLeft),
					},
				},
			}
		}
		return res, nil

	case util.JsonObject:
		rightAsObject, ok := right.(util.JsonObject)
		if !ok {
			res := CompareResult{
				false,
				[]CompareFailure{
					{
						"$",
						"the actual response is no JsonObject",
					},
				},
			}
			return res, nil
		}

		return ObjectEqualWithControl(typedLeft, rightAsObject, control)
	case util.JsonArray:
		/*if len(typedLeft) == 0 {
			res := CompareResult{
				Equal: true,
			}
			return res, nil
		}*/

		rightAsArray, ok := right.(util.JsonArray)
		if !ok {
			res := CompareResult{
				false,
				[]CompareFailure{
					{
						"$",
						"the actual response is no JsonArray",
					},
				},
			}
			return res, nil
		}
		return ArrayEqualWithControl(typedLeft, rightAsArray, control)

	case util.JsonString:
		rightAsString, ok := right.(util.JsonString)
		if !ok {
			res := CompareResult{
				false,
				[]CompareFailure{
					{
						"$",
						"the actual response is no JsonString",
					},
				},
			}
			return res, nil
		}
		if typedLeft == rightAsString {
			res = CompareResult{
				Equal: true,
			}
		} else {
			res = CompareResult{
				Equal: false,
				Failures: []CompareFailure{
					{
						"",
						fmt.Sprintf("Got '%s', expected '%s'", rightAsString, typedLeft),
					},
				},
			}
		}
		return res, nil
	case util.JsonNumber:
		rightAsNumber, ok := right.(util.JsonNumber)
		if !ok {
			res := CompareResult{
				false,
				[]CompareFailure{
					{
						"$",
						"the actual response is no JsonNumber",
					},
				},
			}
			return res, nil
		}
		if typedLeft == rightAsNumber {
			res = CompareResult{
				Equal: true,
			}
		} else {
			res = CompareResult{
				Equal: false,
				Failures: []CompareFailure{
					{
						"",
						fmt.Sprintf("Got '%v', expected '%v'", rightAsNumber, typedLeft),
					},
				},
			}
		}
		return res, nil

	case util.JsonBool:
		rightAsBool, ok := right.(util.JsonBool)
		if !ok {
			res := CompareResult{
				false,
				[]CompareFailure{
					{
						"$",
						"the actual response is no JsonBool",
					},
				},
			}
			return res, nil
		}

		if typedLeft == rightAsBool {
			res = CompareResult{
				Equal: true,
			}
		} else {
			res = CompareResult{
				Equal: false,
				Failures: []CompareFailure{
					{
						"",
						fmt.Sprintf("Got '%t', expected '%t'", rightAsBool, typedLeft),
					},
				},
			}
		}
		return res, nil

	default:
		res := CompareResult{
			false,
			[]CompareFailure{
				{
					"",
					fmt.Sprintf("the type of the expected response is invalid. Got '%T', expected '%T'", right, left),
				},
			},
		}
		return res, nil
	}
}
