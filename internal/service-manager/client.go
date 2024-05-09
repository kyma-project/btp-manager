package servicemanager

import (
	"context"
	"log/slog"
	"net/http"

	clusterobject "github.com/kyma-project/btp-manager/internal/cluster-object"
	"golang.org/x/oauth2/clientcredentials"
	corev1 "k8s.io/api/core/v1"
)

const componentName = "ServiceManagerClient"

type Client struct {
	ctx            context.Context
	logger         *slog.Logger
	secretProvider clusterobject.NamespacedProvider[*corev1.Secret]
	httpClient     *http.Client
}

func NewClient(ctx context.Context, logger *slog.Logger, secretProvider clusterobject.NamespacedProvider[*corev1.Secret]) *Client {
	return &Client{
		ctx:            ctx,
		logger:         logger.With("component", componentName),
		secretProvider: secretProvider,
	}
}

func (c *Client) getHttpClientForGivenSecret(ctx context.Context, secretName, secretNamespace string) (*http.Client, error) {
	secret, err := c.secretProvider.GetByNameAndNamespace(ctx, secretName, secretNamespace)
	if err != nil {
		return nil, err
	}

	cfg := clientcredentials.Config{
		ClientID:     string(secret.Data["clientid"]),
		ClientSecret: string(secret.Data["clientsecret"]),
		TokenURL:     string(secret.Data["tokenurl"]),
	}

	return cfg.Client(ctx), nil
}
