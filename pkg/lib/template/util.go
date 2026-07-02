package template

import (
	"fmt"

	"github.com/programmfabrik/apitest/pkg/lib/jsutil"
	"github.com/programmfabrik/apitest/pkg/lib/util"
)

func LoadManifestDataAsObject(data any, manifestDir string, loader Loader) (pathSpec *util.PathSpec, res any, err error) {
	switch typedData := data.(type) {
	case string:
		pathSpec, err = util.ParsePathSpec(typedData)
		if err != nil {
			return nil, res, fmt.Errorf("parsing pathSpec: %w", err)
		}
		requestTmpl, err := pathSpec.LoadContents(manifestDir)
		if err != nil {
			return nil, res, fmt.Errorf("loading fileFromPathSpec: %w", err)
		}

		// We have json, and load it thereby into our apitest structure
		requestBytes, err := loader.Render(requestTmpl, manifestDir, nil)
		if err != nil {
			return nil, res, fmt.Errorf("rendering request: %w", err)
		}

		// parse once and dispatch on the type, instead of parsing up to
		// twice (object first, array on error)
		var jsonData any
		if err = jsutil.Unmarshal(requestBytes, &jsonData); err != nil {
			return nil, res, fmt.Errorf("unmarshalling: %w", err)
		}
		switch typedJson := jsonData.(type) {
		case jsutil.Object:
			return pathSpec, typedJson, nil
		case jsutil.Array:
			return pathSpec, typedJson, nil
		}
		return nil, res, fmt.Errorf("unmarshalling: %s does not contain a json object or array", pathSpec.Path)
	case jsutil.Object:
		return nil, typedData, nil
	case jsutil.Array:
		return nil, typedData, nil
	default:
		return nil, res, fmt.Errorf("specification needs to be string[@...] or jsonObject but is: %v", data)
	}
}

func LoadManifestDataAsRawJson(data any, manifestDir string) (pathSpec *util.PathSpec, res jsutil.RawMessage, err error) {
	switch typedData := data.(type) {
	case []byte:
		err = res.UnmarshalJSON(typedData)
		return
	case string:
		pathSpec, err = util.ParsePathSpec(typedData)
		if err != nil {
			return nil, res, fmt.Errorf("parsing pathSpec: %w", err)
		}
		res, err := pathSpec.LoadContents(manifestDir)
		if err != nil {
			return nil, res, fmt.Errorf("loading fileFromPathSpec: %w", err)
		}
		return pathSpec, res, nil
	case jsutil.Object, jsutil.Array:
		jsonMar, err := jsutil.Marshal(typedData)
		if err != nil {
			return nil, res, fmt.Errorf("marshaling: %w", err)
		}
		return nil, jsonMar, nil
	default:
		return nil, res, fmt.Errorf("specification needs to be string[@...] or jsonObject but is: %v", data)
	}
}
