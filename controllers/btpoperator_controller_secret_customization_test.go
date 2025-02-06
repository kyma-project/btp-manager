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
	clusterIdSecretKey         = "cluster_id"
	clientIdSecretKey          = "clientid"
	clientSecretSecretKey      = "clientsecret"
	tokenUrlSecretKey          = "tokenurl"
	smUrlSecretKey             = "sm_url"
	customCredentialsNamespace = "credentials-namespace"
)

var _ = Describe("BTP Operator controller - secret customization", Label("customization"), func() {
	var cr *v1alpha1.BtpOperator

	defaultBtpManagerSecret, err := createCorrectSecretFromYaml()
	Expect(err).To(BeNil())

	var (
		defaultClientId     = string(defaultBtpManagerSecret.Data[clientIdSecretKey])
		defaultClientSecret = string(defaultBtpManagerSecret.Data[clientSecretSecretKey])
		defaultTokenUrl     = string(defaultBtpManagerSecret.Data[tokenUrlSecretKey])
		defaultSmUrl        = string(defaultBtpManagerSecret.Data[smUrlSecretKey])
		defaultClusterId    = string(defaultBtpManagerSecret.Data[clusterIdSecretKey])
	)

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

	When("the required secret has original unchanged values", func() {
		It("should install chart successfully and survive resource reconciliation with no changes", func() {
			btpManagerSecret, err := createCorrectSecretFromYaml()
			Expect(err).To(BeNil())

			Expect(k8sClient.Create(ctx, btpManagerSecret)).To(Succeed())
			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
			btpServiceOperatorDeployment := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).To(Succeed())

			expectSecretToHaveCredentials(getOperatorSecret(), defaultClientId, defaultClientSecret, defaultSmUrl, defaultTokenUrl)
			expectConfigMapToHave(getOperatorConfigMap(), defaultClusterId, kymaNamespace, kymaNamespace)

			//nothing to reconcile
			_ = reconciler.enqueueOldestBtpOperator()
			Expect(err).To(BeNil())

			Eventually(updateCh).ShouldNot(Receive())

			expectSecretToHaveCredentials(getOperatorSecret(), defaultClientId, defaultClientSecret, defaultSmUrl, defaultTokenUrl)
			expectConfigMapToHave(getOperatorConfigMap(), defaultClusterId, kymaNamespace, kymaNamespace)

		})
	})

	When("the required secret is created with a custom cluster ID", func() {
		It("should apply the custom cluster ID in ConfigMap", func() {
			btpManagerSecret, err := createCorrectSecretFromYaml()
			Expect(err).To(BeNil())

			const clusterId = "new_cluster_id"
			btpManagerSecret.StringData[ClusterIdSecretKey] = clusterId
			Expect(k8sClient.Create(ctx, btpManagerSecret)).To(Succeed())
			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
			btpServiceOperatorDeployment := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).To(Succeed())

			expectSecretToHaveCredentials(getOperatorSecret(), defaultClientId, defaultClientSecret, defaultSmUrl, defaultTokenUrl)
			expectConfigMapToHave(getOperatorConfigMap(), clusterId, kymaNamespace, kymaNamespace)

			_ = reconciler.enqueueOldestBtpOperator()
			Expect(err).To(BeNil())

			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
			expectSecretToHaveCredentials(getOperatorSecret(), defaultClientId, defaultClientSecret, defaultSmUrl, defaultTokenUrl)
			expectConfigMapToHave(getOperatorConfigMap(), clusterId, kymaNamespace, kymaNamespace)
		})
	})

	When("the required secret is created with a custom credentials namespace", func() {
		It("should apply the custom credentials namespace in ConfigMap", func() {
			btpManagerSecret, err := createCorrectSecretFromYaml()
			Expect(err).To(BeNil())

			btpManagerSecret.StringData[CredentialsNamespaceSecretKey] = customCredentialsNamespace
			Expect(k8sClient.Create(ctx, btpManagerSecret)).To(Succeed())
			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
			btpServiceOperatorDeployment := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).To(Succeed())

			expectSecretToHaveCredentials(getSecretFromNamespace(sapBtpServiceOperatorSecretName, customCredentialsNamespace), defaultClientId, defaultClientSecret, defaultSmUrl, defaultTokenUrl)
			expectConfigMapToHave(getOperatorConfigMap(), defaultClusterId, customCredentialsNamespace, customCredentialsNamespace)

			_ = reconciler.enqueueOldestBtpOperator()
			Expect(err).To(BeNil())

			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
			expectSecretToHaveCredentials(getSecretFromNamespace(sapBtpServiceOperatorSecretName, customCredentialsNamespace), defaultClientId, defaultClientSecret, defaultSmUrl, defaultTokenUrl)
			expectConfigMapToHave(getOperatorConfigMap(), defaultClusterId, customCredentialsNamespace, customCredentialsNamespace)
		})
	})

	When("the required secret has client ID changed", func() {
		It("should reconcile and change only the client ID", func() {

			const clientId = "new-clientid"

			btpManagerSecret, err := createCorrectSecretFromYaml()
			Expect(err).To(BeNil())

			Expect(k8sClient.Create(ctx, btpManagerSecret)).To(Succeed())
			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
			btpServiceOperatorDeployment := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).To(Succeed())

			expectSecretToHaveCredentials(getOperatorSecret(), defaultClientId, defaultClientSecret, defaultSmUrl, defaultTokenUrl)
			expectConfigMapToHave(getOperatorConfigMap(), defaultClusterId, kymaNamespace, kymaNamespace)

			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: SecretName, Namespace: kymaNamespace}, btpManagerSecret)).To(Succeed())
			btpManagerSecret.StringData[clientIdSecretKey] = clientId
			Expect(k8sClient.Update(ctx, btpManagerSecret)).To(Succeed())

			_ = reconciler.enqueueOldestBtpOperator()
			Expect(err).To(BeNil())

			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
			expectSecretToHaveCredentials(getOperatorSecret(), clientId, defaultClientSecret, defaultSmUrl, defaultTokenUrl)
			expectConfigMapToHave(getOperatorConfigMap(), defaultClusterId, kymaNamespace, kymaNamespace)
		})
	})

	When("the required secret has client ID, cluster ID, credentials namespace changed", func() {
		It("should reconcile and change the values in secrets and configmap", func() {

			const (
				clientId  = "new-clientid"
				clusterId = "new-cluster-id"
			)

			btpManagerSecret, err := createCorrectSecretFromYaml()
			Expect(err).To(BeNil())

			Expect(k8sClient.Create(ctx, btpManagerSecret)).To(Succeed())
			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
			btpServiceOperatorDeployment := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).To(Succeed())

			expectSecretToHaveCredentials(getOperatorSecret(), defaultClientId, defaultClientSecret, defaultSmUrl, defaultTokenUrl)
			expectConfigMapToHave(getOperatorConfigMap(), defaultClusterId, kymaNamespace, kymaNamespace)

			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: SecretName, Namespace: kymaNamespace}, btpManagerSecret)).To(Succeed())
			btpManagerSecret.StringData[clientIdSecretKey] = clientId
			btpManagerSecret.StringData[clusterIdSecretKey] = clusterId
			btpManagerSecret.StringData[CredentialsNamespaceSecretKey] = customCredentialsNamespace
			Expect(k8sClient.Update(ctx, btpManagerSecret)).To(Succeed())

			_ = reconciler.enqueueOldestBtpOperator()
			Expect(err).To(BeNil())

			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
			expectSecretToHaveCredentials(getSecretFromNamespace(sapBtpServiceOperatorSecretName, customCredentialsNamespace), clientId, defaultClientSecret, defaultSmUrl, defaultTokenUrl)
			expectConfigMapToHave(getOperatorConfigMap(), clusterId, customCredentialsNamespace, customCredentialsNamespace)
		})
	})
})

func expectSecretToHaveCredentials(secret *corev1.Secret, clientId, clientSecret, smUrl, tokenUrl string) {
	Expect(secret.Data).To(HaveKeyWithValue(clientIdSecretKey, []byte(clientId)))
	Expect(secret.Data).To(HaveKeyWithValue(clientSecretSecretKey, []byte(clientSecret)))
	Expect(secret.Data).To(HaveKeyWithValue(smUrlSecretKey, []byte(smUrl)))
	Expect(secret.Data).To(HaveKeyWithValue(tokenUrlSecretKey, []byte(tokenUrl)))
}

func expectConfigMapToHave(configMap *corev1.ConfigMap, clusterId, releaseNamespace, managementNamespace string) {
	Expect(configMap.Data).To(HaveKeyWithValue(ClusterIdConfigMapKey, clusterId))
	Expect(configMap.Data).To(HaveKeyWithValue(ReleaseNamespaceConfigMapKey, releaseNamespace))
	Expect(configMap.Data).To(HaveKeyWithValue(ManagementNamespaceConfigMapKey, managementNamespace))
}
