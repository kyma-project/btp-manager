package controllers

import (
	"context"
	"github.com/kyma-project/btp-manager/internal/conditions"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("BTP Operator controller - sap btp manager secret changes", func() {
	var cr *v1alpha1.BtpOperator
	BeforeEach(func() {
		GinkgoWriter.Println("--- PROCESS:", GinkgoParallelProcess(), "---")
		ctx = context.Background()
		cr = createDefaultBtpOperator()
		cr.SetLabels(map[string]string{forceDeleteLabelKey: "true"})
		Eventually(func() error { return k8sClient.Create(ctx, cr) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
		secret, err := createCorrectSecretFromYaml()
		Expect(err).To(BeNil())
		Expect(k8sClient.Patch(ctx, secret, client.Apply, client.ForceOwnership, client.FieldOwner(operatorName))).To(Succeed())
		Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
		btpServiceOperatorDeployment := &appsv1.Deployment{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).To(Succeed())
	})

	AfterEach(func() {
		cr = &v1alpha1.BtpOperator{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())
		Eventually(updateCh).Should(Receive(matchDeleted()))
		Expect(isCrNotFound()).To(BeTrue())
		deleteSecret := &corev1.Secret{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: SecretName}, deleteSecret)).To(Succeed())
		Expect(k8sClient.Delete(ctx, deleteSecret)).To(Succeed())
		Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateWarning, metav1.ConditionFalse, conditions.MissingSecret)))
	})

	When("sap btp secret is updated with new clust id", func() {
		It("should restart and update secret", func() {
			validateDataIntegrity()
			newSapBtpSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      SecretName,
					Namespace: kymaNamespace,
				},
			}
			newSapBtpSecret.StringData["CLUSTER_ID"] = "new-cluster-id"
			Eventually(func() error { return k8sClient.Update(ctx, cr) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
			validateDataIntegrity()
		})
	})

})

func validateDataIntegrity() {
	var match bool
	sapBtpSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SecretName,
			Namespace: kymaNamespace,
		},
	}
	Eventually(func() error { return k8sClient.Get(ctx, client.ObjectKey{}, sapBtpSecret) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

	clusterIdSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterIdSecret,
			Namespace: kymaNamespace,
		},
	}
	Eventually(func() error { return k8sClient.Get(ctx, client.ObjectKey{}, clusterIdSecret) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      btpServiceOperatorConfigMap,
			Namespace: kymaNamespace,
		},
	}
	Eventually(func() error { return k8sClient.Get(ctx, client.ObjectKey{}, configMap) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

	match = (sapBtpSecret.StringData[clusterIdKey] == clusterIdSecret.StringData[initialClusterIdKey]) && (sapBtpSecret.StringData[clusterIdKey] == configMap.Data[clusterIdKey])
	Expect(match).To(BeTrue())
}
