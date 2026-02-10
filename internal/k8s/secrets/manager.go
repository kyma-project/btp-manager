package secrets

import (
	"context"

	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/k8s/generic"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	managedByLabel                  = "app.kubernetes.io/managed-by"
	operatorName                    = "btp-manager"
	sapBtpServiceOperatorSecretName = "sap-btp-service-operator"
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
	secrets, err := m.getManagedSecrets(ctx)
	if err != nil {
		return nil, err
	}
	if len(secrets) == 0 {
		logger.Info("No managed secrets found")
		return nil, nil
	}
	return m.getSapBtpServiceOperatorSecretFromList(secrets), nil

}

func (m *Manager) getManagedSecrets(ctx context.Context) ([]corev1.Secret, error) {
	logger := log.FromContext(ctx)
	secrets := &corev1.SecretList{}

	logger.Info("Listing managed secrets")
	if err := m.List(ctx, secrets, client.MatchingLabels{managedByLabel: operatorName}); err != nil {
		logger.Error(err, "Failed to list managed secrets")
		return nil, err
	}
	return secrets.Items, nil
}

func (m *Manager) getSapBtpServiceOperatorSecretFromList(secrets []corev1.Secret) *corev1.Secret {
	var sapBtpServiceOperatorSecret *corev1.Secret
	for _, secret := range secrets {
		if secret.Name != sapBtpServiceOperatorSecretName {
			continue
		}
		if secret.Namespace == config.ChartNamespace {
			return &secret
		}
		if sapBtpServiceOperatorSecret == nil {
			sapBtpServiceOperatorSecret = &secret
		}
	}

	return sapBtpServiceOperatorSecret
}
