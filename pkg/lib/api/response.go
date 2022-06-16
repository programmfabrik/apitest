package api

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"

	"github.com/programmfabrik/apitest/pkg/lib/csv"
	"github.com/programmfabrik/apitest/pkg/lib/util"
)

type Response struct {
	StatusCode  int
	Headers     map[string][]string
	Cookies     []*http.Cookie
	Body        []byte
	bodyControl util.JsonObject
	Format      ResponseFormat

	ReqDur      time.Duration
	BodyLoadDur time.Duration
}

func (res Response) NeedsCheck() bool {
	if res.StatusCode != http.StatusOK {
		return true
	}
	if len(res.Headers) > 0 || len(res.Cookies) > 0 || len(res.Body) > 0 {
		return true
	}
	return false
}

// Cookie definition
type Cookie struct {
	Name     string        `json:"name"`
	Value    string        `json:"value"`
	Path     string        `json:"path,omitempty"`
	Domain   string        `json:"domain,omitempty"`
	Expires  time.Time     `json:"expires,omitempty"`
	MaxAge   int           `json:"max_age,omitempty"`
	Secure   bool          `json:"secure,omitempty"`
	HttpOnly bool          `json:"http_only,omitempty"`
	SameSite http.SameSite `json:"same_site,omitempty"`
}

type ResponseSerialization struct {
	StatusCode  int                 `yaml:"statuscode" json:"statuscode"`
	Headers     map[string][]string `yaml:"header" json:"header,omitempty"`
	Cookies     map[string]Cookie   `yaml:"cookie" json:"cookie,omitempty"`
	Body        interface{}         `yaml:"body" json:"body,omitempty"`
	BodyControl util.JsonObject     `yaml:"body:control" json:"body:control,omitempty"`
	Format      ResponseFormat      `yaml:"format" json:"format,omitempty"`
}

type ResponseFormat struct {
	IgnoreBody bool   `json:"-"`    // if true, do not try to parse the body (since it is not expected in the response)
	Type       string `json:"type"` // default "json", allowed: "csv", "json", "xml", "binary"
	CSV        struct {
		Comma string `json:"comma,omitempty"`
	} `json:"csv,omitempty"`
	PreProcess *PreProcess `json:"pre_process,omitempty"`
}

func NewResponse(statusCode int, headers map[string][]string, cookies []*http.Cookie, body io.Reader, bodyControl util.JsonObject, bodyFormat ResponseFormat) (res Response, err error) {
	res = Response{
		StatusCode:  statusCode,
		Headers:     headers,
		Cookies:     cookies,
		bodyControl: bodyControl,
		Format:      bodyFormat,
	}
	if body != nil {
		start := time.Now()
		res.Body, err = ioutil.ReadAll(body)
		if err != nil {
			return res, err
		}
		res.BodyLoadDur = time.Since(start)
	}
	return res, nil
}

func NewResponseFromSpec(spec ResponseSerialization) (res Response, err error) {
	var body io.Reader
	if spec.Body != nil {
		bodyBytes, err := json.Marshal(spec.Body)
		if err != nil {
			return res, err
		}
		body = bytes.NewReader(bodyBytes)
	}
	// if statuscode is not explicitly set; we assume 200
	if spec.StatusCode == 0 {
		spec.StatusCode = 200
	}

	// Build standard cookies bag from spec map
	var cookies []*http.Cookie
	if len(spec.Cookies) > 0 {
		cookies = make([]*http.Cookie, 0)
		for _, rck := range spec.Cookies {
			cookies = append(cookies, &http.Cookie{
				Name:     rck.Name,
				Value:    rck.Value,
				Path:     rck.Path,
				Domain:   rck.Domain,
				Expires:  rck.Expires,
				MaxAge:   rck.MaxAge,
				Secure:   rck.Secure,
				HttpOnly: rck.HttpOnly,
				SameSite: rck.SameSite,
			})
		}
	}

	return NewResponse(spec.StatusCode, spec.Headers, cookies, body, spec.BodyControl, spec.Format)
}

// ServerResponseToGenericJSON parse response from server. convert xml, csv, binary to json if necessary
func (response Response) ServerResponseToGenericJSON(responseFormat ResponseFormat, bodyOnly bool) (interface{}, error) {
	var (
		res, bodyJSON interface{}
		bodyData      []byte
		err           error
		resp          Response
	)

	if responseFormat.PreProcess != nil {
		resp, err = responseFormat.PreProcess.RunPreProcess(response)
		if err != nil {
			return res, errors.Wrap(err, "Could not pre process response")
		}
	} else {
		resp = response
	}

	switch responseFormat.Type {
	case "xml", "xml2":
		bodyData, err = util.Xml2Json(resp.Body, responseFormat.Type)
		if err != nil {
			return res, errors.Wrap(err, "Could not marshal xml to json")
		}
	case "csv":
		runeComma := ','
		if responseFormat.CSV.Comma != "" {
			runeComma = []rune(responseFormat.CSV.Comma)[0]
		}

		csvData, err := csv.GenericCSVToMap(resp.Body, runeComma)
		if err != nil {
			return res, errors.Wrap(err, "Could not parse csv")
		}

		bodyData, err = json.Marshal(csvData)
		if err != nil {
			return res, errors.Wrap(err, "Could not marshal csv to json")
		}
	case "binary":
		// We have another file format (binary). We thereby take the md5 Hash of the body and compare that one
		hasher := md5.New()
		hasher.Write([]byte(resp.Body))
		JsonObject := util.JsonObject{
			"md5sum": util.JsonString(hex.EncodeToString(hasher.Sum(nil))),
		}
		bodyData, err = json.Marshal(JsonObject)
		if err != nil {
			return res, errors.Wrap(err, "Could not marshal body with md5sum to json")
		}
	case "":
		// no specific format, we assume a json, and thereby try to unmarshal it into our body
		bodyData = resp.Body
	default:
		return res, fmt.Errorf("Invalid response format '%s'", responseFormat.Type)
	}

	responseJSON := ResponseSerialization{
		StatusCode: resp.StatusCode,
	}
	if len(resp.Headers) > 0 {
		responseJSON.Headers = resp.Headers
	}
	// Build cookies map from standard bag
	if len(resp.Cookies) > 0 {
		responseJSON.Cookies = make(map[string]Cookie)
		for _, ck := range resp.Cookies {
			if ck != nil {
				responseJSON.Cookies[ck.Name] = Cookie{
					Name:     ck.Name,
					Value:    ck.Value,
					Path:     ck.Path,
					Domain:   ck.Domain,
					Expires:  ck.Expires,
					MaxAge:   ck.MaxAge,
					Secure:   ck.Secure,
					HttpOnly: ck.HttpOnly,
					SameSite: ck.SameSite,
				}
			}
		}
	}

	// if the body should not be ignored, serialize the parsed/converted body
	if !responseFormat.IgnoreBody {

		if len(bodyData) > 0 {
			err = json.Unmarshal(bodyData, &bodyJSON)
			if err != nil {
				return res, err
			}
		}

		if bodyOnly {
			return bodyJSON, nil
		}

		responseJSON.Body = &bodyJSON
	}

	responseBytes, err := json.Marshal(responseJSON)
	if err != nil {
		return res, err
	}
	json.Unmarshal(responseBytes, &res)

	return res, nil
}

// ToGenericJSON parse expected response
func (response Response) ToGenericJSON() (interface{}, error) {
	var (
		bodyJSON, res interface{}
		err           error
	)

	// We have a json, and thereby try to unmarshal it into our body
	resBody := response.Body
	if len(resBody) > 0 {
		err = json.Unmarshal(resBody, &bodyJSON)
		if err != nil {
			return res, err
		}
	}

	responseJSON := ResponseSerialization{
		StatusCode:  response.StatusCode,
		BodyControl: response.bodyControl,
	}
	if len(response.Headers) > 0 {
		responseJSON.Headers = response.Headers
	}

	// Build cookies map from standard bag
	if len(response.Cookies) > 0 {
		responseJSON.Cookies = make(map[string]Cookie)
		for _, ck := range response.Cookies {
			if ck != nil {
				responseJSON.Cookies[ck.Name] = Cookie{
					Name:     ck.Name,
					Value:    ck.Value,
					Path:     ck.Path,
					Domain:   ck.Domain,
					Expires:  ck.Expires,
					MaxAge:   ck.MaxAge,
					Secure:   ck.Secure,
					HttpOnly: ck.HttpOnly,
					SameSite: ck.SameSite,
				}
			}
		}
	}

	// necessary because check for <nil> against missing body would fail, but must succeed
	if bodyJSON != nil {
		responseJSON.Body = &bodyJSON
	}

	responseBytes, err := json.Marshal(responseJSON)
	if err != nil {
		return res, err
	}
	json.Unmarshal(responseBytes, &res)

	return res, nil
}

func (response Response) ServerResponseToJsonString(bodyOnly bool) (string, error) {
	genericJSON, err := response.ServerResponseToGenericJSON(response.Format, bodyOnly)
	if err != nil {
		return "", fmt.Errorf("error formatting response: %s", err)
	}
	bytes, err := json.MarshalIndent(genericJSON, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error formatting response: %s", err)
	}
	return string(bytes), nil
}

func (response *Response) marshalBodyInto(target interface{}) (err error) {
	bodyBytes := response.Body
	if len(bodyBytes) > 0 {
		if err = json.Unmarshal(bodyBytes, target); err != nil {
			return fmt.Errorf("error unmarshaling response: %s", err)
		}
	}
	return nil
}

func (response Response) ToString() string {
	var (
		headersString string
		bodyString    string
		err           error
		resp          Response
	)

	for k, v := range response.Headers {
		value := ""
		for _, iv := range v {
			value = fmt.Sprintf("%s %s", value, iv)
		}
		if strings.TrimSpace(value) == "" {
			continue
		}
		headersString = fmt.Sprintf("%s\n%s:%s", headersString, k, value)
	}

	if response.Format.PreProcess != nil {
		resp, err = response.Format.PreProcess.RunPreProcess(response)
		if err != nil {
			resp = response
		}
	} else {
		resp = response
	}

	// for logging, always show the body
	resp.Format.IgnoreBody = false

	body := resp.Body
	switch resp.Format.Type {
	case "xml", "csv":
		if utf8.Valid(body) {
			bodyString, err = resp.ServerResponseToJsonString(true)
			if err != nil {
				bodyString = string(body)
			}
		} else {
			bodyString = fmt.Sprintf("[BINARY DATA NOT DISPLAYED]\n\n")
		}
	case "binary":
		resp.Format.IgnoreBody = false
		bodyString, err = resp.ServerResponseToJsonString(true)
		if err != nil {
			bodyString = string(body)
		}
	default:
		if utf8.Valid(body) {
			bodyString = string(body)
		} else {
			bodyString = fmt.Sprintf("[BINARY DATA NOT DISPLAYED]\n\n")
		}
	}

	return fmt.Sprintf("%d\n%s\n\n%s", resp.StatusCode, headersString, bodyString)
}
