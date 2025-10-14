package controllers

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
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

	Context("When testing network policies integration in reconcileResources", func() {
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

		It("Should load network policies correctly when enabled", func() {
			btpOperator.Spec.NetworkPoliciesEnabled = true
			policies, err := reconciler.loadNetworkPolicies()
			Expect(err).NotTo(HaveOccurred())
			Expect(len(policies)).To(BeNumerically(">", 0))
		})

		It("Should be able to cleanup network policies when disabled", func() {
			btpOperator.Spec.NetworkPoliciesEnabled = false
			err := reconciler.cleanupNetworkPolicies(ctx)
			Expect(err).NotTo(HaveOccurred())
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

	Context("When testing full reconcile loop with network policies", func() {
		var (
			reconciler  *BtpOperatorReconciler
			btpOperator *v1alpha1.BtpOperator
			secret      *corev1.Secret
			ctx         context.Context
		)

		BeforeEach(func() {
			ctx = context.Background()
			reconciler = &BtpOperatorReconciler{
				Client:          k8sClient,
				Scheme:          k8sClient.Scheme(),
				manifestHandler: &manifest.Handler{Scheme: k8sManager.GetScheme()},
			}
			
			btpOperator = &v1alpha1.BtpOperator{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "btpoperator",
					Namespace: "kyma-system",
				},
				Spec: v1alpha1.BtpOperatorSpec{
					NetworkPoliciesEnabled: true,
				},
			}
			
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sap-btp-manager",
					Namespace: "kyma-system",
				},
				Data: map[string][]byte{
					"clientid":     []byte("test-client-id"),
					"clientsecret": []byte("test-client-secret"),
					"sm_url":       []byte("https://test-sm-url"),
					"tokenurl":     []byte("https://test-token-url"),
					"cluster_id":   []byte("test-cluster-id"),
				},
			}
		})

		It("Should successfully reconcile when network policies are enabled", func() {
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			Expect(k8sClient.Create(ctx, btpOperator)).To(Succeed())

			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: client.ObjectKeyFromObject(btpOperator),
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			var updatedBtpOperator v1alpha1.BtpOperator
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(btpOperator), &updatedBtpOperator)).To(Succeed())
			
			Eventually(func() v1alpha1.State {
				k8sClient.Get(ctx, client.ObjectKeyFromObject(btpOperator), &updatedBtpOperator)
				return updatedBtpOperator.Status.State
			}, "30s", "1s").Should(Or(Equal(v1alpha1.StateReady), Equal(v1alpha1.StateProcessing)))
		})

		It("Should handle network policies being disabled during reconciliation", func() {
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			Expect(k8sClient.Create(ctx, btpOperator)).To(Succeed())

			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: client.ObjectKeyFromObject(btpOperator),
			})
			Expect(err).NotTo(HaveOccurred())

			var updatedBtpOperator v1alpha1.BtpOperator
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(btpOperator), &updatedBtpOperator)).To(Succeed())
			updatedBtpOperator.Spec.NetworkPoliciesEnabled = false
			Expect(k8sClient.Update(ctx, &updatedBtpOperator)).To(Succeed())

			_, err = reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: client.ObjectKeyFromObject(btpOperator),
			})
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				policies := &networkingv1.NetworkPolicyList{}
				listOpts := []client.ListOption{
					client.InNamespace("kyma-system"),
					client.MatchingLabels{managedByLabelKey: operatorName},
				}
				err := k8sClient.List(ctx, policies, listOpts...)
				if err != nil {
					return false
				}
				return len(policies.Items) == 0
			}, "30s", "1s").Should(BeTrue(), "All managed network policies should be cleaned up")
		})

		It("Should transition to error state when network policies integration fails", func() {
			By("Creating BtpOperator with network policies enabled but missing manifests")
			
			btpOperatorWithBadConfig := &v1alpha1.BtpOperator{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "btpoperator-bad",
					Namespace: "kyma-system",
				},
				Spec: v1alpha1.BtpOperatorSpec{
					NetworkPoliciesEnabled: true,
				},
			}
			
			By("Simulating a scenario where network policies directory doesn't exist")
			originalPath := ManagerResourcesPath
			ManagerResourcesPath = "/nonexistent/path"
			defer func() { ManagerResourcesPath = originalPath }()

			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			Expect(k8sClient.Create(ctx, btpOperatorWithBadConfig)).To(Succeed())

			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: client.ObjectKeyFromObject(btpOperatorWithBadConfig),
			})
			
			By("The reconciliation should continue despite network policies error")
			Expect(err).To(HaveOccurred())
		})

		AfterEach(func() {
			k8sClient.Delete(ctx, btpOperator)
			k8sClient.Delete(ctx, secret)
			
			policies := &networkingv1.NetworkPolicyList{}
			k8sClient.List(ctx, policies, client.InNamespace("kyma-system"))
			for _, policy := range policies.Items {
				k8sClient.Delete(ctx, &policy)
			}
		})
	})
})
