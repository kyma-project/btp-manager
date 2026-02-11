package managerresource

import (
	"github.com/kyma-project/btp-manager/api/v1alpha1"
)

type Provider struct {
	cr *v1alpha1.BtpOperator
}

func NewProvider(cr *v1alpha1.BtpOperator) *Provider {
	return &Provider{cr: cr}
}

func (p *Provider) Resources() []Resource {
	return []Resource{
		NewNetworkPolicies(!p.cr.IsNetworkPoliciesDisabled()),
	}
}
