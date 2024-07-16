package requests

type CreateServiceBinding struct {
	Name              string `json:"name"`
	Namespace         string `json:"namespace"`
	ServiceInstanceId string `json:"serviceInstanceId"`
	Parameters        string `json:"parameters"`
}
