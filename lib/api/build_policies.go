package api

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"strings"

	"github.com/programmfabrik/fylr-apitest/lib/cjson"
	"github.com/programmfabrik/fylr-apitest/lib/util"
)

func buildMultipart(request Request) (additionalHeaders map[string]string, body io.Reader, err error) {
	additionalHeaders = make(map[string]string)

	var buf = bytes.NewBuffer([]byte{})
	w := multipart.NewWriter(buf)

	additionalHeaders["Content-Type"] = w.FormDataContentType()
	for key, val := range request.Body.(map[string]interface{}) {
		pathSpec, ok := val.(util.JsonString)
		if !ok {
			return additionalHeaders, body, fmt.Errorf("pathSpec should be a string")
		}
		if !util.IsPathSpec(pathSpec) {
			return additionalHeaders, body, fmt.Errorf("pathSpec %s is not valid", pathSpec)
		}

		_, file, err := util.OpenFileOrUrl(pathSpec, request.ManifestDir)
		if err != nil {
			return additionalHeaders, nil, err
		}
		part, err := w.CreateFormFile(key, pathSpec[1:])
		if err != nil {
			return additionalHeaders, nil, err
		}
		if _, err := io.Copy(part, file); err != nil {
			return additionalHeaders, nil, err
		}
	}
	w.Close()

	body = bytes.NewBuffer(buf.Bytes())
	return additionalHeaders, body, nil
}

func buildUrlencoded(request Request) (additionalHeaders map[string]string, body io.Reader, err error) {
	additionalHeaders = make(map[string]string)
	additionalHeaders["Content-Type"] = "application/x-www-form-urlencoded"
	formParams := url.Values{}
	for key, value := range request.Body.(map[string]string) {
		formParams.Add(key, value)
	}
	body = strings.NewReader(formParams.Encode())
	return additionalHeaders, body, nil

}

func buildRegular(request Request) (additionalHeaders map[string]string, body io.Reader, err error) {
	additionalHeaders = make(map[string]string)
	additionalHeaders["Content-Type"] = "application/json"
	bodyBytes, err := cjson.Marshal(request.Body)
	if err != nil {
		return additionalHeaders, body, fmt.Errorf("error marshaling request body: %s", err)
	}
	body = bytes.NewBuffer(bodyBytes)
	return additionalHeaders, body, nil
}
