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
	operatorName                             = "btp-manager"
	sapBtpServiceOperatorSecretName          = "sap-btp-service-operator"
	sapBtpServiceOperatorClusterIdSecretName = "sap-btp-operator-clusterid"
	caServerCertSecretName                   = "ca-server-cert"
	webhookServerCertSecretName              = "webhook-server-cert"
	managedByLabel                           = "app.kubernetes.io/managed-by"
	managedByBTPOperatorLabel                = "services.cloud.sap.com/managed-by-sap-btp-operator"

	logKeyName      = "name"
	logKeyNamespace = "namespace"
	logKeyLabels    = "labels"
)

type Reader interface {
	Get(ctx context.Context, key client.ObjectKey, object *corev1.Secret, opts ...client.GetOption) error
	List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error
}

type Writer interface {
	Create(ctx context.Context, object *corev1.Secret, opts ...client.CreateOption) error
	Apply(ctx context.Context, object *corev1.Secret, opts ...client.PatchOption) error
	Update(ctx context.Context, object *corev1.Secret, opts ...client.UpdateOption) error
	Delete(ctx context.Context, object *corev1.Secret, opts ...client.DeleteOption) error
}

type SecretClient interface {
	Reader
	Writer
}

type Manager struct {
	SecretClient
}

func NewManager(secretClient SecretClient) *Manager {
	return &Manager{
		SecretClient: secretClient,
	}
}

func (m *Manager) GetRequiredSecret(ctx context.Context) (*corev1.Secret, error) {
	logger := log.FromContext(ctx)
	logger.Info("Getting the required secret", logKeyName, config.SecretName, logKeyNamespace, config.KymaSystemNamespaceName)
	secret, err := m.getSecretByNameAndNamespace(ctx, config.SecretName, config.KymaSystemNamespaceName)
	if err != nil {
		logger.Error(err, "Failed to get the required secret")
		return nil, err
	}
	return secret, nil
}

func (m *Manager) GetCaServerCertSecret(ctx context.Context) (*corev1.Secret, error) {
	return m.getOptionalSecretByNameAndNamespace(ctx, caServerCertSecretName, config.ChartNamespace)
}

func (m *Manager) GetWebhookServerCertSecret(ctx context.Context) (*corev1.Secret, error) {
	return m.getOptionalSecretByNameAndNamespace(ctx, webhookServerCertSecretName, config.ChartNamespace)
}

func (m *Manager) getOptionalSecretByNameAndNamespace(ctx context.Context, name, namespace string) (*corev1.Secret, error) {
	logger := log.FromContext(ctx)
	logger.Info("Getting secret", logKeyName, name, logKeyNamespace, namespace)

	secret, err := m.getSecretByNameAndNamespace(ctx, name, namespace)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info("Secret not found", logKeyName, name)
			return nil, nil
		}
		logger.Error(err, "Failed to get secret", logKeyName, name)
		return nil, err
	}
	return secret, nil
}

func (m *Manager) getSecretByNameAndNamespace(ctx context.Context, name, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	if err := m.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, secret); err != nil {
		return nil, err
	}
	return secret, nil
}

func (m *Manager) GetSapBtpServiceOperatorSecret(ctx context.Context) (*corev1.Secret, error) {
	labels := map[string]string{managedByLabel: operatorName}
	return m.getSecretByNameAndLabels(ctx, sapBtpServiceOperatorSecretName, labels)
}

func (m *Manager) GetSapBtpServiceOperatorClusterIdSecret(ctx context.Context) (*corev1.Secret, error) {
	labels := map[string]string{managedByBTPOperatorLabel: "true"}
	return m.getSecretByNameAndLabels(ctx, sapBtpServiceOperatorClusterIdSecretName, labels)
}

func (m *Manager) getSecretByNameAndLabels(ctx context.Context, secretName string, labels map[string]string) (*corev1.Secret, error) {
	logger := log.FromContext(ctx)
	logger.Info("Getting secret by name and labels", logKeyName, secretName, logKeyLabels, labels)

	secrets, err := m.getSecretsByLabels(ctx, labels)
	if err != nil {
		return nil, err
	}
	if len(secrets) == 0 {
		logger.Info("No secrets found")
		return nil, nil
	}
	return m.findSecretInList(secrets, secretName, config.ChartNamespace), nil
}

func (m *Manager) getSecretsByLabels(ctx context.Context, labels map[string]string) ([]corev1.Secret, error) {
	logger := log.FromContext(ctx)
	secrets := &corev1.SecretList{}

	logger.Info("Listing secrets by labels", logKeyLabels, labels)
	if err := m.List(ctx, secrets, client.MatchingLabels(labels)); err != nil {
		logger.Error(err, "Failed to list secrets")
		return nil, err
	}
	return secrets.Items, nil
}

func (m *Manager) findSecretInList(secrets []corev1.Secret, secretName, preferredNamespace string) *corev1.Secret {
	var fallbackSecret *corev1.Secret
	for _, s := range secrets {
		if s.Name != secretName {
			continue
		}
		if s.Namespace == preferredNamespace {
			return &s
		}
		if fallbackSecret == nil {
			fallbackSecret = &s
		}
	}

	return fallbackSecret
}
