package template

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/programmfabrik/apitest/pkg/lib/util"
)

func loadFileFromPathSpec(pathSpec, manifestDir string) (string, []byte, error) {
	if !util.IsPathSpec([]byte(pathSpec)) {
		return "", nil, fmt.Errorf("spec was expected to be path spec, got %s instead", pathSpec)
	}

	filepath, requestFile, err := util.OpenFileOrUrl(pathSpec, manifestDir)
	if err != nil {
		return "", nil, fmt.Errorf("error opening path: %s", err)
	}

	defer requestFile.Close()
	requestTmpl, err := ioutil.ReadAll(requestFile)

	if err != nil {
		return "", nil, fmt.Errorf("error loading file %s: %s", requestFile, err)
	}

	return filepath, requestTmpl, nil
}

func LoadManifestDataAsObject(data interface{}, manifestDir string, loader Loader) (filepath string, res interface{}, err error) {
	switch typedData := data.(type) {
	case string:
		filepath, requestTmpl, err := loadFileFromPathSpec(typedData, manifestDir)
		if err != nil {
			return "", res, fmt.Errorf("error loading fileFromPathSpec: %s", err)
		}

		// We have json, and load it thereby into our apitest structure
		requestBytes, err := loader.Render(requestTmpl, manifestDir, nil)
		if err != nil {
			return "", res, fmt.Errorf("error rendering request: %s", err)
		}

		var jsonObject util.JsonObject
		var jsonArray util.JsonArray

		if err = util.Unmarshal(requestBytes, &jsonObject); err != nil {
			if err = util.Unmarshal(requestBytes, &jsonArray); err == nil {

				return filepath, jsonArray, nil
			}
			return "", res, fmt.Errorf("error unmarshalling: %s", err)
		}
		return filepath, jsonObject, nil
	case util.JsonObject:
		return "", typedData, nil
	case util.JsonArray:
		return "", typedData, nil
	default:
		return "", res, fmt.Errorf("specification needs to be string[@...] or jsonObject but is: %s", data)
	}
}

func LoadManifestDataAsRawJson(data interface{}, manifestDir string) (filepath string, res json.RawMessage, err error) {
	switch typedData := data.(type) {
	case []byte:
		err = res.UnmarshalJSON(typedData)
		return
	case string:
		filepath, res, err := loadFileFromPathSpec(typedData, manifestDir)
		if err != nil {
			return "", res, fmt.Errorf("error loading fileFromPathSpec: %s", err)
		}
		return filepath, res, nil
	case util.JsonObject, util.JsonArray:
		jsonMar, err := json.Marshal(typedData)
		if err != nil {
			return "", res, fmt.Errorf("error marshaling: %s", err)
		}
		if err = util.Unmarshal(jsonMar, &res); err != nil {
			return "", res, fmt.Errorf("error unmarshalling: %s", err)
		}
		return "", res, nil
	default:
		return "", res, fmt.Errorf("specification needs to be string[@...] or jsonObject but is: %s", data)
	}
}
