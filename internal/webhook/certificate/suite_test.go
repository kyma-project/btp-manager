package certificate_test

import (
	"context"
	"testing"
	"time"

	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/certs"
	"github.com/kyma-project/btp-manager/internal/webhook/certificate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	kymaNamespace = "kyma-system"

	caCertField      = "ca.crt"
	caKeyField       = "ca.key"
	webhookCertField = "tls.crt"
	webhookKeyField  = "tls.key"
)

var (
	validCACert    []byte
	validCAKey     []byte
	expiringCACert []byte
	expiringCAKey  []byte

	validWebhookCert       []byte
	validWebhookKey        []byte
	expiringWebhookCert    []byte
	expiringWebhookKey     []byte
	wrongSignedWebhookCert []byte
	wrongSignedWebhookKey  []byte
)

func TestCertificateManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Certificate Manager Suite")
}

var _ = BeforeSuite(func() {
	// Use 1024-bit keys (minimum accepted by Go's crypto/rsa) so cert generation is fast in tests.
	certs.SetRsaKeyBits(1024)

	var err error

	validCACert, validCAKey, err = certs.GenerateSelfSignedCertificate(
		time.Now().UTC().Add(config.CaCertificateExpiration),
	)
	Expect(err).NotTo(HaveOccurred())

	// Expiry within ExpirationBoundary (1 week) triggers regeneration.
	expiringCACert, expiringCAKey, err = certs.GenerateSelfSignedCertificate(
		time.Now().UTC().Add(time.Hour),
	)
	Expect(err).NotTo(HaveOccurred())

	validWebhookCert, validWebhookKey, err = certs.GenerateSignedCertificate(
		time.Now().UTC().Add(config.WebhookCertificateExpiration),
		validCACert, validCAKey,
	)
	Expect(err).NotTo(HaveOccurred())

	expiringWebhookCert, expiringWebhookKey, err = certs.GenerateSignedCertificate(
		time.Now().UTC().Add(time.Hour),
		validCACert, validCAKey,
	)
	Expect(err).NotTo(HaveOccurred())

	// Cert signed by a completely different CA — triggers CertificateSignError.
	wrongCACert, wrongCAKey, err := certs.GenerateSelfSignedCertificate(
		time.Now().UTC().Add(config.CaCertificateExpiration),
	)
	Expect(err).NotTo(HaveOccurred())
	wrongSignedWebhookCert, wrongSignedWebhookKey, err = certs.GenerateSignedCertificate(
		time.Now().UTC().Add(config.WebhookCertificateExpiration),
		wrongCACert, wrongCAKey,
	)
	Expect(err).NotTo(HaveOccurred())
})

// fakeSecretsManager is a test double for secrets.Manager. Only the two cert
// getters matter for the certificate manager; the rest return nil.
type fakeSecretsManager struct {
	caCertSecret      *corev1.Secret
	caCertSecretErr   error
	webhookCertSecret *corev1.Secret
	webhookCertErr    error
}

func (f *fakeSecretsManager) GetRequiredSecret(_ context.Context) (*corev1.Secret, error) {
	return nil, nil
}
func (f *fakeSecretsManager) GetCaServerCertSecret(_ context.Context) (*corev1.Secret, error) {
	return f.caCertSecret, f.caCertSecretErr
}
func (f *fakeSecretsManager) GetWebhookServerCertSecret(_ context.Context) (*corev1.Secret, error) {
	return f.webhookCertSecret, f.webhookCertErr
}
func (f *fakeSecretsManager) GetSapBtpServiceOperatorSecret(_ context.Context) (*corev1.Secret, error) {
	return nil, nil
}
func (f *fakeSecretsManager) GetSapBtpServiceOperatorClusterIdSecret(_ context.Context) (*corev1.Secret, error) {
	return nil, nil
}

func caSecret(cert, key []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: certificate.CaCertSecretName, Namespace: kymaNamespace},
		Data:       map[string][]byte{caCertField: cert, caKeyField: key},
	}
}

func webhookSecret(cert, key []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: certificate.WebhookCertSecretName, Namespace: kymaNamespace},
		Data:       map[string][]byte{webhookCertField: cert, webhookKeyField: key},
	}
}

func mutatingWebhookConfig(name string) *unstructured.Unstructured {
	return webhookConfigUnstructured("MutatingWebhookConfiguration", name)
}

func validatingWebhookConfig(name string) *unstructured.Unstructured {
	return webhookConfigUnstructured("ValidatingWebhookConfiguration", name)
}

func webhookConfigUnstructured(kind, name string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetKind(kind)
	u.SetAPIVersion("admissionregistration.k8s.io/v1")
	u.SetName(name)
	_ = unstructured.SetNestedSlice(u.Object, []interface{}{
		map[string]interface{}{
			"name":         "test-webhook",
			"clientConfig": map[string]interface{}{},
		},
	}, "webhooks")
	return u
}

// fakeWebhookMetrics is a minimal test double for certificate.WebhookMetrics.
type fakeWebhookMetrics struct {
	counter int
}

func (f *fakeWebhookMetrics) IncrementCertsRegenerationCounter() {
	f.counter++
}

func webhookResourceIn(resources []*unstructured.Unstructured) *unstructured.Unstructured {
	for _, r := range resources {
		if certificate.IsAdmissionWebhook(r.GetKind()) {
			return r
		}
	}
	return nil
}

func secretNamesIn(resources []*unstructured.Unstructured) []string {
	var names []string
	for _, r := range resources {
		if r.GetKind() == "Secret" {
			names = append(names, r.GetName())
		}
	}
	return names
}

func caBundleIn(resource *unstructured.Unstructured) []byte {
	// NestedFieldNoCopy is used instead of NestedSlice because the CA bundle is
	// stored as []byte inside the nested map, which NestedSlice cannot deep-copy.
	webhooksRaw, exists, _ := unstructured.NestedFieldNoCopy(resource.Object, "webhooks")
	if !exists {
		return nil
	}
	webhooks, ok := webhooksRaw.([]interface{})
	if !ok || len(webhooks) == 0 {
		return nil
	}
	webhook, ok := webhooks[0].(map[string]interface{})
	if !ok {
		return nil
	}
	clientConfig, ok := webhook["clientConfig"].(map[string]interface{})
	if !ok {
		return nil
	}
	raw, ok := clientConfig["caBundle"].([]byte)
	if !ok {
		return nil
	}
	return raw
}
