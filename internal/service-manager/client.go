package servicemanager

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	clusterobject "github.com/kyma-project/btp-manager/internal/cluster-object"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

const componentName = "ServiceManagerClient"

type Config struct {
	ClientID       string
	ClientSecret   string
	URL            string
	TokenURL       string
	TokenURLSuffix string
}

type Client struct {
	ctx            context.Context
	logger         *slog.Logger
	secretProvider clusterobject.NamespacedProvider[*corev1.Secret]
	httpClient     *http.Client
	smURL          string
}

func NewClient(ctx context.Context, logger *slog.Logger, secretProvider clusterobject.NamespacedProvider[*corev1.Secret]) *Client {
	return &Client{
		ctx:            ctx,
		logger:         logger.With("component", componentName),
		secretProvider: secretProvider,
	}
}

func (c *Client) SetHTTPClient(httpClient *http.Client) {
	c.httpClient = httpClient
}

func (c *Client) buildHTTPClient(ctx context.Context, secretName, secretNamespace string) error {
	cfg, err := c.getSMConfigFromGivenSecret(ctx, secretName, secretNamespace)
	if err != nil {
		return err
	}

	oauth2ClientCfg := &clientcredentials.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		TokenURL:     cfg.TokenURL + cfg.TokenURLSuffix,
	}
	httpClient := preconfiguredHTTPClient()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	c.smURL = cfg.URL
	c.httpClient = oauth2.NewClient(ctx, oauth2ClientCfg.TokenSource(ctx))

	return nil
}

func (c *Client) getSMConfigFromGivenSecret(ctx context.Context, secretName, secretNamespace string) (*Config, error) {
	secret, err := c.secretProvider.GetByNameAndNamespace(ctx, secretName, secretNamespace)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			c.logger.Warn("secret not found", "name", secretName, "namespace", secretNamespace)
		}
		return nil, err
	}

	return &Config{
		ClientID:       string(secret.Data["clientid"]),
		ClientSecret:   string(secret.Data["clientsecret"]),
		URL:            string(secret.Data["sm_url"]),
		TokenURL:       string(secret.Data["tokenurl"]),
		TokenURLSuffix: string(secret.Data["tokenurlsuffix"]),
	}, nil
}

func preconfiguredHTTPClient() *http.Client {
	client := &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	return client
}
