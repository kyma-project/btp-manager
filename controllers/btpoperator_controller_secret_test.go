package controllers

import (
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	"strings"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/internal/conditions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("BTP Operator controller - sap btp manager secret changes", Label("secret"), func() {
	When("sap btp secret is updated with new clust id", func() {

		When("change cluster id in sap btp secret", func() {
			It("should restart and update secret", func() {
				cr := &v1alpha1.BtpOperator{}
				sapBtpSecret := &corev1.Secret{}
				clusterSecret := &corev1.Secret{}
				configMap := &corev1.ConfigMap{}

				sapBtpSecret, err := createCorrectSecretFromYaml()

				Expect(err).To(BeNil())
				Eventually(func() error {
					return k8sClient.Create(ctx, sapBtpSecret)
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

				cr = createDefaultBtpOperator()
				Expect(k8sClient.Create(ctx, cr)).To(Succeed())
				Eventually(updateCh).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Receive(matchReadyCondition(v1alpha1.StateProcessing, metav1.ConditionFalse, conditions.Initialized)))

				btpServiceOperatorDeployment := &appsv1.Deployment{}
				Eventually(func() error {
					return k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

				clusterSecret = generateClusterIDSecret("test_cluster_id")
				Eventually(func() error {
					return k8sClient.Create(ctx, clusterSecret)
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

				Eventually(func() error {
					return k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: btpServiceOperatorConfigMap}, configMap)
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

				// simulate update
				if sapBtpSecret.Data == nil {
					sapBtpSecret.Data = make(map[string][]byte)
				}
				sapBtpSecret.Data[sapBtpManagerSecretClusterIdKey] = []byte("new-cluster-id")
				Eventually(func() error { return k8sClient.Update(ctx, sapBtpSecret) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

				// simulate creation of secret created by SAP BTP Operator
				clusterSecret = generateClusterIDSecret("new-cluster-id")
				Eventually(func() error {
					return k8sClient.Update(ctx, clusterSecret)
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))

				// check integrity after update
				Eventually(func() error {
					return k8sClient.Get(ctx, client.ObjectKey{Name: SecretName, Namespace: kymaNamespace}, sapBtpSecret)
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

				Eventually(func() error {
					return k8sClient.Get(ctx, client.ObjectKey{Name: clusterIdSecretName, Namespace: kymaNamespace}, clusterSecret)
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

				Eventually(func() error {
					return k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: btpServiceOperatorConfigMap}, configMap)
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

				Expect(isMatch(clusterSecret, sapBtpSecret, configMap)).To(BeTrue())

				Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).Should(Succeed())
				Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())
				Eventually(updateCh).Should(Receive(matchDeleted()))
				Expect(isCrNotFound()).To(BeTrue())
				Eventually(func() error { return k8sClient.Delete(ctx, sapBtpSecret) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
			})
		})
	})
})

func isMatch(clusterId, secretName *corev1.Secret, configMap *corev1.ConfigMap) bool {
	match1 := clusterId.StringData[sapBtpManagerSecretClusterIdKey] == secretName.StringData[clusterIdSecretKey]

	match2 := strings.EqualFold(string(clusterId.Data[sapBtpManagerSecretClusterIdKey]), configMap.Data[clusterIdKeyConfigMap])
	fmt.Printf("string(clusterId.Data[sapBtpManagerSecretClusterIdKey] %s \n", string(clusterId.Data[sapBtpManagerSecretClusterIdKey]))
	fmt.Printf("configMap.Data[clusterIdKeyConfigMap] %s \n", configMap.Data[clusterIdKeyConfigMap])

	match := match1 && match2 && strings.EqualFold(string(clusterId.Data[sapBtpManagerSecretClusterIdKey]), "new-cluster-id")
	return match
}

func generateClusterIDSecret(key string) *corev1.Secret {
	clusterSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterIdSecretName,
			Namespace: kymaNamespace,
		},
		Data: map[string][]byte{
			clusterIdSecretKey: []byte(key),
		},
	}
	return clusterSecret
}
