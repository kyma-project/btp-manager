package servicemanager

import (
	"context"
	"log/slog"

	clusterobject "github.com/kyma-project/btp-manager/internal/cluster-object"
	corev1 "k8s.io/api/core/v1"
)

const componentName = "ServiceManagerClient"

type Client struct {
	ctx            context.Context
	logger         *slog.Logger
	secretProvider clusterobject.NamespacedProvider[*corev1.Secret]
}

func NewClient(ctx context.Context, logger *slog.Logger, secretProvider clusterobject.NamespacedProvider[*corev1.Secret]) *Client {
	return &Client{
		ctx:            ctx,
		logger:         logger.With("component", componentName),
		secretProvider: secretProvider,
	}
}
