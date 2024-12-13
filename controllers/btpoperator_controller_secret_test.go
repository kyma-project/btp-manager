package controllers

import (
	"fmt"
	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/internal/conditions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("BTP Operator controller - sap btp manager secret changes", Label("secret"), func() {
	When("sap btp secret is updated with new clust id", func() {

		When("change cluster id in sap btp secret", func() {
			It("should restart and update secret", func() {
				var match bool
				sapBtpSecret := &corev1.Secret{}
				clusterSecret := &corev1.Secret{}
				configMap := &corev1.ConfigMap{}
				var cr *v1alpha1.BtpOperator

				sapBtpSecret, err := createCorrectSecretFromYaml()
				Expect(err).To(BeNil())
				Eventually(func() error {
					return k8sClient.Patch(ctx, sapBtpSecret, client.Apply, client.ForceOwnership, client.FieldOwner(operatorName))
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

				cr = createDefaultBtpOperator()
				Expect(k8sClient.Create(ctx, cr)).To(Succeed())
				Eventually(updateCh).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))

				btpServiceOperatorDeployment := &appsv1.Deployment{}
				Eventually(func() error {
					return k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

				// check before integrity
				Eventually(func() error {
					return k8sClient.Get(ctx, client.ObjectKey{Name: SecretName, Namespace: kymaNamespace}, sapBtpSecret)
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

				Eventually(func() error {
					return k8sClient.Get(ctx, client.ObjectKey{Name: clusterIdSecretName, Namespace: kymaNamespace}, clusterSecret)
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

				Eventually(func() error {
					return k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: btpServiceOperatorConfigMap}, configMap)
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

				match = (string(sapBtpSecret.Data[sapBtpManagerSecretClusterIdKey]) == string(clusterSecret.Data[clusterIdSecretKey])) && (string(sapBtpSecret.Data[sapBtpManagerSecretClusterIdKey]) == configMap.Data[clusterIdKeyConfigMap])
				fmt.Println("sapBtpSecret.Data[sapBtpManagerClusterIdKey]: ", string(sapBtpSecret.Data[sapBtpManagerSecretClusterIdKey]))
				fmt.Println("clusterSecret.Data[initialClusterIdKey]: ", string(clusterSecret.Data[clusterIdSecretKey]))
				fmt.Println("configMap.Data[clusterIdKey]: ", configMap.Data[clusterIdKeyConfigMap])

				Expect(match).To(BeTrue())

				// simulate update
				sapBtpSecret.Data[sapBtpManagerSecretClusterIdKey] = []byte("new-cluster-id")
				Eventually(func() error { return k8sClient.Update(ctx, sapBtpSecret) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
				_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}})
				if err != nil {
					return
				}
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

				match = (sapBtpSecret.StringData[clusterIdKeyConfigMap] == clusterSecret.StringData[clusterIdSecretKey]) && (sapBtpSecret.StringData[clusterIdKeyConfigMap] == configMap.Data[clusterIdKeyConfigMap])
				match = match && sapBtpSecret.StringData[clusterIdKeyConfigMap] == "new-cluster-id"
				fmt.Println("sapBtpSecret.Data[sapBtpManagerClusterIdKey]: ", string(sapBtpSecret.Data[sapBtpManagerSecretClusterIdKey]))
				fmt.Println("clusterSecret.Data[initialClusterIdKey]: ", string(clusterSecret.Data[clusterIdSecretKey]))
				fmt.Println("configMap.Data[clusterIdKey]: ", configMap.Data[clusterIdKeyConfigMap])
				Expect(match).To(BeTrue())

				GinkgoWriter.Println("start AfterEach")

				cr = &v1alpha1.BtpOperator{}
				Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).Should(Succeed())
				Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())
				Eventually(updateCh).Should(Receive(matchDeleted()))
				Expect(isCrNotFound()).To(BeTrue())
				deleteSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      SecretName,
						Namespace: kymaNamespace,
					},
				}
				Eventually(func() error { return k8sClient.Delete(ctx, deleteSecret) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
				GinkgoWriter.Println("end AfterEach")
			})
		})
	})
})

func validateDataIntegrity() {

}
