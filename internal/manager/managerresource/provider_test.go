package managerresource

import (
	"github.com/kyma-project/btp-manager/api/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Provider", func() {

	It("returns NetworkPolicies enabled when CR does not disable them", func() {
		cr := &v1alpha1.BtpOperator{}
		cr.Annotations = map[string]string{
			v1alpha1.DisableNetworkPoliciesAnnotation: "false",
		}

		provider := NewProvider(cr)
		resources := provider.Resources()

		Expect(resources).To(HaveLen(1))

		np, ok := resources[0].(*NetworkPolicies)
		Expect(ok).To(BeTrue())
		Expect(np.enabled).To(BeTrue())
	})

	It("returns NetworkPolicies disabled when CR disables them", func() {
		cr := &v1alpha1.BtpOperator{}
		cr.Annotations = map[string]string{
			v1alpha1.DisableNetworkPoliciesAnnotation: "true",
		}

		provider := NewProvider(cr)
		resources := provider.Resources()

		Expect(resources).To(HaveLen(1))

		np, ok := resources[0].(*NetworkPolicies)
		Expect(ok).To(BeTrue())
		Expect(np.enabled).To(BeFalse())
	})
})
