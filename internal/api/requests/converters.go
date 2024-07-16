package requests

import (
	"github.com/kyma-project/btp-manager/internal/service-manager/types/requests"
)

func CreateServiceBindingVM(request CreateServiceBinding) requests.CreateServiceBindingRequestPayload {
	payload := requests.CreateServiceBindingRequestPayload{
		Name:              request.Name,
		ServiceInstanceID: request.ServiceInstanceId,
		Parameters:        []byte(request.Parameters),
		Labels:            map[string][]string{},
	}
	payload.Labels["_clusterid"] = append(payload.Labels["_clusterid"], payload.Name)
	payload.Labels["_namespace"] = append(payload.Labels["_namespace"], request.Namespace)
	payload.Labels["_k8sname"] = append(payload.Labels["_k8sname"], request.Name)
	return payload
}
