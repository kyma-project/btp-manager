package networkpolicy_test

import (
	"context"

	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/k8s/networkpolicy"
	"github.com/kyma-project/btp-manager/internal/manifest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const testManagerResourcesPath = "./testdata"

var _ = Describe("Network Policy Manager", func() {
	var (
		mgr *networkpolicy.Manager
		ctx context.Context
	)

	BeforeEach(func() {
		config.ManagerResourcesPath = testManagerResourcesPath

		fakeClient = newFakeClient()
		mgr = networkpolicy.NewManager(fakeClient, &manifest.Handler{Scheme: scheme})
		ctx = context.Background()
	})

	Describe("LoadNetworkPolicies", func() {
		It("should return unstructured network policies from the manifests directory", func() {
			policies, err := mgr.LoadNetworkPolicies()

			Expect(err).NotTo(HaveOccurred())
			Expect(policies).To(HaveLen(2))

			names := extractNames(policies)
			Expect(names).To(ContainElements(policyName1, policyName2))
		})

		It("should return an error when the manifests directory does not exist", func() {
			config.ManagerResourcesPath = "./non-existent"

			_, err := mgr.LoadNetworkPolicies()

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("CleanupNetworkPolicies", func() {
		It("should delete all managed network policies from the cluster", func() {
			policy1 := managedNetworkPolicy(policyName1)
			policy2 := managedNetworkPolicy(policyName2)
			fakeClient = newFakeClient(policy1, policy2)
			mgr = networkpolicy.NewManager(fakeClient, &manifest.Handler{Scheme: scheme})

			err := mgr.CleanupNetworkPolicies(ctx)

			Expect(err).NotTo(HaveOccurred())

			remaining := &networkingv1.NetworkPolicyList{}
			Expect(fakeClient.List(ctx, remaining, client.InNamespace(kymaNamespace))).To(Succeed())
			Expect(remaining.Items).To(BeEmpty())
		})

		It("should succeed when no managed network policies exist", func() {
			err := mgr.CleanupNetworkPolicies(ctx)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should not delete unmanaged network policies", func() {
			managedPolicy := managedNetworkPolicy(policyName1)
			unmanagedPolicy := unmanagedNetworkPolicy("user-defined-policy")
			fakeClient = newFakeClient(managedPolicy, unmanagedPolicy)
			mgr = networkpolicy.NewManager(fakeClient, &manifest.Handler{Scheme: scheme})

			err := mgr.CleanupNetworkPolicies(ctx)

			Expect(err).NotTo(HaveOccurred())

			remaining := &networkingv1.NetworkPolicyList{}
			Expect(fakeClient.List(ctx, remaining, client.InNamespace(kymaNamespace))).To(Succeed())
			Expect(remaining.Items).To(HaveLen(1))
			Expect(remaining.Items[0].Name).To(Equal(unmanagedPolicy.Name))
		})
	})

	Describe("DeleteOldWebhookNetworkPolicy", func() {
		const oldWebhookPolicyName = "kyma-project.io--btp-operator-allow-to-webhook"

		It("should delete the old webhook network policy when it exists", func() {
			oldPolicy := &networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      oldWebhookPolicyName,
					Namespace: kymaNamespace,
				},
			}
			fakeClient = newFakeClient(oldPolicy)
			mgr = networkpolicy.NewManager(fakeClient, &manifest.Handler{Scheme: scheme})

			err := mgr.DeleteOldWebhookNetworkPolicy(ctx)

			Expect(err).NotTo(HaveOccurred())

			remaining := &networkingv1.NetworkPolicy{}
			getErr := fakeClient.Get(ctx, client.ObjectKeyFromObject(oldPolicy), remaining)
			Expect(getErr).To(HaveOccurred())
		})

		It("should succeed without an error when the old webhook policy does not exist", func() {
			err := mgr.DeleteOldWebhookNetworkPolicy(ctx)

			Expect(err).NotTo(HaveOccurred())
		})
	})
})

func extractNames(us []*unstructured.Unstructured) []string {
	names := make([]string, 0, len(us))
	for _, u := range us {
		names = append(names, u.GetName())
	}
	return names
}
