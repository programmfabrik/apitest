package template

import (
	"fmt"
	"strings"
	"testing"

	"github.com/programmfabrik/apitest/pkg/lib/datastore"
	"github.com/programmfabrik/apitest/pkg/lib/test_utils"
	"github.com/programmfabrik/golib"

	"github.com/programmfabrik/apitest/pkg/lib/api"
	"github.com/programmfabrik/apitest/pkg/lib/filesystem"
	go_test_utils "github.com/programmfabrik/go-test-utils"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/afero"
)

func TestRender_Custom_Delimiters(t *testing.T) {
	root := []byte(`** range $i, $v := N 3 ** ** $i ** ** end **`)
	loader := NewLoader(datastore.NewStore(false))
	loader.Delimiters.Left = "**"
	loader.Delimiters.Right = "**"
	res, err := loader.Render(root, "", nil)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	go_test_utils.AssertStringEquals(t, string(res), " 0  1  2 ")
}

func TestRender_Custom_Delimiters_Comments(t *testing.T) {
	root := []byte(`// range $i, $v := N 3 // // $i // // end //`)
	loader := NewLoader(datastore.NewStore(false))
	loader.Delimiters.Left = "//"
	loader.Delimiters.Right = "//"
	res, err := loader.Render(root, "", nil)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	go_test_utils.AssertStringEquals(t, string(res), " 0  1  2 ")
}

func TestRender_Custom_Delimiters_Comments_Stripped(t *testing.T) {
	root := []byte(`// ## range $i, $v := N 3 ## ## $i ## ## end ##`)
	loader := NewLoader(datastore.NewStore(false))
	loader.Delimiters.Left = "##"
	loader.Delimiters.Right = "##"
	res, err := loader.Render(root, "", nil)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	go_test_utils.AssertStringEquals(t, string(res), "")
}

func TestRender_LoadFile_withParam(t *testing.T) {
	root := []byte(`{{ file_render "somefile.json" "bogus"}}`)
	target := []byte(`{{ .Param1 }}`)

	filesystem.Fs = afero.NewMemMapFs()
	err := filesystem.Fs.MkdirAll("some/path", 0755)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	err = afero.WriteFile(filesystem.Fs, "some/path/somefile.json", target, 0644)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

	loader := NewLoader(datastore.NewStore(false))
	res, err := loader.Render(root, "some/path", nil)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	go_test_utils.AssertStringEquals(t, string(res), "bogus")
}

func TestRenderWithDataStore_LoadFile_withParam_recursive(t *testing.T) {
	root := []byte(`{{ file_render "a/next.tmpl" "bogus" }}`)
	next := []byte(`{{ file_render "b/last.tmpl" .Param1 }}`)
	last := []byte(`{{ .Param1 }}`)

	filesystem.Fs = afero.NewMemMapFs()
	err := filesystem.Fs.MkdirAll("root/a/b/", 0755)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	err = afero.WriteFile(filesystem.Fs, "root/a/next.tmpl", next, 0644)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	err = afero.WriteFile(filesystem.Fs, "root/a/b/last.tmpl", last, 0644)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

	loader := NewLoader(datastore.NewStore(false))
	res, err := loader.Render(root, "root", nil)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	go_test_utils.AssertStringEquals(t, string(res), "bogus")
}

func TestBigIntRender(t *testing.T) {
	store := datastore.NewStore(false)
	loader := NewLoader(store)

	inputNumber := "132132132182323"

	resp, err := api.NewResponse(golib.IntRef(200), nil, nil, strings.NewReader(fmt.Sprintf(`{"bigINT":%s}`, inputNumber)), nil, api.ResponseFormat{})
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

	respJson, err := resp.ServerResponseToJsonString(false)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	err = store.SetWithGjson(respJson, map[string]string{"testINT": "body.bigINT"})
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

	res, err := loader.Render([]byte(`{{ datastore "testINT" }}`), "", nil)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	if string(res) != inputNumber {
		t.Error(string(res), " != ", inputNumber)
	}
}

func TestRowsToMapTemplate(t *testing.T) {
	inputJson := `[{\"column_a\": \"row1a\",\"column_b\": \"row1b\",\"column_c\": \"row1c\"},{\"column_a\": \"row2a\",\"column_b\": \"row2b\",\"column_c\": \"row2c\"}]`

	root := []byte(`{{ unmarshal "` + inputJson + `" | rows_to_map "column_a" "column_c" }}`)

	loader := NewLoader(datastore.NewStore(false))
	res, err := loader.Render(root, "some/path", nil)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

	t.Log(string(res))
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

	go_test_utils.AssertStringContainsSubstringsNoOrder(t, string(res), []string{
		"row1a:row1c",
		"row2a:row2c",
	})
}

func TestRender_LoadFile_GJson_Params(t *testing.T) {
	root := []byte(`{{ file_render "somefile.json" "foo" "bar" | gjson "key.1" }}`)
	target := []byte(`{ "key": ["{{ .Param1 }}", "{{ .Param2 }}"]}`)

	filesystem.Fs = afero.NewMemMapFs()
	err := filesystem.Fs.MkdirAll("some/path", 0755)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	err = afero.WriteFile(filesystem.Fs, "some/path/somefile.json", target, 0644)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

	loader := NewLoader(datastore.NewStore(false))
	res, err := loader.Render(root, "some/path", nil)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	go_test_utils.AssertStringEquals(t, string(res), `"bar"`)
}

func TestRender_LoadFile_CSV(t *testing.T) {
	testCases := []struct {
		csv       string
		expected  string
		expectErr bool
	}{
		{
			`id,name,friends,ages
			int64,string,"string,array","int64,array"
			1,simon,"simon,jonas,stefan","21,24,12"`,
			`[{"ages":[21,24,12],"friends":["simon","jonas","stefan"],"id":1,"name":"simon"}]`,
			false,
		},
		{
			`id,name,friends,ages

			int64,string,"string,array","int64,array"

			,,,
			1,simon,"simon,jonas,stefan","21,24,12"`,
			`[{"ages":[21,24,12],"friends":["simon","jonas","stefan"],"id":1,"name":"simon"}]`,
			false,
		},
		{
			`id,name,friends,ages

			int64,string,"string,array","int64,array"

			,,,
			,hans,,
			1,simon,"simon,jonas,stefan","21,24,12"`,
			`[{"ages":[],"friends":[],"id":0,"name":"hans"},{"ages":[21,24,12],"friends":["simon","jonas","stefan"],"id":1,"name":"simon"}]`,
			false,
		},
		{
			`id,name,friends,ages

			int64,string,"string,array","int64,array"

			,,,
			,hans,,
			1,simon,"simon,jo
			nas,ste
			fan","21,24,12"`,
			``,
			true,
		},
		{
			`id,name,friends,ages

			int64,string,"string,array","int64,array"

			,,,
			#,hans,,
			1,simon,"simon,""jo
			nas"",""a,b""","21,24,12"`,
			`[{"ages":[21,24,12],"friends":["simon","jo\nnas","a,b"],"id":1,"name":"simon"}]`,
			false,
		},
		{
			`id,name,friends,ages

			int64,string,"string,array"

			,,,
			#,hans,,
			1,simon,"simon,""jo
			nas"",""a,b""","21,24,12"`,
			`[{"friends":["simon","jo\nnas","a,b"],"id":1,"name":"simon"}]`,
			false,
		},
		{
			`id,name, ,ages

			int64,string,"string,array"

			,,,
			#,hans,,
			1,simon,"simon,""jo
			nas"",""a,b""","21,24,12"`,
			`[{"id":1,"name":"simon"}]`,
			false,
		},
		{
			`id,name,de,friends,ages

			int64,string,,"string,array"

			,,,
			#,hans,,
			1,simon,LALALALA,"simon,""jo
			nas"",""a,b""","21,24,12"`,
			`[{"friends":["simon","jo\nnas","a,b"],"id":1,"name":"simon"}]`,
			false,
		},
		{
			`id,name,de,friends,ages

			int64,string,s,"string,array"

			,,,
			#,hans,,
			1,simon,LALALALA,"simon,""jo
			nas"",""a,b""","21,24,12"`,
			``,
			true,
		},
		{
			`id,name,,ages
			int64,string,"string,array","int64,array"`,
			`[]`,
			false,
		},
		{
			`id,name,friends,ages
			int64,string,"stringer,array","int64,array"`,
			``,
			true,
		},
	}
	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			root := []byte(`{{ file_csv "somefile.json" ',' | marshal }}`)

			target := []byte(strings.ReplaceAll(testCase.csv, "\t", ""))

			filesystem.Fs = afero.NewMemMapFs()
			err := afero.WriteFile(filesystem.Fs, "somefile.json", target, 0644)
			go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

			loader := NewLoader(datastore.NewStore(false))
			res, err := loader.Render(root, "", nil)

			if err == nil {
				if testCase.expectErr {
					t.Errorf("Error expected")
				}
				if string(res) != testCase.expected {
					dmp := diffmatchpatch.New()

					diffs := dmp.DiffMain(string(res), testCase.expected, false)

					t.Errorf("Result differs: %s", dmp.DiffPrettyText(diffs))
				}
			} else {
				if !testCase.expectErr {
					go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
				}
			}
		})
	}
}

func TestRender_LoadFile_CSV_And_Row_To_Map(t *testing.T) {
	testCases := []struct {
		csv         string
		expected    string
		expectedErr error
	}{
		{
			`id,name,friends,ages

			int64,string,"string,array","int64,array"

			,,,
			,hans,,
			1,simon,"simon,jonas,stefan","21,24,12"`,
			`{"hans":[],"simon":[21,24,12]}`,
			nil,
		},
	}
	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			root := []byte(`{{ file_csv "somefile.json" ',' | rows_to_map "name" "ages" | marshal }}`)

			target := []byte(strings.ReplaceAll(testCase.csv, "\t", ""))

			filesystem.Fs = afero.NewMemMapFs()
			err := afero.WriteFile(filesystem.Fs, "somefile.json", target, 0644)
			go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

			loader := NewLoader(datastore.NewStore(false))
			res, err := loader.Render(root, "", nil)
			if testCase.expectedErr != nil {
				go_test_utils.AssertErrorContains(t, err, testCase.expectedErr.Error())
			} else {
				go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
				go_test_utils.AssertStringEquals(t, string(res), testCase.expected)
			}
		})
	}
}

func TestRender_LoadFile_CSV_Gjson(t *testing.T) {
	testCases := []struct {
		csv         string
		gjson       string
		expected    string
		expectedErr error
	}{
		{
			`id,name,friends,ages
			int64,string,"string,array","int64,array"
			1,simon,"simon,jonas,stefan","21,24,12"`,
			`0.name`,
			`"simon"`,
			nil,
		},
		{
			`id,name,friends,ages
			int64,string,"string,array","int64,array"
			1,simon,"simon,jonas,stefan","21,24,12"
			2,stefan,"simon,jonas,stefan","21,24,12"`,
			`1.friends.2`,
			`"stefan"`,
			fmt.Errorf(""),
		},
	}
	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			root := []byte(fmt.Sprintf(`{{ file_csv "somefile.json" ',' | marshal | gjson %q }}`, testCase.gjson))

			target := []byte(strings.ReplaceAll(testCase.csv, "\t", ""))

			filesystem.Fs = afero.NewMemMapFs()
			err := afero.WriteFile(filesystem.Fs, "somefile.json", target, 0644)
			go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

			loader := NewLoader(datastore.NewStore(false))
			res, err := loader.Render(root, "", nil)
			if testCase.expectedErr != nil {
				go_test_utils.AssertErrorContains(t, err, testCase.expectedErr.Error())
			} else {
				go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
				go_test_utils.AssertStringEquals(t, string(res), testCase.expected)
			}
		})
	}
}

func TestRender_LoadFile_GJson(t *testing.T) {
	testCases := []struct {
		path        string
		json        string
		expected    string
		expectedErr error
	}{
		{
			`body.1._id`,
			`{
				"body": [
					{
						"_id": 1
					},
					{
						"_id": 2
					}
				]
			}`,
			`2`,
			nil,
		},
		{
			`body.0`,
			`{
				"body": [
					{
						"_id": 1
					},
					{
						"_id": 2
					}
				]
			}`,
			`{
				"_id": 1
			}`,
			nil,
		},
		{
			`body.invalid`,
			`{
				"body": [
					{
						"_id": 1
					},
					{
						"_id": 2
					}
				]
			}`,
			``,
			fmt.Errorf("'body.invalid' was not found or was empty string"), // beware wrong access returns nothing
		},
		{
			`body.array`,
			`{
				"body": {
					"array": [
						1,
						2
					]
				}
				}`,
			`[
				1,
				2
			]`,
			nil,
		},
		{
			`body.array.1`,
			`{
				"body": {
					"array": [
						1,
						2
					]
				}
			}`,
			`2`,
			nil,
		},
		{
			`body.0._id`,
			`{
				"body": [
					{
						"_id": 2
					}
				]
			}`,
			`2`,
			nil,
		},
	}
	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			root := []byte(fmt.Sprintf(`{{ file "somefile.json" | gjson %q }}`, testCase.path))
			target := []byte(testCase.json)

			filesystem.Fs = afero.NewMemMapFs()
			err := filesystem.Fs.MkdirAll("some/path", 0755)
			go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
			err = afero.WriteFile(filesystem.Fs, "some/path/somefile.json", target, 0644)
			go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))

			loader := NewLoader(datastore.NewStore(false))
			res, err := loader.Render(root, "some/path", nil)
			if testCase.expectedErr != nil {
				go_test_utils.AssertErrorContains(t, err, testCase.expectedErr.Error())
			} else {
				go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
				go_test_utils.AssertStringEquals(
					t,
					strings.ReplaceAll(string(res), "\t", ""),
					strings.ReplaceAll(testCase.expected, "\t", ""),
				)
			}
		})
	}
}

func Test_DataStore_GJson(t *testing.T) {
	response, _ := api.NewResponse(
		golib.IntRef(200),
		map[string]any{"x-header": []string{"foo", "bar"}},
		nil,
		strings.NewReader(`{
			"flib": [
				"flab",
				"flob"
			]
		}`),
		nil,
		api.ResponseFormat{},
	)
	store := datastore.NewStore(false)
	jsonResponse, err := response.ServerResponseToJsonString(false)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	store.AppendResponse(string(jsonResponse))

	loader := NewLoader(store)

	testCases := []struct {
		path     string
		expected string
	}{
		{"header.x-header.0", `"foo"`},
		{"header.x-header.1", `"bar"`},
		{"statuscode", `200`},
		{"body.flib.0", `"flab"`},
		{"body.flib.1", `"flob"`},
		{"body.flib", `[
			"flab",
			"flob"
		]`},
		{"body", `{
			"flib": [
				"flab",
				"flob"
			]
		}`},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			root := []byte(fmt.Sprintf(`{{ datastore 0 | gjson %q }}`, testCase.path))
			res, err := loader.Render(root, "some/path", nil)
			go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
			test_utils.AssertJsonStringEquals(t, string(res), testCase.expected)
		})
	}
}

func TestReplaceHost(t *testing.T) {
	var (
		h   string
		err error
	)

	testCases := []struct {
		srcUrl     string
		serverHost string
		expected   string
		expErr     bool
	}{
		{
			"localhost:9925",
			"192.168.122.56:8978",
			"xxx",
			true,
		},
		{
			"localhost:9925",
			"192.168.122.56",
			"192.168.122.56:9925",
			false,
		},
		{
			"http://localhost:8978/images",
			"martins.mac",
			"http://martins.mac:8978/images",
			false,
		},
		{
			"http://localhost:8978",
			"192.168.122.56",
			"http://192.168.122.56:8978",
			false,
		},
	}
	for _, testCase := range testCases {
		h, err = replaceHost(testCase.srcUrl, testCase.serverHost)
		if !testCase.expErr {
			go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
			go_test_utils.AssertStringEquals(t, testCase.expected, h)
		} else {
			go_test_utils.ExpectError(t, err, "expected error for invalid host")
		}
	}
}

func errorStringIfNotNil(err error) (errS string) {
	if err == nil {
		return ""
	}
	return err.Error()
}
