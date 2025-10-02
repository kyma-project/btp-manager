package controllers

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/internal/manifest"
)

func createMockNetworkPolicy(name string) *unstructured.Unstructured {
	policy := &unstructured.Unstructured{}
	policy.SetAPIVersion("networking.k8s.io/v1")
	policy.SetKind("NetworkPolicy")
	policy.SetName(name)
	policy.SetNamespace("kyma-system")
	return policy
}

var _ = Describe("BTP Operator Network Policies", func() {
	Context("When testing network policies path functions", func() {
		It("Should return correct network policies path", func() {
			// Given
			reconciler := &BtpOperatorReconciler{}

			// When
			path := reconciler.getNetworkPoliciesPath()

			// Then
			expected := ManagerResourcesPath + string(os.PathSeparator) + "network-policies"
			Expect(path).To(Equal(expected))
		})
	})

	Context("When testing loadNetworkPolicies function", func() {
		It("Should load network policies from manager-resources directory", func() {
			// Given
			ManagerResourcesPath = "../manager-resources"
			reconciler := &BtpOperatorReconciler{
				manifestHandler: &manifest.Handler{Scheme: k8sManager.GetScheme()},
			}

			// When
			policies, err := reconciler.loadNetworkPolicies()

			// Then
			Expect(err).NotTo(HaveOccurred())
			expectedPolicyCount := 2
			Expect(policies).To(HaveLen(expectedPolicyCount))
			for _, policy := range policies {
				Expect(policy.GetKind()).To(Equal("NetworkPolicy"))
				Expect(policy.GetAPIVersion()).To(Equal("networking.k8s.io/v1"))
			}
		})
	})

	Context("When testing reconcileNetworkPolicies function", func() {
		var (
			reconciler  *BtpOperatorReconciler
			btpOperator *v1alpha1.BtpOperator
			ctx         context.Context
		)

		BeforeEach(func() {
			ctx = context.Background()
			reconciler = &BtpOperatorReconciler{
				Client:          k8sClient,
				Scheme:          k8sClient.Scheme(),
				manifestHandler: &manifest.Handler{Scheme: k8sManager.GetScheme()},
			}

			btpOperator = &v1alpha1.BtpOperator{}
			btpOperator.Name = "test-btpoperator"
			btpOperator.Namespace = "kyma-system"
		})

		It("Should apply network policies when NetworkPoliciesEnabled is true", func() {
			// Given
			btpOperator.Spec.NetworkPoliciesEnabled = true

			// When
			err := reconciler.reconcileNetworkPolicies(ctx, btpOperator)

			// Then
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should cleanup network policies when NetworkPoliciesEnabled is false", func() {
			// Given
			btpOperator.Spec.NetworkPoliciesEnabled = false

			// When
			err := reconciler.reconcileNetworkPolicies(ctx, btpOperator)

			// Then
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When testing cleanupNetworkPoliciesByNames", func() {
		It("Should handle empty policy names list", func() {
			// Given
			reconciler := &BtpOperatorReconciler{
				Client: k8sClient,
			}

			// When
			err := reconciler.cleanupNetworkPoliciesByNames(context.Background(), []string{})

			// Then
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should not fail when trying to delete non-existent network policies", func() {
			// Given
			reconciler := &BtpOperatorReconciler{
				Client: k8sClient,
			}

			// When
			err := reconciler.cleanupNetworkPoliciesByNames(context.Background(), []string{"non-existent-policy"})

			// Then
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should cleanup existing network policies", func() {
			// Given
			reconciler := &BtpOperatorReconciler{
				Client: k8sClient,
			}

			// Create a test network policy
			testPolicy := createMockNetworkPolicy("test-cleanup-policy")
			Expect(k8sClient.Create(context.Background(), testPolicy)).To(Succeed())

			// When
			err := reconciler.cleanupNetworkPoliciesByNames(context.Background(), []string{"test-cleanup-policy"})

			// Then
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
