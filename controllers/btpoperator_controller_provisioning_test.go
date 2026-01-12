package controllers

import (
	"context"
	"os"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/conditions"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("BTP Operator controller - provisioning", func() {
	var cr *v1alpha1.BtpOperator

	BeforeEach(func() {
		GinkgoWriter.Println("--- PROCESS:", GinkgoParallelProcess(), "---")
		ctx = context.Background()
		cr = createDefaultBtpOperator()
		cr.SetLabels(map[string]string{forceDeleteLabelKey: "true"})
		Eventually(func() error { return k8sClient.Create(ctx, cr) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
	})

	AfterEach(func() {
		cr = &v1alpha1.BtpOperator{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: btpOperatorName}, cr)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())
		Eventually(updateCh).Should(Receive(matchDeleted()))
		Expect(isCrNotFound()).To(BeTrue())
	})

	When("The required Secret is missing", func() {
		It("should return warning while getting the required Secret", func() {
			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateProcessing, metav1.ConditionFalse, conditions.Initialized)))
			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateWarning, metav1.ConditionFalse, conditions.MissingSecret)))
		})
	})

	Describe("The required Secret exists", func() {
		AfterEach(func() {
			deleteSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: config.SecretName}, deleteSecret)).To(Succeed())
			Expect(k8sClient.Delete(ctx, deleteSecret)).To(Succeed())
			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateWarning, metav1.ConditionFalse, conditions.MissingSecret)))
		})

		When("the required Secret does not have all required keys", func() {
			It("should return error while verifying keys", func() {
				secret, err := createSecretWithoutKeys()
				Expect(err).To(BeNil())
				Expect(k8sClient.Create(ctx, secret)).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateWarning, metav1.ConditionFalse, conditions.InvalidSecret)))
			})
		})

		When("the required Secret's keys do not have all values", func() {
			It("should return error while verifying values", func() {
				secret, err := createSecretWithoutValues()
				Expect(err).To(BeNil())
				Expect(k8sClient.Create(ctx, secret)).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateWarning, metav1.ConditionFalse, conditions.InvalidSecret)))
			})
		})

		When("the required Secret is correct", func() {
			It("should install chart successfully", func() {
				secret, err := createCorrectSecretFromYaml()
				Expect(err).To(BeNil())
				Expect(k8sClient.Patch(ctx, secret, client.Apply, client.ForceOwnership, client.FieldOwner(operatorName))).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
				btpServiceOperatorDeployment := &appsv1.Deployment{}
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: config.DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).To(Succeed())
			})

			It("should set EnableLimitedCache to false by default in operator ConfigMap", func() {
				secret, err := createCorrectSecretFromYaml()
				Expect(err).To(BeNil())
				Expect(k8sClient.Patch(ctx, secret, client.Apply, client.ForceOwnership, client.FieldOwner(operatorName))).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))

				operatorConfigMap := getOperatorConfigMap()
				Expect(operatorConfigMap.Data).To(HaveKeyWithValue(EnableLimitedCacheConfigMapKey, "false"))
			})

			Context("when EnableLimitedCache configuration is modified", func() {
				var originalValue string

				BeforeEach(func() {
					originalValue = config.EnableLimitedCache
				})

				AfterEach(func() {
					config.EnableLimitedCache = originalValue
				})

				It("should set EnableLimitedCache to true in operator ConfigMap when configured", func() {

					// set via reconciler to exercise production code path
					createOrUpdateConfigMap(map[string]string{"EnableLimitedCache": "true"})
					Eventually(func() string { return config.EnableLimitedCache }).Should(Equal("true"))

					secret, err := createCorrectSecretFromYaml()
					Expect(err).To(BeNil())
					Expect(k8sClient.Patch(ctx, secret, client.Apply, client.ForceOwnership, client.FieldOwner(operatorName))).To(Succeed())
					Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))

					operatorConfigMap := getOperatorConfigMap()
					Expect(operatorConfigMap.Data).To(HaveKeyWithValue(EnableLimitedCacheConfigMapKey, "true"))
				})

				It("should set EnableLimitedCache to false in operator ConfigMap when explicitly configured", func() {

					// set via reconciler to exercise production code path
					createOrUpdateConfigMap(map[string]string{"EnableLimitedCache": "false"})
					Eventually(func() string { return config.EnableLimitedCache }).Should(Equal("false"))

					secret, err := createCorrectSecretFromYaml()
					Expect(err).To(BeNil())
					Expect(k8sClient.Patch(ctx, secret, client.Apply, client.ForceOwnership, client.FieldOwner(operatorName))).To(Succeed())
					Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))

					operatorConfigMap := getOperatorConfigMap()
					Expect(operatorConfigMap.Data).To(HaveKeyWithValue(EnableLimitedCacheConfigMapKey, "false"))
				})
			})

			Describe("dynamic container image setting in sap-btp-service-operator deployment", func() {
				const (
					sapBtpServiceOperatorImage = "test-sap-btp-service-operator:v0.0.1"
					kubeRbacProxyImage         = "test-kube-rbac-proxy:v0.0.1"
				)

				var (
					orgSapBtpServiceOperatorEnv string
					orgKubeRbacProxyEnv         string
				)

				BeforeEach(func() {
					orgSapBtpServiceOperatorEnv = os.Getenv(SapBtpServiceOperatorEnv)
					orgKubeRbacProxyEnv = os.Getenv(KubeRbacProxyEnv)
				})

				AfterEach(func() {
					Expect(os.Setenv(SapBtpServiceOperatorEnv, orgSapBtpServiceOperatorEnv)).To(Succeed())
					Expect(os.Setenv(KubeRbacProxyEnv, orgKubeRbacProxyEnv)).To(Succeed())
				})

				It("should set container images from environment variables", func() {
					Expect(os.Setenv(SapBtpServiceOperatorEnv, sapBtpServiceOperatorImage)).To(Succeed())
					Expect(os.Setenv(KubeRbacProxyEnv, kubeRbacProxyImage)).To(Succeed())
					secret, err := createCorrectSecretFromYaml()
					Expect(err).To(BeNil())
					Expect(k8sClient.Patch(ctx, secret, client.Apply, client.ForceOwnership, client.FieldOwner(operatorName))).To(Succeed())
					Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
					btpServiceOperatorDeployment := &appsv1.Deployment{}
					Expect(k8sClient.Get(ctx, client.ObjectKey{Name: config.DeploymentName, Namespace: config.ChartNamespace}, btpServiceOperatorDeployment)).To(Succeed())
					for _, c := range btpServiceOperatorDeployment.Spec.Template.Spec.Containers {
						if c.Name == sapBtpServiceOperatorContainerName {
							Expect(c.Image).To(Equal(sapBtpServiceOperatorImage))
						}
						if c.Name == kubeRbacProxyContainerName {
							Expect(c.Image).To(Equal(kubeRbacProxyImage))
						}
					}
				})

				It("should return reconciliation error on missing environment variables", func() {
					_ = os.Unsetenv(SapBtpServiceOperatorEnv)
					_ = os.Unsetenv(KubeRbacProxyEnv)
					secret, err := createCorrectSecretFromYaml()
					Expect(err).To(BeNil())
					Expect(k8sClient.Patch(ctx, secret, client.Apply, client.ForceOwnership, client.FieldOwner(operatorName))).To(Succeed())
					Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateError, metav1.ConditionFalse, conditions.ProvisioningFailed)))
				})
			})
		})

		When("the btpoperator resource is not in the required namespace", func() {
			It("should set state to Warning", func() {
				secret, err := createCorrectSecretFromYaml()
				Expect(err).To(BeNil())
				Expect(k8sClient.Patch(ctx, secret, client.Apply, client.ForceOwnership, client.FieldOwner(operatorName))).To(Succeed())
				cr := createDefaultBtpOperator()
				cr.SetNamespace("default")
				Expect(k8sClient.Create(ctx, cr)).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateWarning, metav1.ConditionFalse, conditions.WrongNamespaceOrName)))

				// cleanup
				Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())
				Eventually(updateCh).Should(Receive(matchDeleted()))
				Expect(isNamedCrNotFound(btpOperatorName, "default")).To(BeTrue())
			})
		})

		When("the btpoperator resource's name is not as required", func() {
			It("should set state to Warning", func() {
				secret, err := createCorrectSecretFromYaml()
				Expect(err).To(BeNil())
				Expect(k8sClient.Patch(ctx, secret, client.Apply, client.ForceOwnership, client.FieldOwner(operatorName))).To(Succeed())
				cr := createDefaultBtpOperator()
				cr.SetName("wrong")
				Expect(k8sClient.Create(ctx, cr)).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateWarning, metav1.ConditionFalse, conditions.WrongNamespaceOrName)))

				// cleanup
				Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())
				Eventually(updateCh).Should(Receive(matchDeleted()))
				Expect(isNamedCrNotFound("wrong", kymaNamespace)).To(BeTrue())
			})
		})
	})
})
