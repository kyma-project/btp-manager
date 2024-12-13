package controllers

import (
	appsv1 "k8s.io/api/apps/v1"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/internal/conditions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterIdInit = "test_cluster_id"
	clusterIdNew  = "new-cluster-id"
)

var _ = Describe("BTP Operator controller - sap btp manager secret changes", Label("secret"), func() {
	When("sap btp secret is updated with new clust id", func() {
		cr := &v1alpha1.BtpOperator{}
		sapBtpSecret := &corev1.Secret{}
		configMap := &corev1.ConfigMap{}
		btpServiceOperatorDeployment := &appsv1.Deployment{}
		clusterSecret := &corev1.Secret{}

		BeforeEach(func() {
			var err error
			sapBtpSecret, err = createCorrectSecretFromYaml()
			Expect(err).To(BeNil())
			Eventually(func() error {
				return k8sClient.Create(ctx, sapBtpSecret)
			}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

			cr = createDefaultBtpOperator()
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			Eventually(updateCh).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Receive(matchReadyCondition(v1alpha1.StateProcessing, metav1.ConditionFalse, conditions.Initialized)))

			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)
			}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

			clusterSecret = generateClusterIDSecret(clusterIdInit)
			Eventually(func() error {
				return k8sClient.Create(ctx, clusterSecret)
			}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: btpServiceOperatorConfigMap}, configMap)
			}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())
			Eventually(updateCh).Should(Receive(matchDeleted()))
			Expect(isCrNotFound()).To(BeTrue())
			Eventually(func() error { return k8sClient.Delete(ctx, sapBtpSecret) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
			Eventually(func() error { return k8sClient.Delete(ctx, clusterSecret) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
		})

		When("change cluster id in sap btp secret", func() {
			It("should restart and update secret", func() {
				// simulate update
				if sapBtpSecret.Data == nil {
					sapBtpSecret.Data = make(map[string][]byte)
				}
				sapBtpSecret.Data[sapBtpManagerSecretClusterIdKey] = []byte(clusterIdNew)
				Eventually(func() error { return k8sClient.Update(ctx, sapBtpSecret) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

				// simulate creation of secret created by SAP BTP Operator
				clusterSecret := generateClusterIDSecret(clusterIdNew)
				Eventually(func() error {
					return k8sClient.Update(ctx, clusterSecret)
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))

				_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKey{Name: cr.Name, Namespace: cr.Namespace}})
				Expect(err).To(BeNil())

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
			})
		})
	})
})

func isMatch(clusterId, sapBtpManagerSecret *corev1.Secret, configMap *corev1.ConfigMap) bool {
	match1 := reflect.DeepEqual(sapBtpManagerSecret.Data[sapBtpManagerSecretClusterIdKey], clusterId.Data[clusterIdSecretKey])
	match2 := strings.EqualFold(string(sapBtpManagerSecret.Data[sapBtpManagerSecretClusterIdKey]), configMap.Data[clusterIdKeyConfigMap])
	return match1 && match2 && strings.EqualFold(string(clusterId.Data[clusterIdSecretKey]), clusterIdNew)
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
