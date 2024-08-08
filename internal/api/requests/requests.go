package requests

import (
	"encoding/json"

	"github.com/kyma-project/btp-manager/internal/service-manager/types"
)

type CreateServiceBinding struct {
	Name              string          `json:"name"`
	ServiceInstanceID string          `json:"service_instance_id"`
	Parameters        json.RawMessage `json:"parameters"`
	SecretName        string          `json:"secret_name"`
	SecretNamespace   string          `json:"secret_namespace"`
}

type CreateServiceInstance struct {
	Name          string              `json:"name"`
	ServicePlanID string              `json:"service_plan_id"`
	Namespace     string              `json:"namespace"`
	ClusterID     string              `json:"cluster_id"`
	Labels        map[string][]string `json:"labels"`
	Parameters    json.RawMessage     `json:"parameters"`
}

func (csi *CreateServiceInstance) ConvertToServiceInstance() *types.ServiceInstance {
	labels := map[string][]string{
		types.NamespaceLabel: {csi.Namespace},
		types.K8sNameLabel:   {csi.Name},
		types.ClusterIDLabel: {csi.ClusterID},
	}
	for k, v := range csi.Labels {
		labels[k] = v
	}
	return &types.ServiceInstance{
		Common: types.Common{
			Name:   csi.Name,
			Labels: labels,
		},
		ServicePlanID: csi.ServicePlanID,
		Parameters:    csi.Parameters,
	}
}
