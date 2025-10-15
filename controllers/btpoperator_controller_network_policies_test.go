package controllers

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/internal/conditions"
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

		It("Should load and prepare network policies when NetworkPoliciesEnabled is true", func() {
			btpOperator.Spec.NetworkPoliciesEnabled = true
			policies, err := reconciler.loadNetworkPolicies()
			Expect(err).NotTo(HaveOccurred())
			Expect(policies).To(HaveLen(4))
			for _, policy := range policies {
				Expect(policy.GetKind()).To(Equal("NetworkPolicy"))
				Expect(policy.GetAPIVersion()).To(Equal("networking.k8s.io/v1"))
				policy.SetNamespace("kyma-system")
				labels := policy.GetLabels()
				if labels == nil {
					labels = make(map[string]string)
				}
				labels[managedByLabelKey] = operatorName
				policy.SetLabels(labels)
				Expect(policy.GetNamespace()).To(Equal("kyma-system"))
				Expect(policy.GetLabels()).To(HaveKeyWithValue(managedByLabelKey, operatorName))
			}
		})

		It("Should call cleanupNetworkPolicies when NetworkPoliciesEnabled is false", func() {
			for _, name := range expectedPolicyNames {
				policy := createMockNetworkPolicy(name)
				labels := map[string]string{
					managedByLabelKey: operatorName,
				}
				policy.SetLabels(labels)
				Expect(k8sClient.Create(ctx, policy)).To(Succeed())
			}
			err := reconciler.cleanupNetworkPolicies(ctx)
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

	Context("When testing full reconcile loop with network policies", func() {
		var (
			cr     *v1alpha1.BtpOperator
			ctx    context.Context
			secret *corev1.Secret
		)

		BeforeEach(func() {
			GinkgoWriter.Println("--- PROCESS:", GinkgoParallelProcess(), "---")
			ctx = context.Background()
			cr = createDefaultBtpOperator()
			cr.SetLabels(map[string]string{forceDeleteLabelKey: "true"})
		})

		AfterEach(func() {
			if cr != nil {
				if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: btpOperatorName}, cr); err == nil {
					Expect(k8sClient.Delete(ctx, cr)).To(Succeed())
					Eventually(updateCh).Should(Receive(matchDeleted()))
					Expect(isCrNotFound()).To(BeTrue())
				}
			}
			if secret != nil {
				if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: SecretName}, secret); err == nil {
					Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
				}
			}
		})

		createCRWithNetworkPoliciesAndWaitForReady := func(networkPoliciesEnabled bool) {
			cr.Spec.NetworkPoliciesEnabled = networkPoliciesEnabled
			Eventually(func() error { return k8sClient.Create(ctx, cr) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

			var err error
			secret, err = createCorrectSecretFromYaml()
			Expect(err).To(BeNil())
			Expect(k8sClient.Patch(ctx, secret, client.Apply, client.ForceOwnership, client.FieldOwner(operatorName))).To(Succeed())

			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateProcessing, metav1.ConditionFalse, conditions.Initialized)))
			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
		}

		createCRWithNetworkPoliciesEnabledAndWaitForReady := func() {
			createCRWithNetworkPoliciesAndWaitForReady(true)
		}

		createCRWithNetworkPoliciesDisabledAndWaitForReady := func() {
			createCRWithNetworkPoliciesAndWaitForReady(false)
		}

		updateNetworkPoliciesSettingAndWait := func(enabled bool) {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: btpOperatorName}, cr)).To(Succeed())
			cr.Spec.NetworkPoliciesEnabled = enabled
			Expect(k8sClient.Update(ctx, cr)).To(Succeed())
			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
		}

		enableNetworkPoliciesAndWait := func() {
			updateNetworkPoliciesSettingAndWait(true)
		}

		disableNetworkPoliciesAndWait := func() {
			updateNetworkPoliciesSettingAndWait(false)
		}

		verifyNetworkPoliciesExist := func() {
			Eventually(func() bool {
				for _, policyName := range expectedPolicyNames {
					policy := &unstructured.Unstructured{}
					policy.SetAPIVersion("networking.k8s.io/v1")
					policy.SetKind("NetworkPolicy")
					err := k8sClient.Get(ctx, client.ObjectKey{Name: policyName, Namespace: "kyma-system"}, policy)
					if err != nil {
						return false
					}
					labels := policy.GetLabels()
					if labels == nil || labels[managedByLabelKey] != operatorName {
						return false
					}
				}
				return true
			}).Should(BeTrue(), "All network policies should exist")
		}

		verifyNetworkPoliciesDeleted := func() {
			Eventually(func() bool {
				for _, policyName := range expectedPolicyNames {
					policy := &unstructured.Unstructured{}
					policy.SetAPIVersion("networking.k8s.io/v1")
					policy.SetKind("NetworkPolicy")
					err := k8sClient.Get(ctx, client.ObjectKey{Name: policyName, Namespace: "kyma-system"}, policy)
					if err == nil {
						return false
					}
				}
				return true
			}).Should(BeTrue(), "All network policies should be deleted")
		}

		Context("When NetworkPoliciesEnabled is set to true", func() {
			It("Should create network policies during provisioning", func() {
				createCRWithNetworkPoliciesEnabledAndWaitForReady()
				verifyNetworkPoliciesExist()
			})

			It("Should handle network policies when updating from disabled to enabled", func() {
				createCRWithNetworkPoliciesDisabledAndWaitForReady()
				verifyNetworkPoliciesDeleted()

				enableNetworkPoliciesAndWait()
				verifyNetworkPoliciesExist()
			})
		})

		Context("When NetworkPoliciesEnabled is set to false", func() {
			It("Should not create network policies during provisioning", func() {
				createCRWithNetworkPoliciesDisabledAndWaitForReady()
				verifyNetworkPoliciesDeleted()
			})

			It("Should clean up existing network policies when updating from enabled to disabled", func() {
				createCRWithNetworkPoliciesEnabledAndWaitForReady()
				verifyNetworkPoliciesExist()

				disableNetworkPoliciesAndWait()
				verifyNetworkPoliciesDeleted()
			})
		})
	})
})
