package template

import (
	"encoding/json"
	"fmt"

	"github.com/programmfabrik/apitest/pkg/lib/util"
)

func LoadManifestDataAsObject(data any, manifestDir string, loader Loader) (pathSpec *util.PathSpec, res any, err error) {
	switch typedData := data.(type) {
	case string:
		pathSpec, err = util.ParsePathSpec(typedData)
		if err != nil {
			return nil, res, fmt.Errorf("error parsing pathSpec: %w", err)
		}
		requestTmpl, err := pathSpec.LoadContents(manifestDir)
		if err != nil {
			return nil, res, fmt.Errorf("error loading fileFromPathSpec: %s", err)
		}

		// We have json, and load it thereby into our apitest structure
		requestBytes, err := loader.Render(requestTmpl, manifestDir, nil)
		if err != nil {
			return nil, res, fmt.Errorf("error rendering request: %s", err)
		}

		var jsonObject util.JsonObject
		var jsonArray util.JsonArray

		if err = util.Unmarshal(requestBytes, &jsonObject); err != nil {
			if err = util.Unmarshal(requestBytes, &jsonArray); err == nil {

				return pathSpec, jsonArray, nil
			}
			return nil, res, fmt.Errorf("error unmarshalling: %s", err)
		}
		return pathSpec, jsonObject, nil
	case util.JsonObject:
		return nil, typedData, nil
	case util.JsonArray:
		return nil, typedData, nil
	default:
		return nil, res, fmt.Errorf("specification needs to be string[@...] or jsonObject but is: %s", data)
	}
}

func LoadManifestDataAsRawJson(data any, manifestDir string) (pathSpec *util.PathSpec, res json.RawMessage, err error) {
	switch typedData := data.(type) {
	case []byte:
		err = res.UnmarshalJSON(typedData)
		return
	case string:
		pathSpec, err = util.ParsePathSpec(typedData)
		if err != nil {
			return nil, res, fmt.Errorf("error parsing pathSpec: %w", err)
		}
		res, err := pathSpec.LoadContents(manifestDir)
		if err != nil {
			return nil, res, fmt.Errorf("error loading fileFromPathSpec: %s", err)
		}
		return pathSpec, res, nil
	case util.JsonObject, util.JsonArray:
		jsonMar, err := json.Marshal(typedData)
		if err != nil {
			return nil, res, fmt.Errorf("error marshaling: %s", err)
		}
		if err = util.Unmarshal(jsonMar, &res); err != nil {
			return nil, res, fmt.Errorf("error unmarshalling: %s", err)
		}
		return nil, res, nil
	default:
		return nil, res, fmt.Errorf("specification needs to be string[@...] or jsonObject but is: %s", data)
	}
}
