package types

import (
	"encoding/json"
	"fmt"
)

// Refs
// https://github.com/SAP/sap-btp-service-operator/blob/main/client/sm/types/service_offering.go
// https://github.com/Peripli/service-manager/blob/master/pkg/types/service_offering.go

const (
	ServiceOfferingDisplayName      = "displayName"
	ServiceOfferingDocumentationUrl = "documentationUrl"
	ServiceOfferingImageUrl         = "imageUrl"
	ServiceOfferingLongDescription  = "longDescription"
	ServiceOfferingSupportURL       = "supportUrl"
)

type ServiceOfferingDetails struct {
	ServiceOffering
	ServicePlans `json:"plans" yaml:"plans"`
}

type ServiceOfferings struct {
	ServiceOfferings []ServiceOffering `json:"items" yaml:"items"`
}

type ServiceOffering struct {
	Common
	Bindable             bool `json:"bindable,omitempty" yaml:"bindable,omitempty"`
	InstancesRetrievable bool `json:"instances_retrievable,omitempty" yaml:"instances_retrievable,omitempty"`
	BindingsRetrievable  bool `json:"bindings_retrievable,omitempty" yaml:"bindings_retrievable,omitempty"`
	PlanUpdatable        bool `json:"plan_updateable,omitempty" yaml:"plan_updateable,omitempty"`
	AllowContextUpdates  bool `json:"allow_context_updates,omitempty" yaml:"allow_context_updates,omitempty"`

	Tags                 json.RawMessage `json:"tags,omitempty" yaml:"-"`
	Metadata             json.RawMessage `json:"metadata,omitempty" yaml:"-"`
	unmarshalledMetadata map[string]interface{}

	BrokerID    string `json:"broker_id,omitempty" yaml:"broker_id,omitempty"`
	CatalogID   string `json:"catalog_id,omitempty" yaml:"catalog_id,omitempty"`
	CatalogName string `json:"catalog_name,omitempty" yaml:"catalog_name,omitempty"`
}

func (o *ServiceOffering) MetadataValueByFieldName(fieldName string) (string, error) {
	if err := o.unmarshalMetadata(); err != nil {
		return "", err
	}
	val, ok := o.unmarshalledMetadata[fieldName]
	if !ok {
		return "not found", nil
	}

	return fmt.Sprint(val), nil
}

func (o *ServiceOffering) unmarshalMetadata() error {
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
