package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/programmfabrik/apitest/pkg/lib/util"
)

func buildMultipart(request Request) (additionalHeaders map[string]string, body io.Reader, err error) {
	additionalHeaders = make(map[string]string, 0)

	var buf = bytes.NewBuffer([]byte{})
	w := multipart.NewWriter(buf)

	var replaceFilename *string
	val, ok := request.Body.(map[string]any)["file:filename"]
	if ok {
		f, ok := val.(util.JsonString)
		if !ok {
			return nil, nil, fmt.Errorf("file:filename should be a string")
		}
		replaceFilename = &f
	}

	additionalHeaders["Content-Type"] = w.FormDataContentType()

	createPart := func(key string, val any) error {
		if key == "file:filename" {
			return nil
		}

		rawPathSpec, ok := val.(util.JsonString)
		if !ok {
			return fmt.Errorf("pathSpec should be a string")
		}
		pathSpec, err := util.ParsePathSpec(rawPathSpec)
		if err != nil {
			return fmt.Errorf("pathSpec %s is not valid: %w", rawPathSpec, err)
		}

		file, err := util.OpenFileOrUrl(pathSpec.Path, request.ManifestDir)
		if err != nil {
			return err
		}
		defer file.Close()

		var part io.Writer
		if replaceFilename == nil {
			part, err = w.CreateFormFile(key, pathSpec.Path)
		} else {
			part, err = w.CreateFormFile(key, *replaceFilename)
		}
		if err != nil {
			return err
		}
		if _, err := io.Copy(part, file); err != nil {
			return err
		}

		return nil
	}

	for key, val := range request.Body.(map[string]any) {
		err = createPart(key, val)
		if err != nil {
			return nil, nil, err
		}
	}

	err = w.Close()
	body = bytes.NewBuffer(buf.Bytes())

	return
}

func buildUrlencoded(request Request) (additionalHeaders map[string]string, body io.Reader, err error) {
	additionalHeaders = make(map[string]string, 0)
	additionalHeaders["Content-Type"] = "application/x-www-form-urlencoded"
	formParams := url.Values{}
	for key, value := range request.Body.(map[string]any) {
		switch v := value.(type) {
		case string:
			formParams.Set(key, v)
		case []string:
			formParams[key] = v
		default:
			formParams.Set(key, fmt.Sprintf("%s", v))
		}
	}
	body = strings.NewReader(formParams.Encode())
	return additionalHeaders, body, nil

}

func buildRegular(request Request) (additionalHeaders map[string]string, body io.Reader, err error) {
	additionalHeaders = make(map[string]string, 0)
	additionalHeaders["Content-Type"] = "application/json"

	if request.Body == nil {
		body = bytes.NewBuffer([]byte{})
	} else {
		bodyBytes, err := json.Marshal(request.Body)
		if err != nil {
			return nil, nil, fmt.Errorf("error marshaling request body: %s", err)
		}
		body = bytes.NewBuffer(bodyBytes)
	}
	return additionalHeaders, body, nil
}

// buildFile opens a file for use with buildPolicy.
// WARNING: This returns a file handle that must be closed!
func buildFile(req Request) (map[string]string, io.Reader, error) {
	headers := map[string]string{}

	if req.BodyFile == "" {
		return nil, nil, errors.New(`Request.buildFile: Missing "body_file"`)
	}

	path := req.BodyFile
	pathSpec, err := util.ParsePathSpec(req.BodyFile)
	if err == nil && pathSpec != nil { // we unwrap the path, if an @-notation path spec was passed into body_file
		path = pathSpec.Path
	}

	file, err := util.OpenFileOrUrl(path, req.ManifestDir)
	if err != nil {
		return nil, nil, err
	}
	return headers, file, err
}
