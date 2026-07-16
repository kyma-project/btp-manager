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
