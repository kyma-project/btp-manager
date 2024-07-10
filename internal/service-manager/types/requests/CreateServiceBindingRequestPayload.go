package requests

import (
	"encoding/json"

	"github.com/kyma-project/btp-manager/internal/service-manager/types"
)

type CreateServiceBindingRequestPayload struct {
	Name              string          `json:"name"`
	ServiceInstanceID string          `json:"service_instance_id" yaml:"service_instance_id,omitempty"`
	Parameters        json.RawMessage `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Labels            types.Labels    `json:"labels,omitempty" yaml:"labels,omitempty"`
	BindResource      json.RawMessage `json:"bind_resource" yaml:"bind_resource"`
}
