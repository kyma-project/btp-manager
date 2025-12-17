package controllers

import (
	"context"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/conditions"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
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
	var credentialsNamespace *corev1.Namespace
	var defaultClientId, defaultClientSecret, defaultTokenUrl, defaultSmUrl, defaultClusterId string

	BeforeEach(func() {
		GinkgoWriter.Println("--- PROCESS:", GinkgoParallelProcess(), "---")
		ctx = context.Background()
		defaultBtpManagerSecret, err := createCorrectSecretFromYaml()
		Expect(err).To(BeNil())

		defaultClientId = string(defaultBtpManagerSecret.Data[clientIdSecretKey])
		defaultClientSecret = string(defaultBtpManagerSecret.Data[clientSecretSecretKey])
		defaultTokenUrl = string(defaultBtpManagerSecret.Data[tokenUrlSecretKey])
		defaultSmUrl = string(defaultBtpManagerSecret.Data[smUrlSecretKey])
		defaultClusterId = string(defaultBtpManagerSecret.Data[clusterIdSecretKey])

		credentialsNamespace = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: customCredentialsNamespace}}
		if k8serrors.IsNotFound(k8sClient.Get(ctx, client.ObjectKeyFromObject(credentialsNamespace), credentialsNamespace)) {
			Eventually(func() error { return k8sClient.Create(ctx, credentialsNamespace) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
		}

		cr = createDefaultBtpOperator()
		cr.SetLabels(map[string]string{forceDeleteLabelKey: "true"})
		Eventually(func() error { return k8sClient.Create(ctx, cr) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
	})

	AfterEach(func() {
		cr, secret := &v1alpha1.BtpOperator{}, &corev1.Secret{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: btpOperatorName}, cr)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())
		Eventually(updateCh).Should(Receive(matchDeleted()))
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: config.SecretName, Namespace: kymaNamespace}, secret)).To(Succeed())
		Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
		Expect(isCrNotFound()).To(BeTrue())
	})

	When("the required secret has original unchanged values", func() {
		It("should install chart successfully and survive resource reconciliation with no changes", func() {
			btpManagerSecret, err := createCorrectSecretFromYaml()
			Expect(err).To(BeNil())

			Expect(k8sClient.Create(ctx, btpManagerSecret)).To(Succeed())
			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
			btpServiceOperatorDeployment := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: config.DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).To(Succeed())

			expectSecretToHaveCredentials(getOperatorSecret(), defaultClientId, defaultClientSecret, defaultSmUrl, defaultTokenUrl)
			expectConfigMapToHave(getOperatorConfigMap(), defaultClusterId, kymaNamespace, kymaNamespace)

			_, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKey{Name: btpOperatorName, Namespace: kymaNamespace}})
			Expect(err).To(BeNil())

			expectSecretToHaveCredentials(getOperatorSecret(), defaultClientId, defaultClientSecret, defaultSmUrl, defaultTokenUrl)
			expectConfigMapToHave(getOperatorConfigMap(), defaultClusterId, kymaNamespace, kymaNamespace)
		})
	})

	When("the required secret is created with a custom cluster ID", func() {
		It("should apply the custom cluster ID in ConfigMap", func() {
			btpManagerSecret, err := createCorrectSecretFromYaml()
			Expect(err).To(BeNil())

			const clusterId = "new_cluster_id"
			btpManagerSecret.Data[ClusterIdSecretKey] = []byte(clusterId)
			Expect(k8sClient.Create(ctx, btpManagerSecret)).To(Succeed())
			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
			btpServiceOperatorDeployment := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: config.DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).To(Succeed())

			expectSecretToHaveCredentials(getOperatorSecret(), defaultClientId, defaultClientSecret, defaultSmUrl, defaultTokenUrl)
			expectConfigMapToHave(getOperatorConfigMap(), clusterId, kymaNamespace, kymaNamespace)

			_, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKey{Name: btpOperatorName, Namespace: kymaNamespace}})
			Expect(err).To(BeNil())

			expectSecretToHaveCredentials(getOperatorSecret(), defaultClientId, defaultClientSecret, defaultSmUrl, defaultTokenUrl)
			expectConfigMapToHave(getOperatorConfigMap(), clusterId, kymaNamespace, kymaNamespace)
		})
	})

	When("the required secret is created with a custom credentials namespace", func() {
		It("should apply the custom credentials namespace in ConfigMap", func() {
			btpManagerSecret, err := createCorrectSecretFromYaml()
			Expect(err).To(BeNil())

			btpManagerSecret.Data[CredentialsNamespaceSecretKey] = []byte(customCredentialsNamespace)
			Expect(k8sClient.Create(ctx, btpManagerSecret)).To(Succeed())
			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
			btpServiceOperatorDeployment := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: config.DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).To(Succeed())

			expectSecretToHaveCredentials(getSecretFromNamespace(sapBtpServiceOperatorSecretName, customCredentialsNamespace), defaultClientId, defaultClientSecret, defaultSmUrl, defaultTokenUrl)
			expectConfigMapToHave(getOperatorConfigMap(), defaultClusterId, customCredentialsNamespace, customCredentialsNamespace)

			_, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKey{Name: btpOperatorName, Namespace: kymaNamespace}})
			Expect(err).To(BeNil())

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
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: config.DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).To(Succeed())

			expectSecretToHaveCredentials(getOperatorSecret(), defaultClientId, defaultClientSecret, defaultSmUrl, defaultTokenUrl)
			expectConfigMapToHave(getOperatorConfigMap(), defaultClusterId, kymaNamespace, kymaNamespace)

			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: config.SecretName, Namespace: kymaNamespace}, btpManagerSecret)).To(Succeed())
			btpManagerSecret.Data[clientIdSecretKey] = []byte(clientId)
			Expect(k8sClient.Update(ctx, btpManagerSecret)).To(Succeed())
			Eventually(func() string {
				secret := getOperatorSecret()
				return string(secret.Data[clientIdSecretKey])
			}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(BeIdenticalTo(clientId))

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
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: config.DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).To(Succeed())

			expectSecretToHaveCredentials(getOperatorSecret(), defaultClientId, defaultClientSecret, defaultSmUrl, defaultTokenUrl)
			expectConfigMapToHave(getOperatorConfigMap(), defaultClusterId, kymaNamespace, kymaNamespace)

			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: config.SecretName, Namespace: kymaNamespace}, btpManagerSecret)).To(Succeed())
			btpManagerSecret.Data[clientIdSecretKey] = []byte(clientId)
			btpManagerSecret.Data[clusterIdSecretKey] = []byte(clusterId)
			btpManagerSecret.Data[CredentialsNamespaceSecretKey] = []byte(customCredentialsNamespace)
			Expect(k8sClient.Update(ctx, btpManagerSecret)).To(Succeed())

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
