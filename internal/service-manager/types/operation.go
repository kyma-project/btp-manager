package types

import "encoding/json"

const (
	AddLabelOperation    = "add"
	RemoveLabelOperation = "remove"
)

type LabelChange struct {
	Operation string   `json:"op"`
	Key       string   `json:"key"`
	Values    []string `json:"values"`
}

type OperationCategory string

const (
	CREATE OperationCategory = "create"
	UPDATE OperationCategory = "update"
	DELETE OperationCategory = "delete"
)

type OperationState string

const (
	PENDING    OperationState = "pending"
	SUCCEEDED  OperationState = "succeeded"
	INPROGRESS OperationState = "in progress"
	FAILED     OperationState = "failed"
)

const ResourceOperationsURL = "/operations"

type Operation struct {
	Common
	Type         OperationCategory `json:"type,omitempty" yaml:"type,omitempty"`
	State        OperationState    `json:"state,omitempty" yaml:"state,omitempty"`
	ResourceID   string            `json:"resource_id,omitempty" yaml:"resource_id,omitempty"`
	ResourceType string            `json:"resource_type,omitempty" yaml:"resource_type,omitempty"`
	Errors       json.RawMessage   `json:"errors,omitempty" yaml:"errors,omitempty"`
}
