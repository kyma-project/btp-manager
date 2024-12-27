package controllers

import (
	"context"
	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/internal/conditions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const managementNamespace = "management"

var _ = Describe("BTP Operator controller - secret customization", Label("customization"), func() {
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
		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())
		Eventually(updateCh).Should(Receive(matchDeleted()))
		Expect(isCrNotFound()).To(BeTrue())
	})

	Describe("The sap-btp-manager secret exists with default values", func() {

		When("the secret has original unchanged values", func() {
			It("should install chart successfully and survive resource reconciliation", func() {
				btpManagerSecret, err := createCorrectSecretFromYaml()
				Expect(err).To(BeNil())

				Expect(k8sClient.Patch(ctx, btpManagerSecret, client.Apply, client.ForceOwnership, client.FieldOwner(operatorName))).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
				btpServiceOperatorDeployment := &appsv1.Deployment{}
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).To(Succeed())

				expectSecretToHaveCredentials(getOperatorSecret(), "test_clientid", "test_clientsecret", "test_sm_url", "test_tokenurl")
				expectConfigMapToHave(getOperatorConfigMap(), "test_cluster_id", "kyma-system")

				reconciler.reconcileResources(ctx, btpManagerSecret)
				//Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
				expectSecretToHaveCredentials(getOperatorSecret(), "test_clientid", "test_clientsecret", "test_sm_url", "test_tokenurl")
				expectConfigMapToHave(getOperatorConfigMap(), "test_cluster_id", "kyma-system")
			})
		})

		When("the required secret has cluster_id changed", func() {
			It("should reconcile and change value in config", func() {
				btpManagerSecret, err := createCorrectSecretFromYaml()
				Expect(err).To(BeNil())

				btpManagerSecret.Data["CLUSTER_ID"] = []byte("new_cluster_id")
				Expect(k8sClient.Patch(ctx, btpManagerSecret, client.Apply, client.ForceOwnership, client.FieldOwner("user"))).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
				btpServiceOperatorDeployment := &appsv1.Deployment{}
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).To(Succeed())

				expectSecretToHaveCredentials(getOperatorSecret(), "test_clientid", "test_clientsecret", "test_sm_url", "test_tokenurl")
				expectConfigMapToHave(getOperatorConfigMap(), "new_cluster_id", "kyma-system")

				reconciler.reconcileResources(ctx, btpManagerSecret)
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
				expectSecretToHaveCredentials(getOperatorSecret(), "test_clientid", "test_clientsecret", "test_sm_url", "test_tokenurl")
				expectConfigMapToHave(getOperatorConfigMap(), "new_cluster_id", "kyma-system")
			})
		})

		When("the required secret has managementNamespace changed", func() {
			It("should reconcile and change value in config and change location of operator secret", func() {
				btpManagerSecret, err := createCorrectSecretFromYaml()
				Expect(err).To(BeNil())

				btpManagerSecret.Data["MANAGEMENT_NAMESPACE"] = []byte(managementNamespace)
				Expect(k8sClient.Patch(ctx, btpManagerSecret, client.Apply, client.ForceOwnership, client.FieldOwner("user"))).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
				btpServiceOperatorDeployment := &appsv1.Deployment{}
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).To(Succeed())

				expectSecretToHaveCredentials(getSecretFromNamespace(btpServiceOperatorSecret, managementNamespace), "test_clientid", "test_clientsecret", "test_sm_url", "test_tokenurl")
				expectConfigMapToHave(getOperatorConfigMap(), "test_cluster_id", managementNamespace)

				reconciler.reconcileResources(ctx, btpManagerSecret)
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
				expectSecretToHaveCredentials(getSecretFromNamespace(btpServiceOperatorSecret, managementNamespace), "test_clientid", "test_clientsecret", "test_sm_url", "test_tokenurl")
				expectConfigMapToHave(getOperatorConfigMap(), "test_cluster_id", managementNamespace)
			})
		})

		When("the required secret has credentials changed", func() {
			It("should reconcile and change the values in the operator secret", func() {
				btpManagerSecret, err := createCorrectSecretFromYaml()
				Expect(err).To(BeNil())

				btpManagerSecret.Data["CLIENT_ID"] = []byte("new_clientid")
				Expect(k8sClient.Patch(ctx, btpManagerSecret, client.Apply, client.ForceOwnership, client.FieldOwner("user"))).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
				btpServiceOperatorDeployment := &appsv1.Deployment{}
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).To(Succeed())

				expectSecretToHaveCredentials(getSecretFromNamespace(btpServiceOperatorSecret, managementNamespace), "new_clientid", "test_clientsecret", "test_sm_url", "test_tokenurl")
				expectConfigMapToHave(getOperatorConfigMap(), "test_cluster_id", managementNamespace)

				reconciler.reconcileResources(ctx, btpManagerSecret)
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
				expectSecretToHaveCredentials(getSecretFromNamespace(btpServiceOperatorSecret, managementNamespace), "new_clientid", "test_clientsecret", "test_sm_url", "test_tokenurl")
				expectConfigMapToHave(getOperatorConfigMap(), "test_cluster_id", managementNamespace)
			})
		})
	})
})

func expectSecretToHaveCredentials(secret *corev1.Secret, clientId, clientSecret, smUrl, tokenUrl string) {
	Expect(secret.Data).To(HaveKeyWithValue("clientid", []byte(clientId)))
	Expect(secret.Data).To(HaveKeyWithValue("clientsecret", []byte(clientSecret)))
	Expect(secret.Data).To(HaveKeyWithValue("sm_url", []byte(smUrl)))
	Expect(secret.Data).To(HaveKeyWithValue("tokenurl", []byte(tokenUrl)))
}

func expectConfigMapToHave(configMap *corev1.ConfigMap, clusterId, managmentNamespace string) {
	Expect(configMap.Data).To(HaveKeyWithValue("MANAGEMENT_NAMESPACE", managmentNamespace))
	Expect(configMap.Data).To(HaveKeyWithValue("CLUSTER_ID", clusterId))
}
