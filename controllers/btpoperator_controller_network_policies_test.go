package controllers

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/conditions"
	"github.com/kyma-project/btp-manager/internal/k8s/networkpolicy"
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

func getNetworkPolicy(ctx context.Context, name, namespace string) (*unstructured.Unstructured, error) {
	policy := &unstructured.Unstructured{}
	policy.SetAPIVersion("networking.k8s.io/v1")
	policy.SetKind("NetworkPolicy")
	err := k8sClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, policy)
	return policy, err
}

var expectedPolicyNames = []string{
	"kyma-project.io--btp-operator-allow-to-apiserver",
	"kyma-project.io--btp-operator-to-dns",
	"kyma-project.io--allow-btp-operator-metrics",
	"kyma-project.io--allow-btp-operator-webhook",
}

var _ = Describe("BTP Operator Network Policies", func() {
	Context("When testing loadNetworkPolicies function", func() {
		It("Should load network policies from manager-resources directory", func() {
			npMgr := networkpolicy.NewManager(k8sClient, &manifest.Handler{Scheme: k8sManager.GetScheme()})
			policies, err := npMgr.LoadNetworkPolicies()
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
			npMgr := networkpolicy.NewManager(k8sClient, &manifest.Handler{Scheme: k8sManager.GetScheme()})
			reconciler = &BtpOperatorReconciler{
				Client:               k8sClient,
				Scheme:               k8sClient.Scheme(),
				networkPolicyManager: npMgr,
			}
			btpOperator = &v1alpha1.BtpOperator{}
			btpOperator.Name = "test-btpoperator"
			btpOperator.Namespace = "kyma-system"
			Expect(reconciler.cleanupNetworkPolicies(ctx)).To(Succeed())
		})

		It("Should load and prepare network policies", func() {
			policies, err := reconciler.networkPolicyManager.LoadNetworkPolicies()
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

		It("Should call cleanupNetworkPolicies", func() {
			for _, name := range expectedPolicyNames {
				policy := createMockNetworkPolicy(name)
				policy.SetLabels(map[string]string{
					managedByLabelKey:         operatorName,
					kymaProjectModuleLabelKey: moduleName,
				})
				Expect(k8sClient.Create(ctx, policy)).To(Succeed())
			}
			err := reconciler.cleanupNetworkPolicies(ctx)
			Expect(err).NotTo(HaveOccurred())
			for _, name := range expectedPolicyNames {
				_, getErr := getNetworkPolicy(ctx, name, "kyma-system")
				Expect(getErr).To(HaveOccurred(), "NetworkPolicy %s should be deleted", name)
			}
		})

		It("Should check annotation correctly for network policies", func() {
			btpOperator.Annotations = nil
			Expect(btpOperator.IsNetworkPoliciesDisabled()).To(BeFalse())

			btpOperator.Annotations = map[string]string{}
			Expect(btpOperator.IsNetworkPoliciesDisabled()).To(BeFalse())

			btpOperator.Annotations = map[string]string{
				v1alpha1.DisableNetworkPoliciesAnnotation: "false",
			}
			Expect(btpOperator.IsNetworkPoliciesDisabled()).To(BeFalse())

			btpOperator.Annotations = map[string]string{
				v1alpha1.DisableNetworkPoliciesAnnotation: "true",
			}
			Expect(btpOperator.IsNetworkPoliciesDisabled()).To(BeTrue())

			btpOperator.Annotations = map[string]string{
				v1alpha1.DisableNetworkPoliciesAnnotation: "TRUE",
			}
			Expect(btpOperator.IsNetworkPoliciesDisabled()).To(BeTrue())

			btpOperator.Annotations = map[string]string{
				v1alpha1.DisableNetworkPoliciesAnnotation: "random",
			}
			Expect(btpOperator.IsNetworkPoliciesDisabled()).To(BeFalse())
		})
	})

	Context("When testing cleanupNetworkPolicies", func() {
		It("Should not fail when no managed network policies exist", func() {
			npMgr := networkpolicy.NewManager(k8sClient, &manifest.Handler{Scheme: k8sManager.GetScheme()})
			reconciler := &BtpOperatorReconciler{
				Client:               k8sClient,
				networkPolicyManager: npMgr,
			}
			err := reconciler.cleanupNetworkPolicies(context.Background())
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should cleanup existing network policies with managed-by btp-manager label", func() {
			npMgr := networkpolicy.NewManager(k8sClient, &manifest.Handler{Scheme: k8sManager.GetScheme()})
			reconciler := &BtpOperatorReconciler{
				Client:               k8sClient,
				networkPolicyManager: npMgr,
			}
			testPolicy := createMockNetworkPolicy("test-cleanup-policy")
			testPolicy.SetLabels(map[string]string{
				managedByLabelKey:         operatorName,
				kymaProjectModuleLabelKey: moduleName,
			})
			Expect(k8sClient.Create(context.Background(), testPolicy)).To(Succeed())
			err := reconciler.cleanupNetworkPolicies(context.Background())
			Expect(err).NotTo(HaveOccurred())
			_, getErr := getNetworkPolicy(context.Background(), "test-cleanup-policy", "kyma-system")
			Expect(getErr).To(HaveOccurred())
		})

		It("Should not cleanup network policies without managed-by btp-manager label", func() {
			npMgr := networkpolicy.NewManager(k8sClient, &manifest.Handler{Scheme: k8sManager.GetScheme()})
			reconciler := &BtpOperatorReconciler{
				Client:               k8sClient,
				networkPolicyManager: npMgr,
			}
			testPolicy := createMockNetworkPolicy("test-unmanaged-policy")
			Expect(k8sClient.Create(context.Background(), testPolicy)).To(Succeed())
			err := reconciler.cleanupNetworkPolicies(context.Background())
			Expect(err).NotTo(HaveOccurred())
			_, getErr := getNetworkPolicy(context.Background(), "test-unmanaged-policy", "kyma-system")
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
				if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: config.SecretName}, secret); err == nil {
					Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
				}
			}
		})

		createCRWithNetworkPoliciesAndWaitForReady := func(networkPoliciesEnabled bool) {
			if !networkPoliciesEnabled {
				if cr.Annotations == nil {
					cr.Annotations = make(map[string]string)
				}
				cr.Annotations[v1alpha1.DisableNetworkPoliciesAnnotation] = "true"
			} else {
				if cr.Annotations != nil {
					delete(cr.Annotations, v1alpha1.DisableNetworkPoliciesAnnotation)
				}
			}

			Eventually(func() error { return k8sClient.Create(ctx, cr) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

			var err error
			secret, err = createCorrectSecretFromYaml()
			Expect(err).To(BeNil())
			Expect(k8sClient.Create(ctx, secret, client.FieldOwner(operatorName))).To(Succeed())

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
			if cr.Annotations == nil {
				cr.Annotations = make(map[string]string)
			}
			if enabled {
				delete(cr.Annotations, v1alpha1.DisableNetworkPoliciesAnnotation)
			} else {
				cr.Annotations[v1alpha1.DisableNetworkPoliciesAnnotation] = "true"
			}
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
					policy, err := getNetworkPolicy(ctx, policyName, "kyma-system")
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
					_, err := getNetworkPolicy(ctx, policyName, "kyma-system")
					if err == nil {
						return false
					}
				}
				return true
			}).Should(BeTrue(), "All network policies should be deleted")
		}

		Context("When network policies are enabled", func() {
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

		Context("When network policies are disabled", func() {
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

	Context("When testing migration logic", func() {
		It("should delete the old webhook network policy during migration", func() {
			npMgr := networkpolicy.NewManager(k8sClient, &manifest.Handler{Scheme: k8sManager.GetScheme()})
			reconciler := &BtpOperatorReconciler{
				Client:               k8sClient,
				networkPolicyManager: npMgr,
			}
			oldPolicy := createMockNetworkPolicy("kyma-project.io--btp-operator-allow-to-webhook")
			Expect(k8sClient.Create(context.Background(), oldPolicy)).To(Succeed())
			_, err := getNetworkPolicy(context.Background(), "kyma-project.io--btp-operator-allow-to-webhook", "kyma-system")
			Expect(err).NotTo(HaveOccurred())
			err = reconciler.deleteOldWebhookNetworkPolicy(context.Background())
			Expect(err).NotTo(HaveOccurred())
			_, err = getNetworkPolicy(context.Background(), "kyma-project.io--btp-operator-allow-to-webhook", "kyma-system")
			Expect(err).To(HaveOccurred())
		})
	})
})
