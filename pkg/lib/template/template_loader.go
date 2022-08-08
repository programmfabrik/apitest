package template

import (
	"bytes"
	"crypto/md5"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/pkg/errors"
	"github.com/programmfabrik/apitest/pkg/lib/datastore"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/semver"
	"golang.org/x/oauth2"

	"github.com/programmfabrik/apitest/pkg/lib/util"

	"io/ioutil"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/tidwall/gjson"
)

// delimiters as go template parsing options
type delimiters struct {
	Left  string
	Right string
}

var delimsRE = regexp.MustCompile(`(?m)^[\t ]*(?://|/\*)[\t ]*template-delims:[\t ]*([^\t ]+)[\t ]+([^\t\n ]+).*$`)

type Loader struct {
	datastore      *datastore.Datastore
	HTTPServerHost string
	ServerURL      *url.URL
	OAuthClient    util.OAuthClientsConfig
	Delimiters     delimiters
}

func NewLoader(datastore *datastore.Datastore) Loader {
	return Loader{datastore: datastore}
}

func (loader *Loader) Render(
	tmplBytes []byte,
	rootDir string,
	ctx interface{}) (res []byte, err error) {

	// First check for custom delimiters
	matches := delimsRE.FindStringSubmatch(string(tmplBytes))
	if len(matches) == 3 {
		loader.Delimiters.Left, loader.Delimiters.Right = matches[1], matches[2]
	}

	// Second check for placeholders removal
	removeCheckRE := regexp.MustCompile(`(?m)^[\t ]*(?://|/\*)[\t ]*template-remove-tokens:[\t ]*(.+)$`)
	matches = removeCheckRE.FindStringSubmatch(string(tmplBytes))
	replacements := []string{}
	if len(matches) > 1 {
		splitRE := regexp.MustCompile(`[\t ]`)
		placeholders := splitRE.Split(matches[1], -1)
		for _, s := range placeholders {
			replacements = append(replacements, s, "")
		}
	}
	newTmplStr := strings.NewReplacer(replacements...).Replace(string(tmplBytes))
	tmplBytes = []byte(newTmplStr)

	// Remove comments from template if comments are not the delimiters
	if loader.Delimiters.Left != "//" {
		var re = regexp.MustCompile(`(?m)^[\t ]*//.*$`)
		tmplBytes = []byte(re.ReplaceAllString(string(tmplBytes), ``))
	}

	funcMap := template.FuncMap{
		"qjson": func(path string, json string) (result string, err error) {
			if json == "" {
				err = fmt.Errorf("The given json was empty")
				return
			}

			result = gjson.Get(json, path).Raw
			if len(result) == 0 {
				err = fmt.Errorf("'%s' was not found or was empty string. Qjson Input: %s", path, json)
			}
			return
		},
		"split": func(s, sep string) []string {
			return strings.Split(s, sep)
		},
		"md5sum": func(path string) (string, error) {
			_, file, err := util.OpenFileOrUrl(path, rootDir)
			if err != nil {
				return "", err
			}

			fileBytes, err := ioutil.ReadAll(file)
			if err != nil {
				return "", err
			}

			hasher := md5.New()
			hasher.Write([]byte(fileBytes))
			return hex.EncodeToString(hasher.Sum(nil)), nil
		},
		// "parse_csv": func(path string, delimiter rune) ([]map[string]interface{}, error) {
		// 	_, file, err := util.OpenFileOrUrl(path, rootDir)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	fileBytes, err := ioutil.ReadAll(file)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	data, err := csv.GenericCSVToMap(fileBytes, delimiter)
		// 	if err != nil {
		// 		return data, fmt.Errorf("'%s' %s", path, err)
		// 	}
		// 	return data, err
		// },
		"file":        loadFile(rootDir, loader),
		"file_render": loadFileAndRender(rootDir, loader),
		"file_csv":    loadFileCSV(rootDir, loader),
		"file_sqlite": func(path, statement string) ([]map[string]interface{}, error) {
			sqliteFile := filepath.Join(rootDir, path)
			database, err := sql.Open("sqlite3", sqliteFile)
			if err != nil {
				return nil, err
			}
			defer database.Close()

			rows, err := database.Query(statement)
			if err != nil {
				return nil, err
			}
			defer rows.Close()

			columns, err := rows.ColumnTypes()
			if err != nil {
				return nil, err
			}
			row := make([]interface{}, len(columns))

			data := []map[string]interface{}{}

			for rows.Next() {
				dataEntry := map[string]interface{}{}
				for idx, col := range columns {
					dataEntry[col.Name()] = new(interface{})
					row[idx] = dataEntry[col.Name()]
				}

				err = rows.Scan(row...)
				if err != nil {
					return nil, err
				}

				for idx, d := range row {
					dataEntry[columns[idx].Name()] = reflect.ValueOf(d).Elem().Interface()
				}

				data = append(data, dataEntry)
			}

			return data, nil
		},
		"file_xml2json": func(path string) (string, error) {
			fileBytes, err := fileReadInternal(path, rootDir)
			if err != nil {
				return "", err
			}

			bytes, err := util.Xml2Json(fileBytes, "xml2")
			if err != nil {
				return "", errors.Wrap(err, "Could not marshal xml to json")
			}

			return string(bytes), nil
		},
		"file_path": func(path string) string {
			return util.LocalPath(path, rootDir)
		},
		"datastore": func(index interface{}) (interface{}, error) {
			var key string

			switch idxType := index.(type) {
			case int:
				key = fmt.Sprintf("%d", idxType)
			case int64:
				key = fmt.Sprintf("%d", idxType)
			case string:
				key = idxType
				// all good
			default:
				return "", fmt.Errorf("datastore needs string, int, or int64 as parameter")
			}

			return loader.datastore.Get(key)
		},
		"unmarshal": func(s string) (interface{}, error) {
			var gj interface{}
			err := util.Unmarshal([]byte(s), &gj)
			if err != nil {
				return nil, err
			}
			return gj, nil
		},
		"N": N,
		"marshal": func(data interface{}) (string, error) {
			bytes, err := json.Marshal(data)
			if err != nil {
				return "", err
			}
			return string(bytes), nil
		},
		// return json escape string
		"str_escape": func(s string) (string, error) {
			s = strings.Replace(s, "\\", "\\\\", -1)
			return strings.Replace(s, "\"", "\\\"", -1), nil
		},
		// return json escape string
		"url_path_escape": func(s string) (string, error) {
			return url.PathEscape(s), nil
		},
		// add a + b
		"add": add,
		// subtract a - b
		"subtract": subtract,
		// multiply a * b
		"multiply": multiply,
		// divide a / b
		"divide": divide,
		// create a slice
		"slice": func(args ...interface{}) []interface{} {
			return args
		},
		"rows_to_map": func(keyColumn, valueColumn string, rowsInput interface{}) (map[string]interface{}, error) {
			return rowsToMap(keyColumn, valueColumn, getRowsFromInput(rowsInput))
		},
		"pivot_rows": pivotRows,
		"group_map_rows": func(groupColumn string, rowsInput interface{}) (map[string][]map[string]interface{}, error) {
			grouped_rows := make(map[string][]map[string]interface{}, 1000)
			rows := getRowsFromInput(rowsInput)
			for _, row := range rows {
				group_key, ok := row[groupColumn]
				if !ok {
					return nil, fmt.Errorf("Group column \"%s\" does not exist in row.", groupColumn)
				}
				switch idx := group_key.(type) {
				case string:
					_, ok := grouped_rows[idx]
					if !ok {
						grouped_rows[idx] = make([]map[string]interface{}, 0)
					}
					grouped_rows[idx] = append(grouped_rows[idx], row)
				default:
					return nil, fmt.Errorf("Group column \"%s\" needs to be int64 but is %t.", groupColumn, idx)
				}
			}
			return grouped_rows, nil
		},
		"group_rows": func(groupColumn string, rowsInput interface{}) ([][]map[string]interface{}, error) {
			grouped_rows := make([][]map[string]interface{}, 1000)
			rows := getRowsFromInput(rowsInput)

			for _, row := range rows {
				group_key, ok := row[groupColumn]
				if !ok {
					return nil, fmt.Errorf("Group column \"%s\" does not exist in row.", groupColumn)
				}
				switch idx := group_key.(type) {
				case int64:
					if idx <= 0 {
						return nil, fmt.Errorf("Group column \"%s\" needs to be >= 0 and < 1000 but is %d.", groupColumn, idx)
					}
					rows2 := grouped_rows[idx]
					if rows2 == nil {
						grouped_rows[idx] = make([]map[string]interface{}, 0)
					}
					grouped_rows[idx] = append(grouped_rows[idx], row)
				default:
					return nil, fmt.Errorf("Group column \"%s\" needs to be int64 but is %t.", groupColumn, idx)
				}
			}
			// remove empty rows
			g_rows := make([][]map[string]interface{}, 0)
			for _, row := range grouped_rows {
				if row == nil {
					continue
				}
				g_rows = append(g_rows, row)
			}
			return g_rows, nil
		},
		"match": func(regex, text string) (bool, error) {
			return regexp.Match(regex, []byte(text))
		},
		"replace_host": func(srcURL string) (string, error) {
			// If no override provided, return original one
			if loader.HTTPServerHost == "" {
				return srcURL, nil
			}
			// Parse source URL or fail
			parsedURL, err := url.Parse(srcURL)
			if err != nil {
				return "", err
			}
			parsedURL.Host = loader.HTTPServerHost
			return parsedURL.String(), nil
		},
		"server_url": func() *url.URL {
			u := new(url.URL)
			*u = *loader.ServerURL
			return u
		},
		"server_url_no_user": func() *url.URL {
			u := new(url.URL)
			*u = *loader.ServerURL
			u.User = nil
			return u
		},
		"is_zero": func(v interface{}) bool {
			if v == nil {
				return true
			}
			return reflect.ValueOf(v).IsZero()
		},
		"oauth2_password_token": func(client string, login string, password string) (tok *oauth2.Token, err error) {
			println("client", client, login, password)
			oAuthClient, ok := loader.OAuthClient[client]
			if !ok {
				return nil, errors.Errorf("OAuth client %q not configured", client)
			}
			oAuthClient.Client = client
			return oAuthClient.GetPasswordCredentialsAuthToken(login, password)

		},
		"oauth2_client_token": func(client string) (tok *oauth2.Token, err error) {
			oAuthClient, ok := loader.OAuthClient[client]
			if !ok {
				return nil, errors.Errorf("OAuth client %q not configured", client)
			}
			oAuthClient.Client = client
			return oAuthClient.GetClientCredentialsAuthToken()
		},
		"oauth2_code_token": func(client string, params ...string) (tok *oauth2.Token, err error) {
			oAuthClient, ok := loader.OAuthClient[client]
			if !ok {
				return nil, errors.Errorf("OAuth client %q not configured", client)
			}
			oAuthClient.Client = client
			return oAuthClient.GetCodeAuthToken(params...)
		},
		"oauth2_implicit_token": func(client string, params ...string) (tok *oauth2.Token, err error) {
			oAuthClient, ok := loader.OAuthClient[client]
			if !ok {
				return nil, errors.Errorf("OAuth client %q not configured", client)
			}
			oAuthClient.Client = client
			return oAuthClient.GetAuthToken(params...)
		},
		"oauth2_client": func(client string) (c *util.OAuthClientConfig, err error) {
			oAuthClient, ok := loader.OAuthClient[client]
			if !ok {
				return nil, errors.Errorf("OAuth client %s not configured", client)
			}
			oAuthClient.Client = client
			return &oAuthClient, nil
		},
		"oauth2_basic_auth": func(client string) (string, error) {
			oAuthClient, ok := loader.OAuthClient[client]
			if !ok {
				return "", errors.Errorf("OAuth client %s not configured", client)
			}
			oAuthClient.Client = client
			return "Basic " + base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", oAuthClient.Client, oAuthClient.Secret))), nil
		},
		"query_escape": func(in string) string {
			return url.QueryEscape(in)
		},
		"query_unescape": func(in string) string {
			out, err := url.QueryUnescape(in)
			if err != nil {
				return err.Error()
			}
			return out
		},
		"base64_encode": func(in string) string {
			return base64.StdEncoding.EncodeToString([]byte(in))
		},
		"base64_decode": func(in string) string {
			b, err := base64.StdEncoding.DecodeString(in)
			if err != nil {
				return err.Error()
			}
			return string(b)
		},
		"semver_compare": func(v, w string) (int, error) {
			if v == "" {
				// empty version
				v = "v0.0.0"
			}
			if w == "" {
				// empty version
				w = "v0.0.0"
			}
			if !semver.IsValid(v) {
				return 0, errors.Errorf("version string %s is invalid", v)
			}
			if !semver.IsValid(w) {
				return 0, errors.Errorf("version string %s is invalid", w)
			}
			return semver.Compare(v, w), nil
		},
		"log": func(msg string, args ...any) string {
			logrus.Debugf(msg, args...)
			return ""
		},
	}
	tmpl, err := template.
		New("tmpl").
		Delims(loader.Delimiters.Left, loader.Delimiters.Right).
		Funcs(sprig.TxtFuncMap()).
		Funcs(funcMap).
		Parse(string(tmplBytes))
	if err != nil {
		return nil, fmt.Errorf("error loading template: %s", err)
	}

	var b []byte
	buf := bytes.NewBuffer(b)
	if err = tmpl.Execute(buf, ctx); err != nil {
		return nil, fmt.Errorf("error executing template: %s", err)
	}
	return buf.Bytes(), nil
}

func getRowsFromInput(rowsInput interface{}) []map[string]interface{} {
	rows := make([]map[string]interface{}, 0)
	switch t := rowsInput.(type) {
	case []map[string]interface{}:
		rows = t
	case []interface{}:
		for _, v := range t {
			rows = append(rows, v.(map[string]interface{}))
		}
	}
	return rows
}
