package controllers

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

var expectedPolicyNames = []string{
	"kyma-project.io--btp-operator-allow-to-apiserver",
	"kyma-project.io--btp-operator-to-dns",
	"kyma-project.io--allow-btp-operator-metrics",
	"kyma-project.io--btp-operator-allow-to-webhook",
}

var _ = Describe("BTP Operator Network Policies", func() {
	Context("When testing network policies path functions", func() {
		It("Should return correct network policies path", func() {
			reconciler := &BtpOperatorReconciler{}
			path := reconciler.getNetworkPoliciesPath()
			expected := ManagerResourcesPath + string(os.PathSeparator) + "network-policies"
			Expect(path).To(Equal(expected))
		})
	})

	Context("When testing loadNetworkPolicies function", func() {
		It("Should load network policies from manager-resources directory", func() {
			reconciler := &BtpOperatorReconciler{
				manifestHandler: &manifest.Handler{Scheme: k8sManager.GetScheme()},
			}
			policies, err := reconciler.loadNetworkPolicies()
			Expect(err).NotTo(HaveOccurred())
			expectedPolicyCount := 4
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
			btpOperator.Spec.NetworkPoliciesEnabled = true
			err := reconciler.reconcileNetworkPolicies(ctx, btpOperator)
			Expect(err).NotTo(HaveOccurred())
			for _, name := range expectedPolicyNames {
				policy := &unstructured.Unstructured{}
				policy.SetAPIVersion("networking.k8s.io/v1")
				policy.SetKind("NetworkPolicy")
				err := k8sClient.Get(ctx, client.ObjectKey{Name: name, Namespace: "kyma-system"}, policy)
				Expect(err).NotTo(HaveOccurred(), "NetworkPolicy %s should exist", name)
			}
		})

		It("Should cleanup network policies when NetworkPoliciesEnabled is false", func() {
			btpOperator.Spec.NetworkPoliciesEnabled = false
			err := reconciler.reconcileNetworkPolicies(ctx, btpOperator)
			Expect(err).NotTo(HaveOccurred())
			for _, name := range expectedPolicyNames {
				policy := &unstructured.Unstructured{}
				policy.SetAPIVersion("networking.k8s.io/v1")
				policy.SetKind("NetworkPolicy")
				getErr := k8sClient.Get(ctx, client.ObjectKey{Name: name, Namespace: "kyma-system"}, policy)
				Expect(getErr).To(HaveOccurred(), "NetworkPolicy %s should be deleted", name)
			}
		})
	})

	Context("When testing cleanupNetworkPolicies", func() {
		It("Should not fail when no managed network policies exist", func() {
			reconciler := &BtpOperatorReconciler{
				Client: k8sClient,
			}
			err := reconciler.cleanupNetworkPolicies(context.Background())
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should cleanup existing network policies with managed-by btp-manager label", func() {
			reconciler := &BtpOperatorReconciler{
				Client: k8sClient,
			}
			testPolicy := createMockNetworkPolicy("test-cleanup-policy")
			labels := map[string]string{
				managedByLabelKey: operatorName,
			}
			testPolicy.SetLabels(labels)
			Expect(k8sClient.Create(context.Background(), testPolicy)).To(Succeed())
			err := reconciler.cleanupNetworkPolicies(context.Background())
			Expect(err).NotTo(HaveOccurred())
			getErr := k8sClient.Get(context.Background(), client.ObjectKey{Name: "test-cleanup-policy", Namespace: "kyma-system"}, testPolicy)
			Expect(getErr).To(HaveOccurred())
		})

		It("Should not cleanup network policies without managed-by btp-manager label", func() {
			reconciler := &BtpOperatorReconciler{
				Client: k8sClient,
			}
			testPolicy := createMockNetworkPolicy("test-unmanaged-policy")
			Expect(k8sClient.Create(context.Background(), testPolicy)).To(Succeed())
			err := reconciler.cleanupNetworkPolicies(context.Background())
			Expect(err).NotTo(HaveOccurred())
			getErr := k8sClient.Get(context.Background(), client.ObjectKey{Name: "test-unmanaged-policy", Namespace: "kyma-system"}, testPolicy)
			Expect(getErr).NotTo(HaveOccurred())
		})
	})
})
