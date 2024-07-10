package types

import "encoding/json"

// ServiceBinding defines the data of a service instance.
type ServiceBinding struct {
	Common
	Labels         Labels `json:"labels,omitempty" yaml:"labels,omitempty"`
	PagingSequence int64  `json:"-" yaml:"-"`

	Credentials json.RawMessage `json:"credentials,omitempty" yaml:"credentials,omitempty"`

	ServiceInstanceID   string `json:"service_instance_id" yaml:"service_instance_id,omitempty"`
	ServiceInstanceName string `json:"service_instance_name,omitempty" yaml:"service_instance_name,omitempty"`

	SyslogDrainURL  string          `json:"syslog_drain_url,omitempty" yaml:"syslog_drain_url,omitempty"`
	RouteServiceURL string          `json:"route_service_url,omitempty"`
	VolumeMounts    json.RawMessage `json:"-" yaml:"-"`
	Endpoints       json.RawMessage `json:"-" yaml:"-"`
	Context         json.RawMessage `json:"context,omitempty" yaml:"context,omitempty"`
	Parameters      json.RawMessage `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	BindResource    json.RawMessage `json:"-" yaml:"-"`

	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// ServiceBindings wraps an array of service bindings
type ServiceBindings struct {
	ServiceBindings []ServiceBinding `json:"items" yaml:"items"`
	Vertical        bool             `json:"-" yaml:"-"`
}
