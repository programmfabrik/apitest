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
	"github.com/programmfabrik/apitest/pkg/lib/datastore"
	"github.com/programmfabrik/golib"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/semver"
	"golang.org/x/oauth2"

	"github.com/programmfabrik/apitest/pkg/lib/util"

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

// Loader loads and executes a manifest template.
//
// A manifest template is a customized version of Go's text/template, plus
// custom template functions (which are initialized with the Loader's
// properties, where applicable).
type Loader struct {
	datastore      *datastore.Datastore
	HTTPServerHost string
	ServerURL      *url.URL
	OAuthClient    util.OAuthClientsConfig
	Delimiters     delimiters

	// ParallelRunIdx is the index of the Parallel Run that this Loader is used in
	ParallelRunIdx int
}

func NewLoader(datastore *datastore.Datastore) Loader {
	return Loader{datastore: datastore, ParallelRunIdx: -1}
}

// Render loads and executes a manifest template.
//
// For a description of the manifest template, refer to Loader's docstring.
//
//   - tmplBytes is the manifest template, as loaded from disk.
//   - rootDir is the path of the directory in which manifest resides.
//   - ctx is the data passed on to the template's Execute function.
//     Contrary to what convention may suggest, it is not a context.Context.
func (loader *Loader) Render(
	tmplBytes []byte,
	rootDir string,
	ctx any) (res []byte, err error) {

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
		"gjson": func(path string, json string) (result string, err error) {
			if json == "" {
				err = fmt.Errorf("The given json was empty")
				return
			}

			result = gjson.Get(json, path).Raw
			if len(result) == 0 {
				err = fmt.Errorf("'%s' was not found or was empty string. Gjson Input: %s", path, json)
			}
			return
		},
		"split": func(s, sep string) []string {
			return strings.Split(s, sep)
		},
		"md5sum": func(path string) (string, error) {
			fileBytes, err := fileReadInternal(path, rootDir)
			if err != nil {
				return "", err
			}

			hasher := md5.New()
			hasher.Write([]byte(fileBytes))
			return hex.EncodeToString(hasher.Sum(nil)), nil
		},
		// "parse_csv": func(path string, delimiter rune) ([]map[string]any, error) {
		// 	_, file, err := util.OpenFileOrUrl(path, rootDir)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	fileBytes, err := io.ReadAll(file)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	data, err := csv.GenericCSVToMap(fileBytes, delimiter)
		// 	if err != nil {
		// 		return data, fmt.Errorf("'%s' %s", path, err)
		// 	}
		// 	return data, err
		// },
		"file":        loadFile(rootDir),
		"file_render": loadFileAndRender(rootDir, loader),
		"file_csv":    loadFileCSV(rootDir),
		"file_sqlite": func(path, statement string) ([]map[string]any, error) {
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
			row := make([]any, len(columns))

			data := []map[string]any{}

			for rows.Next() {
				dataEntry := map[string]any{}
				for idx, col := range columns {
					dataEntry[col.Name()] = new(any)
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
				return "", fmt.Errorf("Could not marshal xml to json: %w", err)
			}

			return string(bytes), nil
		},
		"file_xhtml2json": func(path string) (string, error) {
			fileBytes, err := fileReadInternal(path, rootDir)
			if err != nil {
				return "", err
			}

			bytes, err := util.Xhtml2Json(fileBytes)
			if err != nil {
				return "", fmt.Errorf("Could not marshal xhtml to json: %w", err)
			}

			return string(bytes), nil
		},
		"file_html2json": func(path string) (string, error) {
			fileBytes, err := fileReadInternal(path, rootDir)
			if err != nil {
				return "", err
			}

			bytes, err := util.Html2Json(fileBytes)
			if err != nil {
				return "", fmt.Errorf("Could not marshal html to json: %w", err)
			}

			return string(bytes), nil
		},
		"file_path": func(path string) string {
			return util.LocalPath(path, rootDir)
		},
		"datastore": func(index any) (any, error) {
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
		"unmarshal": func(s string) (any, error) {
			var gj any
			err := util.Unmarshal([]byte(s), &gj)
			if err != nil {
				return nil, err
			}
			return gj, nil
		},
		"N": N,
		"marshal": func(data any) (string, error) {
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
		"slice": func(args ...any) []any {
			return args
		},
		"rows_to_map": func(keyColumn, valueColumn string, rowsInput any) (map[string]any, error) {
			return rowsToMap(keyColumn, valueColumn, getRowsFromInput(rowsInput))
		},
		"pivot_rows": pivotRows,
		"group_map_rows": func(groupColumn string, rowsInput any) (map[string][]map[string]any, error) {
			grouped_rows := make(map[string][]map[string]any, 1000)
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
						grouped_rows[idx] = make([]map[string]any, 0)
					}
					grouped_rows[idx] = append(grouped_rows[idx], row)
				default:
					return nil, fmt.Errorf("Group column \"%s\" needs to be int64 but is %t.", groupColumn, idx)
				}
			}
			return grouped_rows, nil
		},
		"group_rows": func(groupColumn string, rowsInput any) ([][]map[string]any, error) {
			grouped_rows := make([][]map[string]any, 1000)
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
						grouped_rows[idx] = make([]map[string]any, 0)
					}
					grouped_rows[idx] = append(grouped_rows[idx], row)
				default:
					return nil, fmt.Errorf("Group column \"%s\" needs to be int64 but is %t.", groupColumn, idx)
				}
			}
			// remove empty rows
			g_rows := make([][]map[string]any, 0)
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
		"not_match": func(regex, text string) (bool, error) {
			match, err := regexp.Match(regex, []byte(text))
			return !match, err
		},
		"replace_host": func(srcURL string) (string, error) {
			// If no override provided, return original one
			if loader.HTTPServerHost == "" {
				return srcURL, nil
			}
			return replaceHost(srcURL, loader.HTTPServerHost)
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
		"is_zero": func(v any) bool {
			if v == nil {
				return true
			}
			return reflect.ValueOf(v).IsZero()
		},
		"oauth2_password_token": func(client string, login string, password string) (tok *oauth2.Token, err error) {
			// println("client", client, login, password)
			oAuthClient, ok := loader.OAuthClient[client]
			if !ok {
				return nil, fmt.Errorf("OAuth client %q not configured", client)
			}

			return oAuthClient.GetPasswordCredentialsAuthToken(login, password)

		},
		"oauth2_client_token": func(client string) (tok *oauth2.Token, err error) {
			oAuthClient, ok := loader.OAuthClient[client]
			if !ok {
				return nil, fmt.Errorf("OAuth client %q not configured", client)
			}

			return oAuthClient.GetClientCredentialsAuthToken()
		},
		"oauth2_code_token": func(client string, params ...string) (tok *oauth2.Token, err error) {
			oAuthClient, ok := loader.OAuthClient[client]
			if !ok {
				return nil, fmt.Errorf("OAuth client %q not configured", client)
			}

			return oAuthClient.GetCodeAuthToken(params...)
		},
		"oauth2_implicit_token": func(client string, params ...string) (tok *oauth2.Token, err error) {
			oAuthClient, ok := loader.OAuthClient[client]
			if !ok {
				return nil, fmt.Errorf("OAuth client %q not configured", client)
			}

			return oAuthClient.GetAuthToken(params...)
		},
		"oauth2_client": func(client string) (c *util.OAuthClientConfig, err error) {
			oAuthClient, ok := loader.OAuthClient[client]
			if !ok {
				return nil, fmt.Errorf("OAuth client %s not configured", client)
			}

			return &oAuthClient, nil
		},
		"oauth2_basic_auth": func(client string) (string, error) {
			oAuthClient, ok := loader.OAuthClient[client]
			if !ok {
				return "", fmt.Errorf("OAuth client %s not configured", client)
			}

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
				return 0, fmt.Errorf("version string %s is invalid", v)
			}
			if !semver.IsValid(w) {
				return 0, fmt.Errorf("version string %s is invalid", w)
			}
			return semver.Compare(v, w), nil
		},
		"log": func(msg string, args ...any) string {
			logrus.Debugf(msg, args...)
			return ""
		},
		// remove_from_url removes key from url's query part, returns
		// the new url
		"remove_from_url": func(qKey, urlStr string) (urlPatched string) {
			u, err := url.Parse(urlStr)
			if err != nil {
				return urlStr
			}
			q := u.Query()
			q.Del(qKey)
			u.RawQuery = q.Encode()
			return u.String()
		},
		// value_from_url returns the value from url's query part
		"value_from_url": func(qKey, urlStr string) string {
			u, err := url.Parse(urlStr)
			if err != nil {
				return ""
			}
			q := u.Query()
			return q.Get(qKey)
		},
		// parallel_run_idx returns the index of the Parallel Run that the current template
		// is rendered in.
		"parallel_run_idx": func() int {
			return loader.ParallelRunIdx
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

func getRowsFromInput(rowsInput any) []map[string]any {
	rows := make([]map[string]any, 0)
	switch t := rowsInput.(type) {
	case []map[string]any:
		rows = t
	case []any:
		for _, v := range t {
			rows = append(rows, v.(map[string]any))
		}
	}
	return rows
}

// replaceHost uses host of serverHost and replaces it in srcURL
func replaceHost(srcURL, serverHost string) (s string, err error) {
	defer func() {
		golib.Pln("replace %q -> %q = %q", srcURL, serverHost, s)
	}()
	if strings.Contains(serverHost, ":") {
		return "", fmt.Errorf("replaceHost: host %q must not include scheme or port", serverHost)
	}
	// Parse source URL or fail
	parsedURL, err := url.Parse(srcURL)
	if err != nil {
		return "", err
	}
	if parsedURL.Host == "" && parsedURL.Scheme != "" {
		parsedURL.Scheme = serverHost
	} else if parsedURL.Port() != "" {
		parsedURL.Host = serverHost + ":" + parsedURL.Port()
	} else {
		parsedURL.Host = serverHost
	}
	return parsedURL.String(), nil
}
