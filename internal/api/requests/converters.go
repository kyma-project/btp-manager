package requests

import (
	"github.com/kyma-project/btp-manager/internal/service-manager/types"
)

func ToServiceBinding(request CreateServiceBinding, instance *types.ServiceInstance) (types.ServiceBinding, error) {
	clusterID, err := instance.ContextValueByFieldName(types.ServiceInstanceClusterID)
	if err != nil {
		return types.ServiceBinding{}, err
	}
	namespace, err := instance.ContextValueByFieldName(types.ServiceInstanceNamespace)
	if err != nil {
		return types.ServiceBinding{}, err
	}
	sb := types.ServiceBinding{
		Common: types.Common{
			Name:   request.Name,
			Labels: map[string][]string{},
		},
		ServiceInstanceID: request.ServiceInstanceId,
		Parameters:        request.Parameters,
	}
	sb.Labels["_clusterid"] = append(sb.Labels["_clusterid"], clusterID)
	sb.Labels["_namespace"] = append(sb.Labels["_namespace"], namespace)
	sb.Labels["_k8sname"] = append(sb.Labels["_k8sname"], request.Name)
	return sb, nil
}
