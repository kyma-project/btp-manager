package secrets_test

import (
	"github.com/kyma-project/btp-manager/internal/k8s/secrets"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Secret Verification Dispatcher", func() {
	var dispatcher *secrets.VerificationDispatcher

	Context("When creating a new dispatcher", func() {
		It("should accept nil verificators map", func() {
			dispatcher := secrets.NewVerificationDispatcher(nil)
			Expect(dispatcher).ToNot(BeNil())
		})

		It("should accept a verificators map", func() {
			verificators := map[string]secrets.Verificator{
				"test-secret": secrets.NewNoopVerificator(),
			}
			dispatcher := secrets.NewVerificationDispatcher(verificators)
			Expect(dispatcher).ToNot(BeNil())
		})
	})

	Context("When verifying secrets", func() {
		const noopSecretName = "noop-secret"

		BeforeEach(func() {
			verificators := map[string]secrets.Verificator{
				requiredSecretName: secrets.NewRequiredSecretVerificator(),
				noopSecretName:     secrets.NewNoopVerificator(),
			}
			dispatcher = secrets.NewVerificationDispatcher(verificators)
		})

		It("should verify a valid secret with registered verificator", func() {
			secret := requiredSecret()

			err := dispatcher.Verify(secret)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should fail verification for invalid secret with registered verificator", func() {
			secret := requiredSecret()
			secret.Data = map[string][]byte{
				"clientid": []byte("test-client-id"),
			}

			err := dispatcher.Verify(secret)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("clientsecret, sm_url, tokenurl, cluster_id not found"))
		})

		It("should accept any secret with noop verificator", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      noopSecretName,
					Namespace: kymaNamespace,
				},
				Data: map[string][]byte{},
			}

			err := dispatcher.Verify(secret)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return error for unregistered secret", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unknown-secret",
					Namespace: kymaNamespace,
				},
				Data: map[string][]byte{},
			}

			err := dispatcher.Verify(secret)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no verificator registered for secret: unknown-secret"))
		})

		It("should return error for nil secret", func() {
			err := dispatcher.Verify(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("secret is nil"))
		})
	})

	Context("When registering verificators dynamically", func() {
		const testSecretName = "test-secret"

		BeforeEach(func() {
			dispatcher = secrets.NewVerificationDispatcher(nil)
		})

		It("should register a new verificator", func() {
			dispatcher.RegisterVerificator(testSecretName, secrets.NewNoopVerificator())

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testSecretName,
					Namespace: kymaNamespace,
				},
				Data: map[string][]byte{},
			}

			err := dispatcher.Verify(secret)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should override existing verificator", func() {
			dispatcher.RegisterVerificator(testSecretName, secrets.NewRequiredSecretVerificator())

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testSecretName,
					Namespace: kymaNamespace,
				},
				Data: map[string][]byte{},
			}

			err := dispatcher.Verify(secret)
			Expect(err).To(HaveOccurred())

			dispatcher.RegisterVerificator(testSecretName, secrets.NewNoopVerificator())

			err = dispatcher.Verify(secret)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

var _ = Describe("Required Secret Verificator", func() {
	var verificator *secrets.RequiredSecretVerificator

	BeforeEach(func() {
		verificator = secrets.NewRequiredSecretVerificator()
	})

	It("should verify a valid secret", func() {
		secret := requiredSecret()

		err := verificator.Verify(secret)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should return error for nil secret", func() {
		err := verificator.Verify(nil)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("secret is nil"))
	})

	It("should return error for missing keys", func() {
		secret := requiredSecret()
		secret.Data = map[string][]byte{
			"clientid": []byte("test-client-id"),
		}

		err := verificator.Verify(secret)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("clientsecret, sm_url, tokenurl, cluster_id not found"))
	})

	It("should return error for empty values", func() {
		secret := requiredSecret()
		secret.Data["clientid"] = []byte("")

		err := verificator.Verify(secret)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing value(s) for clientid key(s)"))
	})
})

var _ = Describe("Noop Verificator", func() {
	var verificator *secrets.NoopVerificator

	BeforeEach(func() {
		verificator = secrets.NewNoopVerificator()
	})

	It("should accept any secret", func() {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "test-namespace",
			},
			Data: map[string][]byte{},
		}

		err := verificator.Verify(secret)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should accept nil secret", func() {
		err := verificator.Verify(nil)
		Expect(err).ToNot(HaveOccurred())
	})
})
