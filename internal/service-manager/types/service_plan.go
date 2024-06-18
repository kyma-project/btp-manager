package types

import (
	"encoding/json"
)

type ServicePlans struct {
	ServicePlans []ServicePlan `json:"items" yaml:"items"`
}

type ServicePlan struct {
	Common
	CatalogID     string `json:"catalog_id,omitempty" yaml:"catalog_id,omitempty"`
	CatalogName   string `json:"catalog_name,omitempty" yaml:"catalog_name,omitempty"`
	Free          bool   `json:"free,omitempty" yaml:"free,omitempty"`
	Bindable      bool   `json:"bindable,omitempty" yaml:"bindable,omitempty"`
	PlanUpdatable bool   `json:"plan_updateable,omitempty" yaml:"plan_updateable,omitempty"`

	Metadata             json.RawMessage `json:"metadata,omitempty" yaml:"-"`
	Schemas              json.RawMessage `json:"schemas,omitempty" yaml:"-"`
	unmarshalledMetadata map[string]interface{}

	ServiceOfferingID string `json:"service_offering_id,omitempty" yaml:"service_offering_id,omitempty"`
}

func (o *ServicePlan) GetMetadata() (map[string]interface{}, error) {
	if err := o.unmarshalMetadata(); err != nil {
		return nil, err
	}
	return o.unmarshalledMetadata, nil
}

func (o *ServicePlan) unmarshalMetadata() error {
	if o.unmarshalledMetadata != nil && len(o.unmarshalledMetadata) != 0 {
		return nil
	}
	var metadata map[string]interface{}
	if err := json.Unmarshal(o.Metadata, &metadata); err != nil {
		return err
	}
	o.unmarshalledMetadata = metadata

	return nil
}
