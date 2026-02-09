package secrets

import (
	"context"

	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/k8s/generic"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
	secret := &corev1.Secret{}

	logger.Info("Getting the required secret")
	if err := m.Get(ctx, client.ObjectKey{Name: config.SecretName, Namespace: config.KymaSystemNamespaceName}, secret); err != nil {
		logger.Error(err, "Failed to get the required secret")
		return nil, err
	}
	return secret, nil
}
