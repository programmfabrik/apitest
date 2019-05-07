package api

import (
	"fmt"
	"github.com/programmfabrik/fylr-apitest/lib/datastore"
	"github.com/programmfabrik/fylr-apitest/lib/util"
	"time"

	"io"
	"net/http"
	"net/http/httputil"
)

var c http.Client

func init() {
	c = http.Client{
		Timeout: time.Minute * 5,
	}
}

type Request struct {
	Endpoint        string                 `yaml:"endpoint" json:"endpoint"`
	ServerURL       string                 `yaml:"serverurl" json:"serverurl"`
	Method          string                 `yaml:"method" json:"method"`
	QueryParams     map[string]interface{} `yaml:"query_params" json:"query_params"`
	Headers         map[string]*string     `yaml:"header" json:"header"`
	HeaderFromStore map[string]string      `yaml:"header_from_store" json:"header_from_store"`
	BodyType        string                 `yaml:"body_type" json:"body_type"`
	Body            util.GenericJson       `yaml:"body" json:"body"`

	buildPolicy func(Request) (additionalHeaders map[string]string, body io.Reader, err error)
	DoNotStore  bool
	ManifestDir string
	DataStore   *datastore.Datastore
}

func (request Request) buildHttpRequest() (res *http.Request, err error) {
	if request.buildPolicy == nil {
		//Set Build policy
		switch request.BodyType {
		case "multipart":
			request.buildPolicy = buildMultipart
		case "urlencoded":
			request.buildPolicy = buildUrlencoded
		default:
			request.buildPolicy = buildRegular
		}
	}
	//Render Request Url
	requestUrl := fmt.Sprintf("%s/%s", request.ServerURL, request.Endpoint)

	additionalHeaders, body, err := request.buildPolicy(request)
	if err != nil {
		return res, fmt.Errorf("error executing buildpolicy: %s", err)
	}
	res, err = http.NewRequest(request.Method, requestUrl, body)
	if err != nil {
		return res, fmt.Errorf("error creating new request")
	}

	q := res.URL.Query()
	for key, val := range request.QueryParams {
		stringVal, err := util.GetStringFromInterface(val)
		if err != nil {
			return res, fmt.Errorf("error GetStringFromInterface: %s", err)
		}
		q.Add(key, stringVal)
	}
	res.URL.RawQuery = q.Encode()

	for key, val := range additionalHeaders {
		res.Header.Add(key, val)
	}

	for headerName, datastoreKey := range request.HeaderFromStore {
		if request.DataStore == nil {
			return res, fmt.Errorf("can't get header_from_store as the datastore is nil")
		}

		headersInt, err := request.DataStore.Get(datastoreKey)
		if err != nil {
			return nil, fmt.Errorf("could not get '%s' from Datastore: %s", datastoreKey, err)
		}

		ownHeaders, ok := headersInt.([]interface{})
		if ok {
			for _, val := range ownHeaders {
				valString, ok := val.(string)
				if ok {
					res.Header.Add(headerName, valString)
				}
			}
			continue
		}

		ownHeader, ok := headersInt.(string)
		if ok {
			res.Header.Add(headerName, ownHeader)
		} else {
			return nil, fmt.Errorf("could not set header '%s' from Datastore: '%s' is not a string. Got value: '%v'", headerName, datastoreKey, headersInt)
		}
	}

	for key, val := range request.Headers {
		if *val == "" {
			//Unset header explicit
			res.Header.Del(key)
		} else {
			//ADD header
			res.Header.Set(key, *val)
		}
	}

	return res, nil
}

func (request Request) ToString() (res string) {
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
	resBytes, err := httputil.DumpRequestOut(httpRequest, dumpBody)
	if err != nil {
		return fmt.Sprintf("could not dump httpRequest: %s", err)
	}
	return string(resBytes)
}

func (request Request) Send() (response Response, err error) {
	httpRequest, err := request.buildHttpRequest()
	if err != nil {
		return response, err
	}

	httpResponse, err := c.Do(httpRequest)
	if err != nil {
		return response, err
	}

	response, err = NewResponse(httpResponse.StatusCode, httpResponse.Header, httpResponse.Body)
	if err != nil {
		return response, fmt.Errorf("error constructing response from http response")
	}
	return response, err
}
