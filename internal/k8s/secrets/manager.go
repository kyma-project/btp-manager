package secrets

import (
	"context"

	"github.com/kyma-project/btp-manager/controllers/config"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	caServerCertSecretName      = "ca-server-cert"
	webhookServerCertSecretName = "webhook-server-cert"

	logKeyName      = "name"
	logKeyNamespace = "namespace"
)

type Reader interface {
	Get(ctx context.Context, key client.ObjectKey, object *corev1.Secret, opts ...client.GetOption) error
	List(ctx context.Context, list *corev1.SecretList, opts ...client.ListOption) error
}

type Manager interface {
	// GetCaServerCertSecret retrieves the CA server certificate secret.
	// Returns nil if the secret is not found (optional secret).
	GetCaServerCertSecret(ctx context.Context) (*corev1.Secret, error)

	// GetWebhookServerCertSecret retrieves the webhook server certificate secret.
	// Returns nil if the secret is not found (optional secret).
	GetWebhookServerCertSecret(ctx context.Context) (*corev1.Secret, error)
}

type manager struct {
	Reader
}

func NewManager(secretReader Reader) Manager {
	return &manager{
		Reader: secretReader,
	}
}

func (m *manager) GetCaServerCertSecret(ctx context.Context) (*corev1.Secret, error) {
	return m.getOptionalSecretByNameAndNamespace(ctx, caServerCertSecretName, config.ChartNamespace)
}

func (m *manager) GetWebhookServerCertSecret(ctx context.Context) (*corev1.Secret, error) {
	return m.getOptionalSecretByNameAndNamespace(ctx, webhookServerCertSecretName, config.ChartNamespace)
}

func (m *manager) getOptionalSecretByNameAndNamespace(ctx context.Context, name, namespace string) (*corev1.Secret, error) {
	logger := log.FromContext(ctx)
	logger.Info("Getting secret", logKeyName, name, logKeyNamespace, namespace)

	secret := &corev1.Secret{}
	if err := m.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, secret); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info("Secret not found", logKeyName, name)
			return nil, nil
		}
		logger.Error(err, "Failed to get secret", logKeyName, name)
		return nil, err
	}
	return secret, nil
}
