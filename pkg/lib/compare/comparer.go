package compare

import (
	"fmt"

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

func JsonEqual(left, right util.GenericJson, control ComparisonContext) (res CompareResult, err error) {

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
						fmt.Sprintf("Expected '%s' != '%s' Got", typedLeft, rightAsString),
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
						fmt.Sprintf("Expected '%v' != '%v' Got", typedLeft, rightAsNumber),
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
						fmt.Sprintf("Expected '%t' != '%t' Got", typedLeft, rightAsBool),
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
					fmt.Sprintf("the type of the expected response is invalid. Expected '%T' != '%T' Got", left, right),
				},
			},
		}
		return res, nil
	}
}
