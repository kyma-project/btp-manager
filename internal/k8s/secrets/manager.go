package secrets

import (
	"context"

	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/k8s/generic"
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
)

type Manager struct {
	*generic.ObjectManager[*corev1.Secret]
}

func NewManager(k8sClient client.Client) *Manager {
	return &Manager{
		ObjectManager: generic.NewObjectManager[*corev1.Secret](k8sClient),
	}
}

func (m *Manager) GetRequiredSecret(ctx context.Context) (*corev1.Secret, error) {
	logger := log.FromContext(ctx)
	logger.Info("Getting the required secret", "name", config.SecretName, "namespace", config.KymaSystemNamespaceName)
	secret, err := m.getSecretByNameAndNamespace(ctx, config.SecretName, config.KymaSystemNamespaceName)
	if err != nil {
		logger.Error(err, "Failed to get the required secret")
		return nil, err
	}
	return secret, nil
}

func (m *Manager) GetCaServerCertSecret(ctx context.Context) (*corev1.Secret, error) {
	logger := log.FromContext(ctx)
	logger.Info("Getting the CA server cert secret", "name", caServerCertSecretName, "namespace", config.ChartNamespace)
	secret, err := m.getSecretByNameAndNamespace(ctx, caServerCertSecretName, config.ChartNamespace)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info("CA server cert secret not found")
			return nil, nil
		}
		logger.Error(err, "Failed to get the CA server cert secret")
		return nil, err
	}
	return secret, nil
}

func (m *Manager) GetWebhookServerCertSecret(ctx context.Context) (*corev1.Secret, error) {
	logger := log.FromContext(ctx)
	logger.Info("Getting the webhook server cert secret", "name", webhookServerCertSecretName, "namespace", config.ChartNamespace)
	secret, err := m.getSecretByNameAndNamespace(ctx, webhookServerCertSecretName, config.ChartNamespace)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info("webhook server cert secret not found")
			return nil, nil
		}
		logger.Error(err, "Failed to get the webhook server cert secret")
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
	logger := log.FromContext(ctx)
	logger.Info("Getting SAP BTP service operator secret")

	labels := map[string]string{
		managedByLabel: operatorName,
	}

	secrets, err := m.getSecretsByLabels(ctx, labels)
	if err != nil {
		return nil, err
	}
	if len(secrets) == 0 {
		logger.Info("No secrets found")
		return nil, nil
	}
	return m.getSecretFromListByName(secrets, sapBtpServiceOperatorSecretName, config.ChartNamespace), nil
}

func (m *Manager) GetSapBtpServiceOperatorClusterIdSecret(ctx context.Context) (*corev1.Secret, error) {
	logger := log.FromContext(ctx)
	logger.Info("Getting SAP BTP service operator cluster ID secret")

	labels := map[string]string{
		managedByBTPOperatorLabel: "true",
	}

	secrets, err := m.getSecretsByLabels(ctx, labels)
	if err != nil {
		return nil, err
	}
	if len(secrets) == 0 {
		logger.Info("No secrets found")
		return nil, nil
	}
	return m.getSecretFromListByName(secrets, sapBtpServiceOperatorClusterIdSecretName, config.ChartNamespace), nil
}

func (m *Manager) getSecretsByLabels(ctx context.Context, labels map[string]string) ([]corev1.Secret, error) {
	logger := log.FromContext(ctx)
	secrets := &corev1.SecretList{}

	logger.Info("Listing secrets by labels", "labels", labels)
	if err := m.List(ctx, secrets, client.MatchingLabels(labels)); err != nil {
		logger.Error(err, "Failed to list secrets")
		return nil, err
	}
	return secrets.Items, nil
}

func (m *Manager) getSecretFromListByName(secrets []corev1.Secret, secretName, optionalSecretNamespace string) *corev1.Secret {
	var secret *corev1.Secret
	for _, s := range secrets {
		if s.Name != secretName {
			continue
		}
		if s.Namespace == optionalSecretNamespace {
			return &s
		}
		if secret == nil {
			secret = &s
		}
	}

	return secret
}
