package controllers

import (
	"time"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/controllers/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Configuration controller", func() {
	var cr *v1alpha1.BtpOperator

	Context("When EnableLimitedCache is created/updated", func() {
		var originalValue string

		BeforeEach(func() {
			secret, err := createCorrectSecretFromYaml()
			Expect(err).To(BeNil())
			Expect(k8sClient.Create(ctx, secret, client.FieldOwner(operatorName))).To(Succeed())

			cr = createDefaultBtpOperator()
			cr.SetLabels(map[string]string{forceDeleteLabelKey: "true"})
			Eventually(func() error { return k8sClient.Create(ctx, cr) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
			Eventually(updateCh).Should(Receive(matchState(v1alpha1.StateReady)))

			existing := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: sapBtpServiceOperatorConfigMapName, Namespace: kymaNamespace}, existing)).To(Succeed())
			existing.Data = map[string]string{
				EnableLimitedCacheConfigMapKey: "false",
			}
			Expect(k8sClient.Update(ctx, existing)).To(Succeed())

			originalValue = config.EnableLimitedCache
		})

		AfterEach(func() {
			cr = &v1alpha1.BtpOperator{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: btpOperatorName}, cr)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())
			Eventually(updateCh).Should(Receive(matchDeleted()))
			Expect(isCrNotFound()).To(BeTrue())

			deleteSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: config.SecretName}, deleteSecret)).To(Succeed())
			Expect(k8sClient.Delete(ctx, deleteSecret)).To(Succeed())

			config.EnableLimitedCache = originalValue
		})

		It("should update EnableLimitedCache", func() {
			createOrUpdateConfigMap(map[string]string{
				"EnableLimitedCache": "true",
			})

			Eventually(func() string {
				return config.EnableLimitedCache
			}).Should(Equal("true"))

			Eventually(func() map[string]string {
				return getOperatorConfigMap().Data
			}).Should(HaveKeyWithValue(EnableLimitedCacheConfigMapKey, "true"))
		})
	})

	Context("When ProcessingStateRequeueInterval is created/updated", func() {
		var originalValue time.Duration

		BeforeEach(func() {
			originalValue = config.ProcessingStateRequeueInterval
		})

		AfterEach(func() {
			config.ProcessingStateRequeueInterval = originalValue
		})

		It("should update ProcessingStateRequeueInterval", func() {
			createOrUpdateConfigMap(map[string]string{
				"ProcessingStateRequeueInterval": "10s",
			})

			Eventually(func() time.Duration {
				return config.ProcessingStateRequeueInterval
			}).Should(Equal(10 * time.Second))
		})
	})
})
