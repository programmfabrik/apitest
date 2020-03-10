package compare

import (
	"fmt"

	"github.com/programmfabrik/apitest/pkg/lib/util"
)

type Result struct {
	Equal    bool
	Failures []Failure
}

type Failure struct {
	Key     string
	Message string
}

func (f Failure) String() string {
	return fmt.Sprintf("[%s] %s", f.Key, f.Message)
}

func (f Failure) Error() string {
	return f.String()
}

func JSONEqual(left, right interface{}, control ComparisonContext) (res Result, err error) {

	// left may be nil, because we dont specify the content of the field
	if left == nil && right == nil {
		res := Result{
			Equal: true,
		}
		return res, nil
	}
	if right == nil && left != nil {
		res := Result{
			false,
			[]Failure{
				{
					"$",
					"response == nil && expected response != nil",
				},
			},
		}
		return res, nil
	}

	switch typedLeft := left.(type) {
	case util.JSONObject:
		rightAsObject, ok := right.(util.JSONObject)
		if !ok {
			res := Result{
				false,
				[]Failure{
					{
						"$",
						"the actual response is no JSONObject",
					},
				},
			}
			return res, nil
		}

		return ObjectEqualWithControl(typedLeft, rightAsObject, control)
	case util.JSONArray:
		/*if len(typedLeft) == 0 {
			res := Result{
				Equal: true,
			}
			return res, nil
		}*/

		rightAsArray, ok := right.(util.JSONArray)
		if !ok {
			res := Result{
				false,
				[]Failure{
					{
						"$",
						"the actual response is no JSONArray",
					},
				},
			}
			return res, nil
		}
		return ArrayEqualWithControl(typedLeft, rightAsArray, control)

	case util.JSONString:
		rightAsString, ok := right.(util.JSONString)
		if !ok {
			res := Result{
				false,
				[]Failure{
					{
						"$",
						"the actual response is no JSONString",
					},
				},
			}
			return res, nil
		}
		if typedLeft == rightAsString {
			res = Result{
				Equal: true,
			}
		} else {
			res = Result{
				Equal: false,
				Failures: []Failure{
					{
						"",
						fmt.Sprintf("Got '%s', expected '%s'", rightAsString, typedLeft),
					},
				},
			}
		}
		return res, nil
	case util.JSONNumber:
		rightAsNumber, ok := right.(util.JSONNumber)
		if !ok {
			res := Result{
				false,
				[]Failure{
					{
						"$",
						"the actual response is no JSONNumber",
					},
				},
			}
			return res, nil
		}
		if typedLeft == rightAsNumber {
			res = Result{
				Equal: true,
			}
		} else {
			res = Result{
				Equal: false,
				Failures: []Failure{
					{
						"",
						fmt.Sprintf("Got '%v', expected '%v'", rightAsNumber, typedLeft),
					},
				},
			}
		}
		return res, nil

	case util.JSONBool:
		rightAsBool, ok := right.(util.JSONBool)
		if !ok {
			res := Result{
				false,
				[]Failure{
					{
						"$",
						"the actual response is no JSONBool",
					},
				},
			}
			return res, nil
		}

		if typedLeft == rightAsBool {
			res = Result{
				Equal: true,
			}
		} else {
			res = Result{
				Equal: false,
				Failures: []Failure{
					{
						"",
						fmt.Sprintf("Got '%t', expected '%t'", rightAsBool, typedLeft),
					},
				},
			}
		}
		return res, nil

	default:
		res := Result{
			false,
			[]Failure{
				{
					"",
					fmt.Sprintf("the type of the expected response is invalid. Got '%T', expected '%T'", right, left),
				},
			},
		}
		return res, nil
	}
}
