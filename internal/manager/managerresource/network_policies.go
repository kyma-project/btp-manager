package managerresource

import (
	"fmt"
	"os"

	"github.com/kyma-project/btp-manager/controllers/config"

	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NetworkPolicies struct {
	enabled bool
}

func NewNetworkPolicies(enabled bool) *NetworkPolicies {
	return &NetworkPolicies{enabled: enabled}
}

func (n *NetworkPolicies) Name() string {
	return "network policies"
}

func (n *NetworkPolicies) Enabled() bool {
	return n.enabled
}

func (n *NetworkPolicies) ManifestsPath() string {
	return fmt.Sprintf("%s%cnetwork-policies", config.ManagerResourcesPath, os.PathSeparator)
}

func (n *NetworkPolicies) Object() client.Object {
	return &networkingv1.NetworkPolicy{}
}
