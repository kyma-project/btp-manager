package requests

import (
	"encoding/json"

	"github.com/kyma-project/btp-manager/internal/service-manager/types"
)

type CreateServiceBinding struct {
	Name              string          `json:"name"`
	ServiceInstanceId string          `json:"serviceInstanceId"`
	Parameters        json.RawMessage `json:"parameters"`
}

type CreateServiceInstance struct {
	Name       string              `json:"name"`
	PlanID     string              `json:"planID"`
	Namespace  string              `json:"namespace"`
	ClusterID  string              `json:"clusterID"`
	Labels     map[string][]string `json:"labels"`
	Parameters json.RawMessage     `json:"parameters"`
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
		ServicePlanID: csi.PlanID,
		Parameters:    csi.Parameters,
	}
}
