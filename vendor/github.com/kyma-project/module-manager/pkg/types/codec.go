package types

import (
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
	"github.com/xeipuuv/gojsonschema"
	"sigs.k8s.io/yaml"
)

// +kubebuilder:object:generate=false
type Codec struct {
	imageSpecSchema     *gojsonschema.Schema
	helmChartSpecSchema *gojsonschema.Schema
	kustomizeSpecSchema *gojsonschema.Schema
}

func NewCodec() (*Codec, error) {
	imageSpecJSONBytes := jsonschema.Reflect(ImageSpec{})
	bytes, err := imageSpecJSONBytes.MarshalJSON()
	if err != nil {
		return nil, err
	}

	imageSpecSchema, err := gojsonschema.NewSchema(gojsonschema.NewBytesLoader(bytes))
	if err != nil {
		return nil, err
	}

	helmChartSpecJSONBytes := jsonschema.Reflect(HelmChartSpec{})
	bytes, err = helmChartSpecJSONBytes.MarshalJSON()
	if err != nil {
		return nil, err
	}

	helmChartSpecSchema, err := gojsonschema.NewSchema(gojsonschema.NewBytesLoader(bytes))
	if err != nil {
		return nil, err
	}

	kustomizeSpecJSONBytes := jsonschema.Reflect(KustomizeSpec{})
	bytes, err = kustomizeSpecJSONBytes.MarshalJSON()
	if err != nil {
		return nil, err
	}

	kustomizeSpecSchema, err := gojsonschema.NewSchema(gojsonschema.NewBytesLoader(bytes))
	if err != nil {
		return nil, err
	}

	return &Codec{
		imageSpecSchema:     imageSpecSchema,
		helmChartSpecSchema: helmChartSpecSchema,
		kustomizeSpecSchema: kustomizeSpecSchema,
	}, nil
}

func GetSpecType(data []byte) (RefTypeMetadata, error) {
	raw := make(map[string]json.RawMessage)
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return "", err
	}

	var refType RefTypeMetadata
	if err := yaml.Unmarshal(raw["type"], &refType); err != nil {
		return "", err
	}

	return refType, nil
}

func (c *Codec) Decode(data []byte, obj interface{}, refType RefTypeMetadata) error {
	if err := c.Validate(data, refType); err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, &obj); err != nil {
		return err
	}

	return nil
}

func (c *Codec) Validate(data []byte, refType RefTypeMetadata) error {
	dataBytes := gojsonschema.NewBytesLoader(data)
	var result *gojsonschema.Result
	var err error

	switch refType {
	case HelmChartType:
		result, err = c.helmChartSpecSchema.Validate(dataBytes)
		if err != nil {
			return err
		}
	case OciRefType:
		result, err = c.imageSpecSchema.Validate(dataBytes)
		if err != nil {
			return err
		}
	case KustomizeType:
		result, err = c.kustomizeSpecSchema.Validate(dataBytes)
		if err != nil {
			return err
		}
	case NilRefType:
		return fmt.Errorf("unsupported %s passed as installation type", refType)
	}

	if !result.Valid() {
		errorString := ""
		for _, err := range result.Errors() {
			errorString = fmt.Sprintf("%s: %s", errorString, err.String())
		}
		return fmt.Errorf(errorString)
	}
	return nil
}
