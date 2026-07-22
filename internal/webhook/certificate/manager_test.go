package certificate_test

import (
	"context"
	"errors"

	"github.com/kyma-project/btp-manager/internal/webhook/certificate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("Certificate Manager", func() {
	var (
		mgr        *certificate.Manager
		secretsMgr *fakeSecretsManager
		metrics    *fakeWebhookMetrics
		ctx        context.Context
	)

	BeforeEach(func() {
		secretsMgr = &fakeSecretsManager{}
		metrics = &fakeWebhookMetrics{}
		mgr = certificate.NewManager(secretsMgr, metrics)
		ctx = context.Background()
	})

	Describe("PrepareAdmissionWebhooks", func() {
		Context("when CA cert secret does not exist", func() {
			BeforeEach(func() {
				secretsMgr.caCertSecret = nil
			})

			It("returns both the CA and webhook cert secrets", func() {
				result, err := mgr.PrepareAdmissionWebhooks(ctx, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(secretNamesIn(result)).To(ConsistOf(caCertSecretName, webhookCertSecretName))
			})

			It("increments the regeneration counter", func() {
				_, err := mgr.PrepareAdmissionWebhooks(ctx, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(metrics.counter).To(Equal(1))
			})

			It("returns the webhook config with CA bundle injected", func() {
				result, err := mgr.PrepareAdmissionWebhooks(ctx, []*unstructured.Unstructured{
					mutatingWebhookConfig("test-mutating"),
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(caBundleIn(webhookResourceIn(result))).NotTo(BeEmpty())
			})
		})

		Context("when CA cert secret exists but the certificate is expiring soon", func() {
			BeforeEach(func() {
				secretsMgr.caCertSecret = caSecret(expiringCACert, expiringCAKey)
			})

			It("returns both the CA and webhook cert secrets", func() {
				result, err := mgr.PrepareAdmissionWebhooks(ctx, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(secretNamesIn(result)).To(ConsistOf(caCertSecretName, webhookCertSecretName))
			})

			It("increments the regeneration counter", func() {
				_, err := mgr.PrepareAdmissionWebhooks(ctx, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(metrics.counter).To(Equal(1))
			})
		})

		Context("when CA cert is valid but webhook cert secret does not exist", func() {
			BeforeEach(func() {
				secretsMgr.caCertSecret = caSecret(validCACert, validCAKey)
				secretsMgr.webhookCertSecret = nil
			})

			It("returns only the webhook cert secret", func() {
				result, err := mgr.PrepareAdmissionWebhooks(ctx, nil)

				Expect(err).NotTo(HaveOccurred())
				names := secretNamesIn(result)
				Expect(names).To(ConsistOf(webhookCertSecretName))
				Expect(names).NotTo(ContainElement(caCertSecretName))
			})

			It("increments the regeneration counter", func() {
				_, err := mgr.PrepareAdmissionWebhooks(ctx, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(metrics.counter).To(Equal(1))
			})

			It("returns the webhook config with the existing CA bundle injected", func() {
				result, err := mgr.PrepareAdmissionWebhooks(ctx, []*unstructured.Unstructured{
					mutatingWebhookConfig("test-mutating"),
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(caBundleIn(webhookResourceIn(result))).NotTo(BeEmpty())
			})
		})

		Context("when CA cert is valid but webhook cert is expiring soon", func() {
			BeforeEach(func() {
				secretsMgr.caCertSecret = caSecret(validCACert, validCAKey)
				secretsMgr.webhookCertSecret = webhookSecret(expiringWebhookCert, expiringWebhookKey)
			})

			It("returns only the webhook cert secret", func() {
				result, err := mgr.PrepareAdmissionWebhooks(ctx, nil)

				Expect(err).NotTo(HaveOccurred())
				names := secretNamesIn(result)
				Expect(names).To(ConsistOf(webhookCertSecretName))
				Expect(names).NotTo(ContainElement(caCertSecretName))
			})

			It("increments the regeneration counter", func() {
				_, err := mgr.PrepareAdmissionWebhooks(ctx, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(metrics.counter).To(Equal(1))
			})
		})

		Context("when CA cert is valid but webhook cert was signed by a different CA", func() {
			BeforeEach(func() {
				secretsMgr.caCertSecret = caSecret(validCACert, validCAKey)
				secretsMgr.webhookCertSecret = webhookSecret(wrongSignedWebhookCert, wrongSignedWebhookKey)
			})

			It("returns both the CA and webhook cert secrets", func() {
				result, err := mgr.PrepareAdmissionWebhooks(ctx, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(secretNamesIn(result)).To(ConsistOf(caCertSecretName, webhookCertSecretName))
			})

			It("increments the regeneration counter", func() {
				_, err := mgr.PrepareAdmissionWebhooks(ctx, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(metrics.counter).To(Equal(1))
			})
		})

		Context("when both CA and webhook certs are valid", func() {
			BeforeEach(func() {
				secretsMgr.caCertSecret = caSecret(validCACert, validCAKey)
				secretsMgr.webhookCertSecret = webhookSecret(validWebhookCert, validWebhookKey)
			})

			It("returns no cert secrets", func() {
				result, err := mgr.PrepareAdmissionWebhooks(ctx, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(secretNamesIn(result)).To(BeEmpty())
			})

			It("does not increment the regeneration counter", func() {
				_, err := mgr.PrepareAdmissionWebhooks(ctx, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(metrics.counter).To(Equal(0))
			})

			It("returns a MutatingWebhookConfiguration with the CA bundle injected", func() {
				result, err := mgr.PrepareAdmissionWebhooks(ctx, []*unstructured.Unstructured{
					mutatingWebhookConfig("test-mutating"),
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(caBundleIn(webhookResourceIn(result))).To(Equal(validCACert))
			})

			It("returns a ValidatingWebhookConfiguration with the CA bundle injected", func() {
				result, err := mgr.PrepareAdmissionWebhooks(ctx, []*unstructured.Unstructured{
					validatingWebhookConfig("test-validating"),
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(caBundleIn(webhookResourceIn(result))).To(Equal(validCACert))
			})
		})

		Context("error paths", func() {
			It("returns an error when fetching the CA cert secret fails", func() {
				secretsMgr.caCertSecretErr = errors.New("api server unavailable")

				_, err := mgr.PrepareAdmissionWebhooks(ctx, nil)

				Expect(err).To(MatchError(ContainSubstring("api server unavailable")))
			})

			It("returns an error when fetching the webhook cert secret fails", func() {
				secretsMgr.caCertSecret = caSecret(validCACert, validCAKey)
				secretsMgr.webhookCertErr = errors.New("api server unavailable")

				_, err := mgr.PrepareAdmissionWebhooks(ctx, nil)

				Expect(err).To(MatchError(ContainSubstring("api server unavailable")))
			})
		})
	})

	Describe("IsWebhookCertSignedBySelfSignedCA", func() {
		Context("when CA secret is missing", func() {
			BeforeEach(func() {
				secretsMgr.caCertSecret = nil
			})

			It("returns an error", func() {
				_, err := mgr.IsWebhookCertSignedBySelfSignedCA(ctx)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("ca-server-cert"))
			})
		})

		Context("when fetching the CA secret fails", func() {
			BeforeEach(func() {
				secretsMgr.caCertSecretErr = errors.New("api error")
			})

			It("returns the error", func() {
				_, err := mgr.IsWebhookCertSignedBySelfSignedCA(ctx)
				Expect(err).To(MatchError("api error"))
			})
		})

		Context("when webhook secret is missing", func() {
			BeforeEach(func() {
				secretsMgr.caCertSecret = caSecret(validCACert, validCAKey)
				secretsMgr.webhookCertSecret = nil
			})

			It("returns an error", func() {
				_, err := mgr.IsWebhookCertSignedBySelfSignedCA(ctx)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("webhook-server-cert"))
			})
		})

		Context("when fetching the webhook secret fails", func() {
			BeforeEach(func() {
				secretsMgr.caCertSecret = caSecret(validCACert, validCAKey)
				secretsMgr.webhookCertErr = errors.New("api error")
			})

			It("returns the error", func() {
				_, err := mgr.IsWebhookCertSignedBySelfSignedCA(ctx)
				Expect(err).To(MatchError("api error"))
			})
		})

		Context("when webhook cert is signed by the CA", func() {
			BeforeEach(func() {
				secretsMgr.caCertSecret = caSecret(validCACert, validCAKey)
				secretsMgr.webhookCertSecret = webhookSecret(validWebhookCert, validWebhookKey)
			})

			It("returns true", func() {
				ok, err := mgr.IsWebhookCertSignedBySelfSignedCA(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})

		Context("when webhook cert is signed by a different CA", func() {
			BeforeEach(func() {
				secretsMgr.caCertSecret = caSecret(validCACert, validCAKey)
				secretsMgr.webhookCertSecret = webhookSecret(wrongSignedWebhookCert, wrongSignedWebhookKey)
			})

			It("returns false and an error", func() {
				ok, err := mgr.IsWebhookCertSignedBySelfSignedCA(ctx)
				Expect(err).To(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})

	Describe("GetSecretData", func() {
		Context("when requesting CA secret data", func() {
			BeforeEach(func() {
				secretsMgr.caCertSecret = caSecret(validCACert, validCAKey)
			})

			It("returns the secret data", func() {
				data, err := mgr.GetSecretData(ctx, caCertSecretName)
				Expect(err).NotTo(HaveOccurred())
				Expect(data[caCertField]).To(Equal(validCACert))
				Expect(data[caKeyField]).To(Equal(validCAKey))
			})
		})

		Context("when CA secret is missing", func() {
			BeforeEach(func() {
				secretsMgr.caCertSecret = nil
			})

			It("returns an error", func() {
				_, err := mgr.GetSecretData(ctx, caCertSecretName)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("ca-server-cert"))
			})
		})

		Context("when requesting webhook secret data", func() {
			BeforeEach(func() {
				secretsMgr.webhookCertSecret = webhookSecret(validWebhookCert, validWebhookKey)
			})

			It("returns the secret data", func() {
				data, err := mgr.GetSecretData(ctx, webhookCertSecretName)
				Expect(err).NotTo(HaveOccurred())
				Expect(data[webhookCertField]).To(Equal(validWebhookCert))
				Expect(data[webhookKeyField]).To(Equal(validWebhookKey))
			})
		})

		Context("when webhook secret is missing", func() {
			BeforeEach(func() {
				secretsMgr.webhookCertSecret = nil
			})

			It("returns an error", func() {
				_, err := mgr.GetSecretData(ctx, webhookCertSecretName)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("webhook-server-cert"))
			})
		})

		Context("when fetching CA secret fails", func() {
			BeforeEach(func() {
				secretsMgr.caCertSecretErr = errors.New("api error")
			})

			It("returns the error", func() {
				_, err := mgr.GetSecretData(ctx, caCertSecretName)
				Expect(err).To(MatchError("api error"))
			})
		})

		Context("when an unknown secret name is requested", func() {
			It("returns an error", func() {
				_, err := mgr.GetSecretData(ctx, "unknown-secret")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unknown secret"))
			})
		})
	})

	Describe("PartitionWebhooks", func() {
		It("splits webhook configurations from other resources", func() {
			mutating := mutatingWebhookConfig("test-mutating")
			validating := validatingWebhookConfig("test-validating")
			other := &unstructured.Unstructured{}
			other.SetKind("ConfigMap")
			other.SetName("test-config")

			webhooks, rest := certificate.PartitionWebhooks([]*unstructured.Unstructured{mutating, other, validating})

			Expect(webhooks).To(ConsistOf(mutating, validating))
			Expect(rest).To(ConsistOf(other))
		})

		It("returns nil slices when there are no resources", func() {
			webhooks, rest := certificate.PartitionWebhooks(nil)

			Expect(webhooks).To(BeNil())
			Expect(rest).To(BeNil())
		})

		It("places all resources in rest when none are webhooks", func() {
			cm := &unstructured.Unstructured{}
			cm.SetKind("ConfigMap")
			cm.SetName("test-config")

			webhooks, rest := certificate.PartitionWebhooks([]*unstructured.Unstructured{cm})

			Expect(webhooks).To(BeNil())
			Expect(rest).To(ConsistOf(cm))
		})
	})
})
