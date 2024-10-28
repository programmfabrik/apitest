package api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/moul/http2curl"
	"github.com/programmfabrik/apitest/pkg/lib/datastore"
	"github.com/programmfabrik/apitest/pkg/lib/util"
)

var httpClient *http.Client

func init() {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		// We need to avoid the automatically set gzip header
		DisableCompression: true,
	}

	httpClient = &http.Client{
		Timeout:   time.Minute * 5,
		Transport: tr,
	}
}

type RequestCookie struct {
	ValueFromStore string `yaml:"value_from_store" json:"value_from_store"`
	Value          string `yaml:"value" json:"value"`
}

type Request struct {
	Endpoint             string                    `yaml:"endpoint" json:"endpoint"`
	ServerURL            string                    `yaml:"server_url" json:"server_url"`
	Method               string                    `yaml:"method" json:"method"`
	NoRedirect           bool                      `yaml:"no_redirect" json:"no_redirect"`
	QueryParams          map[string]any            `yaml:"query_params" json:"query_params"`
	QueryParamsFromStore map[string]string         `yaml:"query_params_from_store" json:"query_params_from_store"`
	Headers              map[string]*string        `yaml:"header" json:"header"`
	HeaderFromStore      map[string]string         `yaml:"header_from_store" json:"header_from_store"`
	Cookies              map[string]*RequestCookie `yaml:"cookies" json:"cookies"`
	SetCookies           []*Cookie                 `yaml:"header-x-test-set-cookie" json:"header-x-test-set-cookie"`
	BodyType             string                    `yaml:"body_type" json:"body_type"`
	BodyFile             string                    `yaml:"body_file" json:"body_file"`
	Body                 any                       `yaml:"body" json:"body"`

	buildPolicy func(Request) (additionalHeaders map[string]string, body io.Reader, err error)
	ManifestDir string
	DataStore   *datastore.Datastore
}

func (request Request) buildHttpRequest() (req *http.Request, err error) {
	if request.buildPolicy == nil {
		//Set Build policy
		switch request.BodyType {
		case "multipart":
			request.buildPolicy = buildMultipart
		case "urlencoded":
			request.buildPolicy = buildUrlencoded
		case "file":
			request.buildPolicy = buildFile
		default:
			request.buildPolicy = buildRegular
		}
	}
	//Render Request Url

	requestUrl := fmt.Sprintf("%s/%s", request.ServerURL, request.Endpoint)
	if request.Endpoint == "" {
		requestUrl = request.ServerURL
	}

	reqUrl, err := url.Parse(requestUrl)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to buildHttpRequest with URL %q", requestUrl)
	}

	// Note that buildPolicy may return a file handle that needs to be
	// closed. According to standard library documentation, the NewRequest
	// call below will take into account if body also happens to implement
	// io.Closer.
	additionalHeaders, body, err := request.buildPolicy(request)
	if err != nil {
		return req, fmt.Errorf("error executing buildpolicy: %s", err)
	}

	req, err = http.NewRequest(request.Method, requestUrl, body)
	if err != nil {
		return req, fmt.Errorf("error creating new request")
	}

	// Remove library default agent
	req.Header.Set("User-Agent", "")
	req.Close = true

	if reqUrl.User != nil {
		pw, ok := reqUrl.User.Password()
		if ok {
			req.SetBasicAuth(reqUrl.User.Username(), pw)
		}
		req.URL.User = nil
	}

	q := req.URL.Query()

	for queryName, datastoreKey := range request.QueryParamsFromStore {
		skipOnError := false
		if len(datastoreKey) > 0 && datastoreKey[0] == '?' {
			skipOnError = true
			datastoreKey = datastoreKey[1:]
		}

		if request.DataStore == nil {
			return req, fmt.Errorf("can't get header_from_store as the datastore is nil")
		}

		queryParamInterface, err := request.DataStore.Get(datastoreKey)
		if err != nil {
			if skipOnError {
				continue
			}
			return nil, fmt.Errorf("could not get '%s' from Datastore: %s", datastoreKey, err)
		}

		stringVal, err := util.GetStringFromInterface(queryParamInterface)
		if err != nil {
			return req, fmt.Errorf("error GetStringFromInterface: %s", err)
		}

		if stringVal == "" {
			continue
		}
		q.Add(queryName, stringVal)
	}

	for key, val := range request.QueryParams {
		stringVal, err := util.GetStringFromInterface(val)
		if err != nil {
			return req, fmt.Errorf("error GetStringFromInterface: %s", err)
		}
		q.Set(key, stringVal)
	}

	req.URL.RawQuery = q.Encode()

	for key, val := range additionalHeaders {
		req.Header.Add(key, val)
	}

	for headerName, datastoreKey := range request.HeaderFromStore {
		skipOnError := false
		if len(datastoreKey) > 0 && datastoreKey[0] == '?' {
			skipOnError = true
			datastoreKey = datastoreKey[1:]
		}

		if request.DataStore == nil {
			return req, fmt.Errorf("can't get header_from_store as the datastore is nil")
		}

		headersInt, err := request.DataStore.Get(datastoreKey)
		if err != nil {
			if skipOnError {
				continue
			}
			return nil, fmt.Errorf("could not get '%s' from Datastore: %s", datastoreKey, err)
		}

		ownHeaders, ok := headersInt.([]any)
		if ok {
			for _, val := range ownHeaders {
				valString, ok := val.(string)
				if ok {
					if valString == "" {
						continue
					}
					req.Header.Add(headerName, valString)
				}
			}
			continue
		}

		ownHeader, ok := headersInt.(string)
		if ok {
			if ownHeader == "" {
				continue
			}
			req.Header.Set(headerName, ownHeader)
		} else {
			return nil, fmt.Errorf("could not set header '%s' from Datastore: '%s' is not a string. Got value: '%v'", headerName, datastoreKey, headersInt)
		}
	}

	for key, val := range request.Headers {
		if *val == "" {
			//Unset header explicit
			req.Header.Del(key)
		} else {
			//ADD header
			req.Header.Set(key, *val)
		}
	}

	for ckName, reqCookie := range request.Cookies {
		if reqCookie == nil {
			continue
		}
		var ck http.Cookie
		storeKey := reqCookie.ValueFromStore

		// Get cookie from store
		if len(storeKey) > 0 && request.DataStore != nil {
			cookieInt, err := request.DataStore.Get(storeKey)
			if err == nil && cookieInt != "" {
				ckBytes, err := json.Marshal(cookieInt)
				if err != nil {
					return nil, fmt.Errorf("could not marshal cookie '%s' from Datastore", storeKey)
				}
				err = json.Unmarshal(ckBytes, &ck)
				if err != nil {
					return nil, fmt.Errorf("could not unmarshal cookie '%s' from Datastore: %s", storeKey, string(ckBytes))
				}
			}
		}

		// Override with specific values
		if reqCookie.Value != "" {
			ck.Value = reqCookie.Value
		}
		ck.Name = ckName
		req.AddCookie(&ck)
	}

	// Add to custom header cookies to set in server
	for _, v := range request.SetCookies {
		if v == nil {
			continue
		}
		ck := http.Cookie{
			Name:     v.Name,
			Value:    v.Value,
			Path:     v.Path,
			Domain:   v.Domain,
			Expires:  v.Expires,
			MaxAge:   v.MaxAge,
			Secure:   v.Secure,
			HttpOnly: v.HttpOnly,
			SameSite: v.SameSite,
		}
		ckVal := ck.String()
		if ckVal == "" {
			return nil, fmt.Errorf("Invalid cookie to set server-side: %v", v)
		}
		req.Header.Add("X-Test-Set-Cookies", ckVal)
	}

	return req, nil
}

func (request Request) ToString(curl bool) (res string) {
	httpRequest, err := request.buildHttpRequest()
	if err != nil {
		return fmt.Sprintf("could not build httpRequest: %s", err)
	}

	var dumpBody bool
	if request.BodyType == "multipart" {
		dumpBody = false
	} else {
		dumpBody = true
	}

	if curl {
		// Log as curl
		// r := strings.NewReplacer(" -", " \\\n-", "' '", "' \\\n'")

		if dumpBody {
			curl, _ := http2curl.GetCurlCommand(httpRequest)
			return curl.String()
			// return r.Replace(curl.String())
		}

		_, _ = io.Copy(io.Discard, httpRequest.Body)
		_ = httpRequest.Body.Close()

		curl, _ := http2curl.GetCurlCommand(httpRequest)
		cString := curl.String()

		rep := ""
		for key, val := range request.Body.(map[string]any) {
			pathSpec, ok := val.(util.JsonString)
			if !ok {
				panic(fmt.Errorf("pathSpec should be a string"))
			}
			rep = fmt.Sprintf(`%s -F "%s=@%s"`, rep, key, path.Join(request.ManifestDir, pathSpec[1:]))
		}
		// return r.Replace(strings.Replace(cString, " -d ''", rep, 1))
		return strings.Replace(cString, " -d ''", rep, 1)
	}

	resBytes, err := httputil.DumpRequestOut(httpRequest, dumpBody)
	if err != nil {
		return fmt.Sprintf("could not dump httpRequest: %s", err)
	}
	return string(resBytes)
}

func (request Request) Send() (response Response, err error) {
	httpRequest, err := request.buildHttpRequest()
	if err != nil {
		return response, fmt.Errorf("Could not buildHttpRequest: %s", err)
	}

	if request.NoRedirect {
		httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else {
		httpClient.CheckRedirect = nil
	}

	now := time.Now()

	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		return response, fmt.Errorf("Could not do http request: %s", err)
	}

	elapsedTime := time.Since(now)

	defer httpResponse.Body.Close()

	header, err := HttpHeaderToMap(httpResponse.Header)
	if err != nil {
		return response, err
	}
	response, err = NewResponse(httpResponse.StatusCode, header, nil, httpResponse.Cookies(), httpResponse.Body, ResponseFormat{})
	if err != nil {
		return response, fmt.Errorf("error constructing response from http response")
	}
	response.ReqDur = elapsedTime
	return response, err
}
