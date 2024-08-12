package requests

import (
	"github.com/kyma-project/btp-manager/internal/service-manager/types"
)

func ToServiceBinding(request CreateServiceBinding, instance *types.ServiceInstance) (types.ServiceBinding, error) {
	clusterID, err := instance.ContextValueByFieldName(types.ContextClusterID)
	if err != nil {
		return types.ServiceBinding{}, err
	}
	namespace, err := instance.ContextValueByFieldName(types.ContextNamespace)
	if err != nil {
		return types.ServiceBinding{}, err
	}
	labels := map[string][]string{
		types.K8sNameLabel:   {request.Name},
		types.NamespaceLabel: {namespace},
		types.ClusterIDLabel: {clusterID},
	}
	sb := types.ServiceBinding{
		Common: types.Common{
			Name:   request.Name,
			Labels: labels,
		},
		ServiceInstanceID: request.ServiceInstanceID,
		Parameters:        request.Parameters,
	}
	return sb, nil
}
