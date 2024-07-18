package requests

import "encoding/json"

type CreateServiceBinding struct {
	Name              string          `json:"name"`
	ServiceInstanceId string          `json:"serviceInstanceId"`
	Parameters        json.RawMessage `json:"parameters"`
}
