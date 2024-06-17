package template

import (
	"encoding/json"
	"fmt"

	"github.com/programmfabrik/apitest/pkg/lib/util"
)

func LoadManifestDataAsObject(data any, manifestDir string, loader Loader) (filepath string, res any, err error) {
	switch typedData := data.(type) {
	case string:
		pathSpec, err := util.ParsePathSpec(typedData)
		if err != nil {
			return "", res, fmt.Errorf("error parsing pathSpec: %w", err)
		}
		requestTmpl, err := pathSpec.LoadContents(manifestDir)
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
		return pathSpec.Path, jsonObject, nil
	case util.JsonObject:
		return "", typedData, nil
	case util.JsonArray:
		return "", typedData, nil
	default:
		return "", res, fmt.Errorf("specification needs to be string[@...] or jsonObject but is: %s", data)
	}
}

func LoadManifestDataAsRawJson(data any, manifestDir string) (filepath string, res json.RawMessage, err error) {
	switch typedData := data.(type) {
	case []byte:
		err = res.UnmarshalJSON(typedData)
		return
	case string:
		pathSpec, err := util.ParsePathSpec(typedData)
		if err != nil {
			return "", res, fmt.Errorf("error parsing pathSpec: %w", err)
		}
		res, err := pathSpec.LoadContents(manifestDir)
		if err != nil {
			return "", res, fmt.Errorf("error loading fileFromPathSpec: %s", err)
		}
		return pathSpec.Path, res, nil
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
