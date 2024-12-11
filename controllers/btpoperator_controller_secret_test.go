package controllers

import (
	"context"
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
		Eventually(func() error { return k8sClient.Create(ctx, cr) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
	})

	validateDataIntegrity := func(withConfigMap bool) {
		var match bool
		sapBtpSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      SecretName,
				Namespace: ChartNamespace,
			},
		}
		err := k8sClient.Get(ctx, client.ObjectKey{}, sapBtpSecret)
		Expect(err).To(BeNil())

		clusterIdSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterIdSecret,
				Namespace: ChartNamespace,
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKey{}, clusterIdSecret)
		Expect(err).To(BeNil())

		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      btpServiceOperatorConfigMap,
				Namespace: ChartNamespace,
			},
		}
		err = k8sClient.Get(ctx, client.ObjectKey{}, configMap)
		match = (sapBtpSecret.StringData[clusterIdKey] == clusterIdSecret.StringData[initialClusterIdKey]) && (sapBtpSecret.StringData[clusterIdKey] == configMap.Data[clusterIdKey])
		Expect(match).To(BeTrue())
	}

	When("sap btp secret is updated with new clust id", func() {
		It("should restart and update secret", func() {
			validateDataIntegrity(true)
			newSapBtpSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sap-btp-secret",
					Namespace: ChartNamespace,
				},
			}
			newSapBtpSecret.StringData["CLUSTER_ID"] = "new-cluster-id"
			Eventually(func() error { return k8sClient.Update(ctx, cr) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
			validateDataIntegrity(true)
		})
	})
})
