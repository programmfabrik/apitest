package template

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/programmfabrik/fylr-apitest/lib/api"
	"github.com/programmfabrik/fylr-apitest/lib/filesystem"

	"github.com/programmfabrik/fylr-apitest/lib/test_utils"
	"github.com/spf13/afero"
)

func TestRender_LoadFile_withParam(t *testing.T) {
	root := []byte("{{ file \"somefile.json\" \"bogus\"}}")
	target := []byte("{{ .Param1 }}")

	filesystem.Fs = afero.NewMemMapFs()
	filesystem.Fs.MkdirAll("some/path", 0755)
	afero.WriteFile(filesystem.Fs, "some/path/somefile.json", target, 0644)

	loader := NewLoader(api.NewStore())
	res, err := loader.Render(root, "some/path", nil)
	test_utils.CheckError(t, err, fmt.Sprintf("%s", err))
	test_utils.AssertStringEquals(t, string(res), "bogus")
}

func TestRenderWithDataStore_LoadFile_withParam_recursive(t *testing.T) {
	root := []byte("{{ file \"a/next.tmpl\" \"bogus\" }}")
	next := []byte("{{ file \"b/last.tmpl\" .Param1 }}")
	last := []byte("{{ .Param1 }}")

	filesystem.Fs = afero.NewMemMapFs()
	filesystem.Fs.MkdirAll("root/a/b/", 0755)
	afero.WriteFile(filesystem.Fs, "root/a/next.tmpl", next, 0644)
	afero.WriteFile(filesystem.Fs, "root/a/b/last.tmpl", last, 0644)

	loader := NewLoader(api.NewStore())
	res, err := loader.Render(root, "root", nil)
	test_utils.CheckError(t, err, fmt.Sprintf("%s", err))
	test_utils.AssertStringEquals(t, string(res), "bogus")
}

func TestRenderWithDataStore_LoadFile_TooManyParams(t *testing.T) {
	manifestdir := "some/path"
	filename := "somefile.json"
	rootTmplContent := fmt.Sprintf(`{{ file "%s" "1" "2" "3" "4" "5" }}`, filename)
	targetFileContent := ""
	filesystem.Fs = afero.NewMemMapFs()
	filesystem.Fs.MkdirAll(manifestdir, 0755)
	afero.WriteFile(filesystem.Fs, filepath.Join(manifestdir, filename), []byte(targetFileContent), 0644)

	loader := NewLoader(api.NewStore())
	testTmpl := []byte(rootTmplContent)
	_, err := loader.Render(testTmpl, manifestdir, nil)
	if err == nil {
		t.Fatal("expected error, got none")
	}
	if !strings.Contains(err.Error(), "newParams only supports up to 4 parameters") {
		t.Errorf("expected error because of too many params, got: %s", err)
	}
}

func TestBigIntRender(t *testing.T) {
	store := api.NewStore()
	loader := NewLoader(store)

	inputNumber := "132132132182323"

	resp, _ := api.NewResponse(200, nil, strings.NewReader(fmt.Sprintf(`{"bigINT":%s}`, inputNumber)))

	store.SetWithQjson(resp, map[string]string{"testINT": "body.bigINT"})

	res, err := loader.Render([]byte(`{{ datastore "testINT" }}`), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(res) != inputNumber {
		t.Error(string(res), " != ", inputNumber)
	}
}

func TestRowsToMapTemplate(t *testing.T) {
	inputJson := `[{\"column_a\": \"row1a\",\"column_b\": \"row1b\",\"column_c\": \"row1c\"},{\"column_a\": \"row2a\",\"column_b\": \"row2b\",\"column_c\": \"row2c\"}]`

	root := []byte(`{{ unmarshal "` + inputJson + `" | rows_to_map "column_a" "column_c" }}`)

	loader := NewLoader(api.NewStore())
	res, err := loader.Render(root, "some/path", nil)

	t.Log(string(res))
	test_utils.CheckError(t, err, fmt.Sprintf("%s", err))

	test_utils.AssertStringContainsSubstringsNoOrder(t, string(res), []string{
		"row1a:row1c",
		"row2a:row2c",
	})
}

func TestRender_LoadFile_QJson_Params(t *testing.T) {
	root := []byte(
		"{{ file \"somefile.json\" \"foo\" \"bar\" | qjson \"key.1\" }}")
	target := []byte("{ \"key\": [\"{{ .Param1 }}\", \"{{ .Param2 }}\"]}")

	filesystem.Fs = afero.NewMemMapFs()
	filesystem.Fs.MkdirAll("some/path", 0755)
	afero.WriteFile(filesystem.Fs, "some/path/somefile.json", target, 0644)

	loader := NewLoader(api.NewStore())
	res, err := loader.Render(root, "some/path", nil)
	test_utils.CheckError(t, err, fmt.Sprintf("%s", err))
	test_utils.AssertStringEquals(t, string(res), "\"bar\"")
}

func TestRender_LoadFile_CSV(t *testing.T) {
	testCases := []struct {
		csv         string
		expected    string
		expectedErr error
	}{
		{`id,name,friends,ages
int64,string,"string,array","int64,array"
1,simon,"simon,jonas,stefan","21,24,12"`, `[{"ages":[21,24,12],"friends":["simon","jonas","stefan"],"id":1,"name":"simon"}]`, nil},
		{`id,name,friends,ages

int64,string,"string,array","int64,array"

,,,
1,simon,"simon,jonas,stefan","21,24,12"`, `[{"ages":[21,24,12],"friends":["simon","jonas","stefan"],"id":1,"name":"simon"}]`, nil},
		{`id,name,friends,ages

int64,string,"string,array","int64,array"

,,,
,hans,,
1,simon,"simon,jonas,stefan","21,24,12"`, `[{"ages":[],"friends":[],"id":0,"name":"hans"},{"ages":[21,24,12],"friends":["simon","jonas","stefan"],"id":1,"name":"simon"}]`, nil},
		{`id,name,friends,ages

int64,string,"string,array","int64,array"

,,,
,hans,,
1,simon,"simon,jo
nas,ste
fan","21,24,12"`, ``, fmt.Errorf(`error executing body template: template: tmpl:1:3: executing "tmpl" at <file_csv "somefile.j...>: error calling file_csv: 'somefile.json' Only one row is allowed for type 'string,array'`)},

		{`id,name,friends,ages

int64,string,"string,array","int64,array"

,,,
#,hans,,
1,simon,"simon,""jo
nas"",""a,b""","21,24,12"`, `[{"ages":[21,24,12],"friends":["simon","jo\nnas","a,b"],"id":1,"name":"simon"}]`, nil},
		{`id,name,friends,ages

int64,string,"string,array"

,,,
#,hans,,
1,simon,"simon,""jo
nas"",""a,b""","21,24,12"`, `[{"friends":["simon","jo\nnas","a,b"],"id":1,"name":"simon"}]`, nil},
		{`id,name, ,ages

int64,string,"string,array"

,,,
#,hans,,
1,simon,"simon,""jo
nas"",""a,b""","21,24,12"`, `[{"id":1,"name":"simon"}]`, nil},
		{`id,name,de,friends,ages

int64,string,,"string,array"

,,,
#,hans,,
1,simon,LALALALA,"simon,""jo
nas"",""a,b""","21,24,12"`, `[{"friends":["simon","jo\nnas","a,b"],"id":1,"name":"simon"}]`, nil},
		{`id,name,de,friends,ages

int64,string,s,"string,array"

,,,
#,hans,,
1,simon,LALALALA,"simon,""jo
nas"",""a,b""","21,24,12"`, ``, fmt.Errorf(`error executing body template: template: tmpl:1:3: executing "tmpl" at <file_csv "somefile.j...>: error calling file_csv: 'somefile.json' 's' is no valid format`)},
		{`id,name,,ages
int64,string,"string,array","int64,array"`, `[]`, nil},
		{`id,name,friends,ages
int64,string,"stringer,array","int64,array"`, ``, fmt.Errorf(`error executing body template: template: tmpl:1:3: executing "tmpl" at <file_csv "somefile.j...>: error calling file_csv: 'somefile.json' 'stringer,array' is no valid format`)},
	}
	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			root := []byte(fmt.Sprintf(`{{ file_csv "somefile.json" ',' | marshal }}`))

			target := []byte(testCase.csv)

			filesystem.Fs = afero.NewMemMapFs()
			afero.WriteFile(filesystem.Fs, "somefile.json", target, 0644)

			loader := NewLoader(api.NewStore())
			res, err := loader.Render(root, "", nil)
			test_utils.AssertErrorContains(t, err, testCase.expectedErr)

			if err == nil {
				test_utils.AssertStringEquals(t, string(res), testCase.expected)
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
		{`id,name,friends,ages

int64,string,"string,array","int64,array"

,,,
,hans,,
1,simon,"simon,jonas,stefan","21,24,12"`, `{"hans":[],"simon":[21,24,12]}`, nil},
	}
	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			root := []byte(fmt.Sprintf(`{{ file_csv "somefile.json" ',' | rows_to_map "name" "ages" | marshal }}`))

			target := []byte(testCase.csv)

			filesystem.Fs = afero.NewMemMapFs()
			afero.WriteFile(filesystem.Fs, "somefile.json", target, 0644)

			loader := NewLoader(api.NewStore())
			res, err := loader.Render(root, "", nil)
			test_utils.AssertErrorContains(t, err, testCase.expectedErr)

			if err == nil {
				test_utils.AssertStringEquals(t, string(res), testCase.expected)
			}
		})
	}
}

func TestRender_LoadFile_CSVQjson(t *testing.T) {
	testCases := []struct {
		csv         string
		qjson       string
		expected    string
		expectedErr error
	}{
		{`id,name,friends,ages
int64,string,"string,array","int64,array"
1,simon,"simon,jonas,stefan","21,24,12"`, `0.name`, `"simon"`, nil},
		{`id,name,friends,ages
int64,string,"string,array","int64,array"
1,simon,"simon,jonas,stefan","21,24,12"
2,stefan,"simon,jonas,stefan","21,24,12"`, `1.friends.2`, `"stefan"`, nil},
	}
	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			root := []byte(fmt.Sprintf(`{{ file_csv "somefile.json" ',' | marshal | qjson "%s" }}`, testCase.qjson))

			target := []byte(testCase.csv)

			filesystem.Fs = afero.NewMemMapFs()
			afero.WriteFile(filesystem.Fs, "somefile.json", target, 0644)

			loader := NewLoader(api.NewStore())
			res, err := loader.Render(root, "", nil)
			test_utils.AssertErrorContains(t, err, testCase.expectedErr)

			if err == nil {
				test_utils.AssertStringEquals(t, string(res), testCase.expected)
			}
		})
	}
}

func TestRender_LoadFile_QJson(t *testing.T) {
	testCases := []struct {
		path        string
		json        string
		expected    string
		expectedErr error
	}{
		{`body.1._id`, `{"body":[{"_id": 1}, {"_id": 2}]}`, `2`, nil},
		{`body.0`, `{"body":[{"_id": 1}, {"_id": 2}]}`, `{"_id": 1}`, nil},
		{`body.invalid`, `{"body":[{"_id": 1}, {"_id": 2}]}`, ``, fmt.Errorf("'body.invalid' was not found or was empty string")}, //beware wrong access returns nothing
		{`body.array`, `{"body": {"array": [1, 2]}}`, `[1, 2]`, nil},
		{`body.array.1`, `{"body": {"array": [1, 2]}}`, `2`, nil},
		{`body.0._id`, `{"body":[{"_id":2}]}`, `2`, nil},
	}
	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			root := []byte(fmt.Sprintf(`{{ file "somefile.json" | qjson "%s" }}`, testCase.path))
			target := []byte(testCase.json)

			filesystem.Fs = afero.NewMemMapFs()
			filesystem.Fs.MkdirAll("some/path", 0755)
			afero.WriteFile(filesystem.Fs, "some/path/somefile.json", target, 0644)

			loader := NewLoader(api.NewStore())
			res, err := loader.Render(root, "some/path", nil)
			test_utils.AssertErrorContains(t, err, testCase.expectedErr)

			if err != nil {
				test_utils.AssertStringEquals(t, string(res), testCase.expected)
			}
		})
	}
}

func Test_DataStore_QJson(t *testing.T) {
	response, _ := api.NewResponse(
		200,
		map[string][]string{"x-header": {"foo", "bar"}},
		strings.NewReader(`{"flib": ["flab", "flob"]}`),
	)
	store := api.NewStore()
	jsonResponse, err := response.ToJsonString()
	if err != nil {
		t.Fatal(err)
	}
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
		{"body.flib", `["flab","flob"]`},
		{"body", `{"flib":["flab","flob"]}`},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			root := []byte(fmt.Sprintf(`{{ datastore 0 | qjson "%s" }}`, testCase.path))
			res, err := loader.Render(root, "some/path", nil)

			test_utils.CheckError(t, err, fmt.Sprintf("%s", err))
			test_utils.AssertStringEquals(t, string(res), testCase.expected)
		})
	}

}
