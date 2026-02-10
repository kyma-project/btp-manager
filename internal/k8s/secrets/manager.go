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
	logger.Info("Getting the required secret")
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
	secrets, err := m.getManagedSecrets(ctx)
	if err != nil {
		return nil, err
	}
	if secrets.Size() == 0 {
		return nil, nil
	}
	return m.getSapBtpServiceOperatorSecretFromList(secrets), nil

}

func (m *Manager) getManagedSecrets(ctx context.Context) (*corev1.SecretList, error) {
	logger := log.FromContext(ctx)
	secrets := &corev1.SecretList{}

	logger.Info("Getting managed secrets")
	if err := m.List(ctx, secrets, client.MatchingLabels{managedByLabel: operatorName}); err != nil {
		logger.Error(err, "Failed to get managed secrets")
		return nil, err
	}
	return secrets, nil
}

func (m *Manager) getSapBtpServiceOperatorSecretFromList(secrets *corev1.SecretList) *corev1.Secret {
	var sapBtpServiceOperatorSecret *corev1.Secret
	for _, secret := range secrets.Items {
		if isSecretInModuleNamespace(&secret) {
			sapBtpServiceOperatorSecret = &secret
			break
		} else if isSecretInCustomNamespace(&secret) {
			sapBtpServiceOperatorSecret = &secret
		}
	}
	return sapBtpServiceOperatorSecret
}

func isSecretInModuleNamespace(secret *corev1.Secret) bool {
	return secret.Name == sapBtpServiceOperatorSecretName && secret.Namespace == config.ChartNamespace
}

func isSecretInCustomNamespace(secret *corev1.Secret) bool {
	return secret.Name == sapBtpServiceOperatorSecretName
}
