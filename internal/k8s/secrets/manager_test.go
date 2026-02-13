package secrets_test

import (
	"context"

	"github.com/kyma-project/btp-manager/internal/k8s/generic"
	"github.com/kyma-project/btp-manager/internal/k8s/secrets"
	"github.com/kyma-project/btp-manager/internal/manager/moduleresource"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Secrets Manager", func() {
	var mgr *secrets.Manager

	BeforeEach(func() {
		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		mgr = secrets.NewManager(generic.NewObjectManager[*corev1.Secret, *corev1.SecretList](fakeClient))
	})

	Describe("Getting secrets", func() {
		Context("Required sap-btp-manager secret", func() {
			When("the secret exists", func() {
				It("should return the secret", func() {
					expectedSecret := requiredSecret()
					Expect(fakeClient.Create(context.Background(), expectedSecret)).To(Succeed())

					actualSecret, err := mgr.GetRequiredSecret(context.Background())
					Expect(err).ToNot(HaveOccurred())

					Expect(actualSecret).To(Equal(expectedSecret))
				})
			})

			When("the secret does not exist", func() {
				It("should return an error", func() {
					_, err := mgr.GetRequiredSecret(context.Background())

					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("not found"))
				})
			})
		})

		Context("Operand's sap-btp-service-operator secret", func() {
			When("the secret exists in the module's namespace", func() {
				It("should return the secret", func() {
					expectedSecret := sapBtpServiceOperatorSecret()
					Expect(fakeClient.Create(context.Background(), expectedSecret)).To(Succeed())

					actualSecret, err := mgr.GetSapBtpServiceOperatorSecret(context.Background())
					Expect(err).ToNot(HaveOccurred())

					Expect(actualSecret).To(Equal(expectedSecret))
				})
			})

			When("the secret exists in a custom namespace", func() {
				It("should return the secret", func() {
					const expectedNamespace = "test-namespace"
					expectedSecret := sapBtpServiceOperatorSecret()
					expectedSecret.Namespace = expectedNamespace

					Expect(fakeClient.Create(context.Background(), expectedSecret)).To(Succeed())

					actualSecret, err := mgr.GetSapBtpServiceOperatorSecret(context.Background())
					Expect(err).ToNot(HaveOccurred())

					Expect(actualSecret.Name).To(Equal(expectedSecret.Name))
					Expect(actualSecret.Namespace).To(Equal(expectedNamespace))
				})
			})

			When("the secret does not exist", func() {
				It("should return nil", func() {
					actualSecret, err := mgr.GetSapBtpServiceOperatorSecret(context.Background())
					Expect(err).ToNot(HaveOccurred())
					Expect(actualSecret).To(BeNil())
				})
			})
		})

		Context("Operand's sap-btp-operator-clusterid secret", func() {
			When("the secret exists", func() {
				It("should return the secret", func() {
					expectedSecret := sapBtpServiceOperatorClusterIdSecret()
					Expect(fakeClient.Create(context.Background(), expectedSecret)).To(Succeed())

					actualSecret, err := mgr.GetSapBtpServiceOperatorClusterIdSecret(context.Background())
					Expect(err).ToNot(HaveOccurred())

					Expect(actualSecret).To(Equal(expectedSecret))
				})
			})

			When("the secret does not exist", func() {
				It("should return nil", func() {
					actualSecret, err := mgr.GetSapBtpServiceOperatorClusterIdSecret(context.Background())
					Expect(err).ToNot(HaveOccurred())
					Expect(actualSecret).To(BeNil())
				})
			})
		})

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

func sapBtpServiceOperatorClusterIdSecret() *corev1.Secret {
	secret := secretWithNameAndNamespace(sapBtpServiceOperatorClusterIdSecretName, kymaNamespace)
	labels := map[string]string{
		"services.cloud.sap.com/managed-by-sap-btp-operator": "true",
	}
	data := map[string][]byte{
		"INITIAL_CLUSTER_ID": []byte("dGVzdC1jbHVzdGVyLWlk"),
	}
	secret.Labels = labels
	secret.Data = data
	return secret
}

func sapBtpServiceOperatorSecret() *corev1.Secret {
	return credsSecretWithNameAndNamespace(moduleresource.SapBtpServiceOperatorName, kymaNamespace)
}
