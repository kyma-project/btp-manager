package servicemanager

import (
	"bytes"
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
	componentName    = "ServiceManagerClient"
	defaultSecret    = "sap-btp-service-operator"
	defaultNamespace = "kyma-system"

	ServiceOfferingsPath = "/v1/service_offerings"
	ServicePlansPath     = "/v1/service_plans"
	ServiceInstancesPath = "/v1/service_instances"

	// see https://help.sap.com/docs/service-manager/sap-service-manager/filtering-parameters-and-operators
	URLFieldQueryKey                          = "fieldQuery"
	servicePlansForServiceOfferingQueryFormat = "service_offering_id eq '%s'"
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

func NewClient(ctx context.Context, logger *slog.Logger, secretProvider clusterobject.NamespacedProvider[*corev1.Secret]) *Client {
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

	body, err := c.readResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	var serviceOfferings types.ServiceOfferings
	if err := json.Unmarshal(body, &serviceOfferings); err != nil {
		return nil, err
	}

	return &serviceOfferings, nil
}

func (c *Client) ServiceOfferingDetails(serviceOfferingID string) (*types.ServiceOfferingDetails, error) {
	so, err := c.serviceOfferingByID(serviceOfferingID)
	if err != nil {
		return nil, err
	}

	plans, err := c.servicePlansForServiceOffering(serviceOfferingID)
	if err != nil {
		return nil, err
	}

	return &types.ServiceOfferingDetails{
		ServiceOffering: *so,
		ServicePlans:    *plans,
	}, nil
}

func (c *Client) serviceOfferingByID(serviceOfferingID string) (*types.ServiceOffering, error) {
	req, err := http.NewRequest(http.MethodGet, c.smURL+ServiceOfferingsPath+"/"+serviceOfferingID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := c.readResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	var so types.ServiceOffering
	if err := json.Unmarshal(body, &so); err != nil {
		return nil, err
	}

	return &so, nil
}

func (c *Client) servicePlansForServiceOffering(serviceOfferingID string) (*types.ServicePlans, error) {
	req, err := http.NewRequest(http.MethodGet, c.smURL+ServicePlansPath, nil)
	if err != nil {
		return nil, err
	}
	values := req.URL.Query()
	values.Add(URLFieldQueryKey, fmt.Sprintf(servicePlansForServiceOfferingQueryFormat, serviceOfferingID))
	req.URL.RawQuery = values.Encode()
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := c.readResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	var plans types.ServicePlans
	if err := json.Unmarshal(body, &plans); err != nil {
		return nil, err
	}

	return &plans, nil
}

func (c *Client) ServiceInstances() (*types.ServiceInstances, error) {
	req, err := http.NewRequest(http.MethodGet, c.smURL+ServiceInstancesPath, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := c.readResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}
	var serviceInstances types.ServiceInstances
	if err := json.Unmarshal(body, &serviceInstances); err != nil {
		return nil, err
	}

	return &serviceInstances, nil
}

func (c *Client) ServiceInstance(serviceInstanceID string) (*types.ServiceInstance, error) {
	req, err := http.NewRequest(http.MethodGet, c.smURL+ServiceInstancesPath+"/"+serviceInstanceID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := c.readResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	var si types.ServiceInstance
	if err := json.Unmarshal(body, &si); err != nil {
		return nil, err
	}

	return &si, nil
}

func (c *Client) ServiceInstanceParameters(serviceInstanceID string) (map[string]string, error) {
	req, err := http.NewRequest(http.MethodGet, c.smURL+ServiceInstancesPath+"/"+serviceInstanceID+"/parameters", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return c.serviceInstanceParamsResponse(resp)
	default:
		return nil, c.errorResponse(resp)
	}
}

func (c *Client) CreateServiceInstance(si *types.ServiceInstance) (*types.ServiceInstance, error) {
	requestBody, err := json.Marshal(si)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.smURL+ServiceInstancesPath, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusCreated:
		return c.serviceInstanceResponse(resp)
	case http.StatusAccepted:
		return nil, nil
	default:
		return nil, c.errorResponse(resp)
	}
}

func (c *Client) DeleteServiceInstance(serviceInstanceID string) error {
	req, err := http.NewRequest(http.MethodDelete, c.smURL+ServiceInstancesPath+"/"+serviceInstanceID, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		fallthrough
	case http.StatusAccepted:
		return nil
	default:
		return c.errorResponse(resp)
	}
}

func (c *Client) serviceInstanceResponse(resp *http.Response) (*types.ServiceInstance, error) {
	body, err := c.readResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	var siResp types.ServiceInstance
	if err := json.Unmarshal(body, &siResp); err != nil {
		return nil, err
	}

	return &siResp, nil
}

func (c *Client) serviceInstanceParamsResponse(resp *http.Response) (map[string]string, error) {
	body, err := c.readResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	var params map[string]string
	if err := json.Unmarshal(body, params); err != nil {
		return nil, err
	}

	return params, nil
}

func (c *Client) UpdateServiceInstance(si *types.ServiceInstanceUpdateRequest) (*types.ServiceInstance, error) {
	id := *si.ID
	si.ID = nil

	if !c.validServiceInstanceUpdateRequestBody(si) {
		return nil, fmt.Errorf("invalid request body - share fields must be updated alone")
	}

	requestBody, err := json.Marshal(si)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPatch, c.smURL+ServiceInstancesPath+"/"+id, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return c.serviceInstanceResponse(resp)
	case http.StatusAccepted:
		return nil, nil
	default:
		return nil, c.errorResponse(resp)
	}
}

func (c *Client) errorResponse(resp *http.Response) error {
	body, err := c.readResponseBody(resp.Body)
	if err != nil {
		return err
	}

	var errResp types.ErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return err
	}

	return fmt.Errorf("error: %s", errResp.Error())
}

func (c *Client) readResponseBody(respBody io.ReadCloser) ([]byte, error) {
	defer respBody.Close()
	bodyInBytes, err := io.ReadAll(respBody)
	if err != nil {
		return nil, err
	}
	return bodyInBytes, nil
}

func (c *Client) validServiceInstanceUpdateRequestBody(si *types.ServiceInstanceUpdateRequest) bool {
	if si.Shared != nil {
		return si.ID == nil && si.Name == nil && si.ServicePlanID == nil && si.Parameters == nil && len(si.Labels) == 0
	}
	return true
}
