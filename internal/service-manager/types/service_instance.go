package types

import (
	"encoding/json"
	"fmt"
)

const (
	ServiceInstanceClusterID    = "clusterid"
	ServiceInstanceSubaccountID = "subaccount_id"
	ServiceInstanceNamespace    = "namespace"
)

type ServiceInstances struct {
	Items []ServiceInstance `json:"items" yaml:"items"`
}

type ServiceInstance struct {
	Common
	ServiceID       string `json:"service_id,omitempty" yaml:"service_id,omitempty"`
	ServicePlanID   string `json:"service_plan_id,omitempty" yaml:"service_plan_id,omitempty"`
	ServicePlanName string `json:"-" yaml:"-"`
	PlatformID      string `json:"platform_id,omitempty" yaml:"platform_id,omitempty"`

	Parameters json.RawMessage `json:"parameters,omitempty" yaml:"parameters,omitempty"`

	MaintenanceInfo     json.RawMessage `json:"maintenance_info,omitempty" yaml:"-"`
	Context             json.RawMessage `json:"context,omitempty" yaml:"context,omitempty"`
	PreviousValues      json.RawMessage `json:"-" yaml:"-"`
	unmarshalledContext map[string]interface{}

	Ready  bool `json:"ready" yaml:"ready"`
	Usable bool `json:"usable" yaml:"usable"`
	Shared bool `json:"shared,omitempty" yaml:"shared,omitempty"`

	LastOperation *Operation `json:"last_operation,omitempty" yaml:"last_operation,omitempty"`
}

func (i *ServiceInstance) ContextValueByFieldName(fieldName string) (string, error) {
	if i.Context == nil || len(i.Context) == 0 {
		return "", nil
	}
	if err := i.unmarshalContext(); err != nil {
		return "", err
	}
	val, ok := i.unmarshalledContext[fieldName]
	if !ok {
		return "not found", nil
	}

	return fmt.Sprint(val), nil

}

func (i *ServiceInstance) unmarshalContext() error {
	if i.unmarshalledContext != nil && len(i.unmarshalledContext) != 0 {
		return nil
	}
	var context map[string]interface{}
	if err := json.Unmarshal(i.Context, &context); err != nil {
		return err
	}
	i.unmarshalledContext = context

	return nil
}

type ServiceInstanceUpdateRequest struct {
	ID            *string          `json:"id,omitempty" yaml:"id,omitempty"`
	Name          *string          `json:"name,omitempty" yaml:"name,omitempty"`
	ServicePlanID *string          `json:"service_plan_id,omitempty" yaml:"service_plan_id,omitempty"`
	Shared        *bool            `json:"shared,omitempty" yaml:"shared,omitempty"`
	Parameters    *json.RawMessage `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Labels        []LabelChange    `json:"labels,omitempty" yaml:"labels,omitempty"`
}
