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
		It("should accept nil verifiers map", func() {
			dispatcher := secrets.NewVerificationDispatcher(nil)
			Expect(dispatcher).ToNot(BeNil())
		})

		It("should accept a verifiers map", func() {
			verifiers := map[string]secrets.Verifier{
				"test-secret": secrets.NewNoopVerifier(),
			}
			dispatcher := secrets.NewVerificationDispatcher(verifiers)
			Expect(dispatcher).ToNot(BeNil())
		})
	})

	Context("When verifying secrets", func() {
		const noopSecretName = "noop-secret"

		BeforeEach(func() {
			verifiers := map[string]secrets.Verifier{
				requiredSecretName: secrets.NewRequiredSecretVerifier(),
				noopSecretName:     secrets.NewNoopVerifier(),
			}
			dispatcher = secrets.NewVerificationDispatcher(verifiers)
		})

		It("should verify a valid secret with registered verifier", func() {
			secret := requiredSecret()

			err := dispatcher.Verify(secret)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should fail verification for invalid secret with registered verifier", func() {
			secret := requiredSecret()
			secret.Data = map[string][]byte{
				"clientid": []byte("test-client-id"),
			}

			err := dispatcher.Verify(secret)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("clientsecret, sm_url, tokenurl, cluster_id not found"))
		})

		It("should accept any secret with noop verifier", func() {
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
			Expect(err.Error()).To(ContainSubstring("no verifier registered for secret: unknown-secret"))
		})

		It("should return error for nil secret", func() {
			err := dispatcher.Verify(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("secret is nil"))
		})
	})

	Context("When registering verifiers dynamically", func() {
		const testSecretName = "test-secret"

		BeforeEach(func() {
			dispatcher = secrets.NewVerificationDispatcher(nil)
		})

		It("should register a new verifier", func() {
			dispatcher.RegisterVerifier(testSecretName, secrets.NewNoopVerifier())

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

		It("should override existing verifier", func() {
			dispatcher.RegisterVerifier(testSecretName, secrets.NewRequiredSecretVerifier())

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testSecretName,
					Namespace: kymaNamespace,
				},
				Data: map[string][]byte{},
			}

			err := dispatcher.Verify(secret)
			Expect(err).To(HaveOccurred())

			dispatcher.RegisterVerifier(testSecretName, secrets.NewNoopVerifier())

			err = dispatcher.Verify(secret)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

var _ = Describe("Required Secret Verifier", func() {
	var verifier *secrets.RequiredSecretVerifier

	BeforeEach(func() {
		verifier = secrets.NewRequiredSecretVerifier()
	})

	It("should verify a valid secret", func() {
		secret := requiredSecret()

		err := verifier.Verify(secret)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should return error for nil secret", func() {
		err := verifier.Verify(nil)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("secret is nil"))
	})

	It("should return error for missing keys", func() {
		secret := requiredSecret()
		secret.Data = map[string][]byte{
			"clientid": []byte("test-client-id"),
		}

		err := verifier.Verify(secret)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("clientsecret, sm_url, tokenurl, cluster_id not found"))
	})

	It("should return error for empty values", func() {
		secret := requiredSecret()
		secret.Data["clientid"] = []byte("")

		err := verifier.Verify(secret)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing value(s) for clientid key(s)"))
	})
})

var _ = Describe("Noop Verifier", func() {
	var verifier *secrets.NoopVerifier

	BeforeEach(func() {
		verifier = secrets.NewNoopVerifier()
	})

	It("should accept any secret", func() {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "test-namespace",
			},
			Data: map[string][]byte{},
		}

		err := verifier.Verify(secret)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should accept nil secret", func() {
		err := verifier.Verify(nil)
		Expect(err).ToNot(HaveOccurred())
	})
})
