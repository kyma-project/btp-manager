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

const (
	managementNamespaceValue     = "management"
	ClientIdSecretKey            = "clientid"
	ClientSecretSecretKey        = "clientsecret"
	ClusterIdSecretKey           = "cluster_id"
	ManagementNamespaceSecretKey = "management_namespace"
	ClusterIdConfigKey           = "CLUSTER_ID"
	ManagementNamespaceConfigKey = "MANAGEMENT_NAMESPACE"
)

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
			It("should install chart successfully and survive resource reconciliation with no changes", func() {
				btpManagerSecret, err := createCorrectSecretFromYaml()
				Expect(err).To(BeNil())

				Expect(k8sClient.Patch(ctx, btpManagerSecret, client.Apply, client.ForceOwnership, client.FieldOwner(operatorName))).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
				btpServiceOperatorDeployment := &appsv1.Deployment{}
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).To(Succeed())

				expectSecretToHaveCredentials(getOperatorSecret(), "test_clientid", "test_clientsecret", "test_sm_url", "test_tokenurl")
				expectConfigMapToHave(getOperatorConfigMap(), "test_cluster_id", "kyma-system")

				reconciler.reconcileResources(ctx, btpManagerSecret) // nothing to reconcile
				Eventually(updateCh).ShouldNot(Receive())

				expectSecretToHaveCredentials(getOperatorSecret(), "test_clientid", "test_clientsecret", "test_sm_url", "test_tokenurl")
				expectConfigMapToHave(getOperatorConfigMap(), "test_cluster_id", "kyma-system")
			})
		})

		When("the required secret has CLUSTER_ID changed", func() {
			It("should reconcile and change value in config", func() {
				btpManagerSecret, err := createCorrectSecretFromYaml()
				Expect(err).To(BeNil())

				btpManagerSecret.Data[ClusterIdSecretKey] = []byte("new_cluster_id")
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

		When("the required secret has MANAGEMENT_NAMESPACE changed", func() {
			It("should reconcile and change value in config and change location of operator secret", func() {
				btpManagerSecret, err := createCorrectSecretFromYaml()
				Expect(err).To(BeNil())

				btpManagerSecret.Data[ManagementNamespaceSecretKey] = []byte(managementNamespaceValue)
				Expect(k8sClient.Patch(ctx, btpManagerSecret, client.Apply, client.ForceOwnership, client.FieldOwner("user"))).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
				btpServiceOperatorDeployment := &appsv1.Deployment{}
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).To(Succeed())

				expectSecretToHaveCredentials(getSecretFromNamespace(btpServiceOperatorSecret, managementNamespaceValue), "test_clientid", "test_clientsecret", "test_sm_url", "test_tokenurl")
				expectConfigMapToHave(getOperatorConfigMap(), "test_cluster_id", managementNamespaceValue)

				reconciler.reconcileResources(ctx, btpManagerSecret)
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
				expectSecretToHaveCredentials(getSecretFromNamespace(btpServiceOperatorSecret, managementNamespaceValue), "test_clientid", "test_clientsecret", "test_sm_url", "test_tokenurl")
				expectConfigMapToHave(getOperatorConfigMap(), "test_cluster_id", managementNamespaceValue)
			})
		})

		When("the required secret has client_id changed", func() {
			It("should reconcile and change the client_id value in the operator secret", func() {
				btpManagerSecret, err := createCorrectSecretFromYaml()
				Expect(err).To(BeNil())

				btpManagerSecret.Data[ClientIdSecretKey] = []byte("new_clientid")

				Expect(k8sClient.Patch(ctx, btpManagerSecret, client.Apply, client.ForceOwnership, client.FieldOwner("user"))).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
				btpServiceOperatorDeployment := &appsv1.Deployment{}
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).To(Succeed())

				expectSecretToHaveCredentials(getSecretFromNamespace(btpServiceOperatorSecret, managementNamespaceValue), "new_clientid", "test_clientsecret", "test_sm_url", "test_tokenurl")
				expectConfigMapToHave(getOperatorConfigMap(), "test_cluster_id", managementNamespaceValue)

				reconciler.reconcileResources(ctx, btpManagerSecret)
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
				expectSecretToHaveCredentials(getSecretFromNamespace(btpServiceOperatorSecret, managementNamespaceValue), "new_clientid", "test_clientsecret", "test_sm_url", "test_tokenurl")
				expectConfigMapToHave(getOperatorConfigMap(), "test_cluster_id", managementNamespaceValue)
			})
		})

		When("the required secret has client_id, CLUSTER_ID, MANAGEMENT_NAMESPACE changed", func() {
			It("should reconcile and change the client_id value in the operator secret placed in new location and change CLUSTER_ID in the config", func() {
				btpManagerSecret, err := createCorrectSecretFromYaml()
				Expect(err).To(BeNil())

				btpManagerSecret.Data[ClientIdSecretKey] = []byte("brand_new_clientid")
				btpManagerSecret.Data[ClusterIdSecretKey] = []byte("brand_new_cluster_id")
				btpManagerSecret.Data[ManagementNamespaceSecretKey] = []byte(managementNamespaceValue)

				Expect(k8sClient.Patch(ctx, btpManagerSecret, client.Apply, client.ForceOwnership, client.FieldOwner("user"))).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
				btpServiceOperatorDeployment := &appsv1.Deployment{}
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).To(Succeed())

				expectSecretToHaveCredentials(getSecretFromNamespace(btpServiceOperatorSecret, managementNamespaceValue), "brand_new_clientid", "test_clientsecret", "test_sm_url", "test_tokenurl")
				expectConfigMapToHave(getOperatorConfigMap(), "brand_new_cluster_id", managementNamespaceValue)

				reconciler.reconcileResources(ctx, btpManagerSecret)

				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
				expectSecretToHaveCredentials(getSecretFromNamespace(btpServiceOperatorSecret, managementNamespaceValue), "brand_new_clientid", "test_clientsecret", "test_sm_url", "test_tokenurl")
				expectConfigMapToHave(getOperatorConfigMap(), "brand_new_cluster_id", managementNamespaceValue)
			})
		})
	})
})

func expectSecretToHaveCredentials(secret *corev1.Secret, clientId, clientSecret, smUrl, tokenUrl string) {
	Expect(secret.Data).To(HaveKeyWithValue(ClientIdSecretKey, []byte(clientId)))
	Expect(secret.Data).To(HaveKeyWithValue(ClientSecretSecretKey, []byte(clientSecret)))
	Expect(secret.Data).To(HaveKeyWithValue("sm_url", []byte(smUrl)))
	Expect(secret.Data).To(HaveKeyWithValue("tokenurl", []byte(tokenUrl)))
}

func expectConfigMapToHave(configMap *corev1.ConfigMap, clusterId, managmentNamespace string) {
	Expect(configMap.Data).To(HaveKeyWithValue(ManagementNamespaceConfigKey, managmentNamespace))
	Expect(configMap.Data).To(HaveKeyWithValue(ClusterIdConfigKey, clusterId))
}
