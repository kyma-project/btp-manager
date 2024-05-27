package servicemanager

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	clusterobject "github.com/kyma-project/btp-manager/internal/cluster-object"
	"github.com/kyma-project/btp-manager/internal/service-manager/types"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/json"
)

const (
	componentName        = "ServiceManagerClient"
	defaultSecret        = "sap-btp-service-operator"
	defaultNamespace     = "kyma-system"
	ServiceOfferingsPath = "/v1/service_offerings"
)

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

func NewClient(
	ctx context.Context, logger *slog.Logger, secretProvider *clusterobject.SecretProvider,
) *Client {
	return &Client{
		ctx:            ctx,
		logger:         logger.With("component", componentName),
		secretProvider: secretProvider,
	}
}

func (c *Client) Defaults(ctx context.Context) error {
	if err := c.buildHTTPClient(ctx, defaultSecret, defaultNamespace); err != nil {
		if k8serrors.IsNotFound(err) {
			c.logger.Warn(fmt.Sprintf("%s secret not found in %s namespace", defaultSecret, defaultNamespace))
			return nil
		}
		c.logger.Error("failed to build http client", "error", err)
		return err
	}

	return nil
}

func (c *Client) SetForGivenSecret(ctx context.Context, secretName, secretNamespace string) error {
	if err := c.buildHTTPClient(ctx, secretName, secretNamespace); err != nil {
		c.logger.Error("failed to build http client", "error", err)
		return err
	}

	return nil
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

func (c *Client) SetHTTPClient(httpClient *http.Client) {
	c.httpClient = httpClient
}

func (c *Client) SetSMURL(smURL string) {
	c.smURL = smURL
}

func (c *Client) ServiceOfferings() (*types.ServiceOfferings, error) {
	req, err := http.NewRequest(http.MethodGet, c.smURL+ServiceOfferingsPath, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var serviceOfferings types.ServiceOfferings
	if err := json.Unmarshal(body, &serviceOfferings); err != nil {
		return nil, err
	}

	return &serviceOfferings, nil
}
