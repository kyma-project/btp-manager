package types

import "encoding/json"

type ServiceInstances struct {
	ServiceInstances []ServiceInstance `json:"items" yaml:"items"`
}

type ServiceInstance struct {
	Common
	ServiceID     string `json:"service_id,omitempty" yaml:"service_id,omitempty"`
	ServicePlanID string `json:"service_plan_id,omitempty" yaml:"service_plan_id,omitempty"`
	PlatformID    string `json:"platform_id,omitempty" yaml:"platform_id,omitempty"`

	Parameters json.RawMessage `json:"parameters,omitempty" yaml:"parameters,omitempty"`

	MaintenanceInfo json.RawMessage `json:"maintenance_info,omitempty" yaml:"-"`
	Context         json.RawMessage `json:"context,omitempty" yaml:"context,omitempty"`
	PreviousValues  json.RawMessage `json:"-" yaml:"-"`

	Ready  bool `json:"ready" yaml:"ready"`
	Usable bool `json:"usable" yaml:"usable"`
	Shared bool `json:"shared,omitempty" yaml:"shared,omitempty"`

	LastOperation *Operation `json:"last_operation,omitempty" yaml:"last_operation,omitempty"`
}
