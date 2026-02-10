package managerresource

import (
	"fmt"
	"os"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/controllers/config"

	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NetworkPolicies struct{}

func (n *NetworkPolicies) Name() string {
	return "network policies"
}

func (n *NetworkPolicies) Enabled(cr *v1alpha1.BtpOperator) bool {
	return !cr.IsNetworkPoliciesDisabled()
}

func (n *NetworkPolicies) ManifestsPath() string {
	return fmt.Sprintf("%s%cnetwork-policies", config.ManagerResourcesPath, os.PathSeparator)
}

func (n *NetworkPolicies) Object() client.Object {
	return &networkingv1.NetworkPolicy{}
}
