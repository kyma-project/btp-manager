package certificate

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"time"

	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/certs"
	"github.com/kyma-project/btp-manager/internal/k8s/secrets"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	caCertSecretName      = "ca-server-cert"
	webhookCertSecretName = "webhook-server-cert"

	caCertSecretCertField      = "ca.crt"
	caCertSecretKeyField       = "ca.key"
	webhookCertSecretCertField = "tls.crt"
	webhookCertSecretKeyField  = "tls.key"

	MutatingWebhookConfigurationKind   = "MutatingWebhookConfiguration"
	ValidatingWebhookConfigurationKind = "ValidatingWebhookConfiguration"

	secretKind   = "Secret"
	managedByKey = "app.kubernetes.io/managed-by"
	operatorName = "btp-manager"
)

type CertificateSignError struct {
	message string
}

func NewCertificateSignError(message string) CertificateSignError {
	return CertificateSignError{message: message}
}

func (e CertificateSignError) Error() string {
	return e.message
}

type CertificateManager interface {
	PrepareAdmissionWebhooks(ctx context.Context, webhookResources []*unstructured.Unstructured) ([]*unstructured.Unstructured, error)
}

// WebhookMetrics is the metrics sink consumed by the certificate manager.
type WebhookMetrics interface {
	IncrementCertsRegenerationCounter()
}

// IsAdmissionWebhook reports whether the given resource kind is a webhook
// configuration managed by the certificate manager.
func IsAdmissionWebhook(kind string) bool {
	return kind == MutatingWebhookConfigurationKind || kind == ValidatingWebhookConfigurationKind
}

type Manager struct {
	secretsManager secrets.Manager
	webhookMetrics WebhookMetrics
}

func NewManager(secretsManager secrets.Manager, webhookMetrics WebhookMetrics) *Manager {
	return &Manager{
		secretsManager: secretsManager,
		webhookMetrics: webhookMetrics,
	}
}

var _ CertificateManager = (*Manager)(nil)

func (m *Manager) PrepareAdmissionWebhooks(ctx context.Context, webhookResources []*unstructured.Unstructured) ([]*unstructured.Unstructured, error) {
	logger := log.FromContext(ctx)
	logger.Info("preparing admission webhooks")

	logger.Info("checking CA certificate")
	caCertSecret, err := m.secretsManager.GetCaServerCertSecret(ctx)
	if err != nil {
		return nil, err
	}
	if caCertSecret == nil {
		logger.Info("CA cert secret does not exist")
		return m.regenerateCertificates(ctx, webhookResources)
	}
	if err := m.validateCert(caCertSecret); err != nil {
		logger.Info(fmt.Sprintf("CA cert is not valid: %s", err))
		return m.regenerateCertificates(ctx, webhookResources)
	}

	caBundle := caCertSecret.Data[caCertSecretCertField]

	logger.Info("checking webhook certificate")
	webhookCertSecret, err := m.secretsManager.GetWebhookServerCertSecret(ctx)
	if err != nil {
		return nil, err
	}
	if webhookCertSecret == nil {
		logger.Info("webhook cert secret does not exist")
		return m.regenerateWebhookCertificate(ctx, webhookResources, caCertSecret.Data)
	}
	if err := m.validateWebhookCert(webhookCertSecret, caBundle); err != nil {
		logger.Info(fmt.Sprintf("webhook cert is not valid: %s", err))
		var certSignErr CertificateSignError
		if errors.As(err, &certSignErr) {
			return m.regenerateCertificates(ctx, webhookResources)
		}
		return m.regenerateWebhookCertificate(ctx, webhookResources, caCertSecret.Data)
	}

	logger.Info("certificates for admission webhooks are valid")
	return m.prepareWebhooksManifests(ctx, webhookResources, caBundle)
}

func (m *Manager) regenerateCertificates(ctx context.Context, webhookResources []*unstructured.Unstructured) ([]*unstructured.Unstructured, error) {
	logger := log.FromContext(ctx)
	logger.Info("regenerating CA and webhook certificates")

	caCertificate, caPrivateKey, err := m.generateSelfSignedCert(ctx)
	if err != nil {
		return nil, fmt.Errorf("while generating CA self signed cert: %w", err)
	}

	caSecret, err := m.buildCertificateSecret(caCertSecretName, caCertificate, caPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("while building secret with regenerated CA self signed cert: %w", err)
	}

	webhookCertificate, webhookPrivateKey, err := m.generateSignedCert(ctx, caCertificate, caPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("while generating webhook signed cert: %w", err)
	}

	webhookSecret, err := m.buildCertificateSecret(webhookCertSecretName, webhookCertificate, webhookPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("while building regenerated webhook signed cert secret: %w", err)
	}

	preparedWebhooks, err := m.prepareWebhooksManifests(ctx, webhookResources, caCertificate)
	if err != nil {
		return nil, fmt.Errorf("while preparing webhooks manifests: %w", err)
	}

	logger.Info("certificates regeneration succeeded")
	m.webhookMetrics.IncrementCertsRegenerationCounter()
	return append([]*unstructured.Unstructured{caSecret, webhookSecret}, preparedWebhooks...), nil
}

func (m *Manager) regenerateWebhookCertificate(ctx context.Context, webhookResources []*unstructured.Unstructured, caCertSecretData map[string][]byte) ([]*unstructured.Unstructured, error) {
	logger := log.FromContext(ctx)
	logger.Info("regenerating webhook certificate")

	webhookCertificate, webhookPrivateKey, err := m.generateSignedCert(ctx, caCertSecretData[caCertSecretCertField], caCertSecretData[caCertSecretKeyField])
	if err != nil {
		return nil, fmt.Errorf("while regenerating webhook signed cert: %w", err)
	}

	webhookSecret, err := m.buildCertificateSecret(webhookCertSecretName, webhookCertificate, webhookPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("while building regenerated webhook signed cert secret: %w", err)
	}

	preparedWebhooks, err := m.prepareWebhooksManifests(ctx, webhookResources, caCertSecretData[caCertSecretCertField])
	if err != nil {
		return nil, err
	}

	logger.Info("webhook certificate regeneration succeeded")
	m.webhookMetrics.IncrementCertsRegenerationCounter()
	return append([]*unstructured.Unstructured{webhookSecret}, preparedWebhooks...), nil
}

func (m *Manager) generateSelfSignedCert(ctx context.Context) ([]byte, []byte, error) {
	logger := log.FromContext(ctx)
	logger.Info("generating self signed cert")
	caCertificate, caPrivateKey, err := certs.GenerateSelfSignedCertificate(time.Now().UTC().Add(config.CaCertificateExpiration))
	if err != nil {
		return nil, nil, fmt.Errorf("while generating self signed cert: %w", err)
	}
	return caCertificate, caPrivateKey, nil
}

func (m *Manager) generateSignedCert(ctx context.Context, caCert, caPrivateKey []byte) ([]byte, []byte, error) {
	logger := log.FromContext(ctx)
	logger.Info("generating webhook signed cert")
	webhookCertificate, webhookPrivateKey, err := certs.GenerateSignedCertificate(time.Now().UTC().Add(config.WebhookCertificateExpiration), caCert, caPrivateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("while generating webhook signed cert: %w", err)
	}
	return webhookCertificate, webhookPrivateKey, nil
}

func (m *Manager) buildCertificateSecret(secretName string, certificate, privateKey []byte) (*unstructured.Unstructured, error) {
	certFieldName, err := certFieldFromSecretBySecretName(secretName)
	if err != nil {
		return nil, err
	}
	privateKeyFieldName, err := privateKeyFieldFromSecretBySecretName(secretName)
	if err != nil {
		return nil, err
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{Kind: secretKind, APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: config.ChartNamespace,
			Labels:    map[string]string{managedByKey: operatorName},
		},
		Data: map[string][]byte{
			certFieldName:       certificate,
			privateKeyFieldName: privateKey,
		},
	}

	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(secret)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: unstructuredObj}, nil
}

func (m *Manager) prepareWebhooksManifests(ctx context.Context, webhookResources []*unstructured.Unstructured, caBundle []byte) ([]*unstructured.Unstructured, error) {
	logger := log.FromContext(ctx)
	logger.Info("preparing webhooks manifests")

	prepared := make([]*unstructured.Unstructured, 0, len(webhookResources))
	for _, resource := range webhookResources {
		webhookManifest, err := prepareWebhookManifest(ctx, resource, caBundle)
		if err != nil {
			return nil, err
		}
		prepared = append(prepared, webhookManifest)
	}

	logger.Info("webhooks manifests have been prepared successfully")
	return prepared, nil
}

func prepareWebhookManifest(ctx context.Context, webhookManifest *unstructured.Unstructured, caBundle []byte) (*unstructured.Unstructured, error) {
	const (
		webhooksKey     = "webhooks"
		clientConfigKey = "clientConfig"
		caBundleKey     = "caBundle"
	)
	webhookManifestCopy := webhookManifest.DeepCopy()

	logger := log.FromContext(ctx)
	logger.Info(fmt.Sprintf("setting CA bundle in %s %s", webhookManifestCopy.GetName(), webhookManifestCopy.GetKind()))

	webhooks, exists, err := unstructured.NestedSlice(webhookManifestCopy.Object, webhooksKey)
	if err != nil {
		return nil, fmt.Errorf("while getting webhooks array from the webhook manifest: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("webhooks array does not exist in the webhook manifest")
	}
	webhookManifestCopy.SetManagedFields(nil)

	for i, webhook := range webhooks {
		genericWebhook, ok := webhook.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("while casting webhook object to map[string]interface{}")
		}
		genericClientConfig, ok := genericWebhook[clientConfigKey].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("while casting webhook.clientConfig object to map[string]interface{}")
		}
		genericClientConfig[caBundleKey] = caBundle
		genericWebhook[clientConfigKey] = genericClientConfig
		webhooks[i] = genericWebhook
	}
	webhookManifestCopy.Object[webhooksKey] = webhooks

	return webhookManifestCopy, nil
}

func (m *Manager) validateCert(secret *corev1.Secret) error {
	certFieldName, err := certFieldFromSecretBySecretName(secret.GetName())
	if err != nil {
		return err
	}
	encodedCert, err := getSecretDataValueByKey(certFieldName, secret.Data)
	if err != nil {
		return err
	}
	privateKeyFieldName, err := privateKeyFieldFromSecretBySecretName(secret.GetName())
	if err != nil {
		return err
	}
	if _, err = getSecretDataValueByKey(privateKeyFieldName, secret.Data); err != nil {
		return err
	}
	block, err := certs.DecodeCertificate(encodedCert)
	if err != nil {
		return err
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return err
	}
	if certs.CertificateExpires(cert, config.ExpirationBoundary) {
		return fmt.Errorf("CA cert expires soon")
	}
	return nil
}

func (m *Manager) validateWebhookCert(webhookCertSecret *corev1.Secret, caCert []byte) error {
	if err := m.validateCert(webhookCertSecret); err != nil {
		return err
	}
	return verifyCASign(caCert, webhookCertSecret.Data[webhookCertSecretCertField])
}

func verifyCASign(caCert, signedCert []byte) error {
	ok, err := certs.VerifyIfLeafIsSignedByGivenCA(caCert, signedCert)
	if err != nil {
		return NewCertificateSignError(err.Error())
	}
	if !ok {
		return NewCertificateSignError("certificate is not signed by the provided CA")
	}
	return nil
}

func getSecretDataValueByKey(key string, data map[string][]byte) ([]byte, error) {
	value, ok := data[key]
	if !ok {
		return nil, fmt.Errorf("missing key: %s", key)
	}
	if len(value) == 0 {
		return nil, fmt.Errorf("empty value for key: %s", key)
	}
	return value, nil
}

func certFieldFromSecretBySecretName(secretName string) (string, error) {
	switch secretName {
	case caCertSecretName:
		return caCertSecretCertField, nil
	case webhookCertSecretName:
		return webhookCertSecretCertField, nil
	}
	return "", fmt.Errorf("unknown secret %q - cert field undefined", secretName)
}

func privateKeyFieldFromSecretBySecretName(secretName string) (string, error) {
	switch secretName {
	case caCertSecretName:
		return caCertSecretKeyField, nil
	case webhookCertSecretName:
		return webhookCertSecretKeyField, nil
	}
	return "", fmt.Errorf("unknown secret %q - private key field undefined", secretName)
}
