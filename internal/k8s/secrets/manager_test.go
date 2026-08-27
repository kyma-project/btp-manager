package secrets_test

import (
	"context"

	"github.com/kyma-project/btp-manager/internal/k8s/generic"
	"github.com/kyma-project/btp-manager/internal/k8s/secrets"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Secrets Manager", func() {
	var mgr secrets.Manager

	BeforeEach(func() {
		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		mgr = secrets.NewManager(
			generic.NewObjectManager[*corev1.Secret, *corev1.SecretList](fakeClient),
		)
	})

	Describe("Getting secrets", func() {
		Context("Webhook certificate secrets", func() {
			When("the CA server cert secret exists", func() {
				It("should return the secret", func() {
					expectedSecret := caServerCertSecret()
					Expect(fakeClient.Create(context.Background(), expectedSecret)).To(Succeed())

					actualSecret, err := mgr.GetCaServerCertSecret(context.Background())
					Expect(err).ToNot(HaveOccurred())

					Expect(actualSecret).To(Equal(expectedSecret))
				})
			})

			When("the CA server cert secret does not exist", func() {
				It("should return nil", func() {
					actualSecret, err := mgr.GetCaServerCertSecret(context.Background())
					Expect(err).ToNot(HaveOccurred())
					Expect(actualSecret).To(BeNil())
				})
			})

			When("the webhook server cert secret exists", func() {
				It("should return the secret", func() {
					expectedSecret := webhookServerCertSecret()
					Expect(fakeClient.Create(context.Background(), expectedSecret)).To(Succeed())

					actualSecret, err := mgr.GetWebhookServerCertSecret(context.Background())
					Expect(err).ToNot(HaveOccurred())

					Expect(actualSecret).To(Equal(expectedSecret))
				})
			})

			When("the webhook server cert secret does not exist", func() {
				It("should return nil", func() {
					actualSecret, err := mgr.GetWebhookServerCertSecret(context.Background())
					Expect(err).ToNot(HaveOccurred())
					Expect(actualSecret).To(BeNil())
				})
			})
		})
	})
})

func webhookServerCertSecret() *corev1.Secret {
	secret := secretWithNameAndNamespaceManagedByBtpManager(webhookServerCertSecretName, kymaNamespace)
	return secret
}

func caServerCertSecret() *corev1.Secret {
	secret := secretWithNameAndNamespaceManagedByBtpManager(caServerCertSecretName, kymaNamespace)
	return secret
}
