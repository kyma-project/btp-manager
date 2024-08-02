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
	ServiceBindingsPath  = "/v1/service_bindings"

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
		errMsg := fmt.Sprintf("failed to build http client from secret %q in namespace %q", secretName, secretNamespace)
		c.logger.Error(errMsg, "error", err)
		return types.NewServiceManagerClientError(errMsg)
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
	resp, err := c.sendRequest(http.MethodGet, c.smURL+ServiceOfferingsPath, nil)
	if err != nil {
		c.logger.Error("failed to send GET Service Offerings request", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return c.serviceOfferingsResponse(resp)
	default:
		return nil, c.errorResponse(resp)
	}
}

func (c *Client) serviceOfferingsResponse(resp *http.Response) (*types.ServiceOfferings, error) {
	body, err := c.readResponseBody(resp.Body)
	if err != nil {
		c.logger.Error("failed to read Service Offerings response body", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	sos := &types.ServiceOfferings{}
	if err := json.Unmarshal(body, sos); err != nil {
		c.logger.Error("failed to unmarshal Service Offerings response body", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	return sos, nil
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
	resp, err := c.sendRequest(http.MethodGet, c.smURL+ServiceOfferingsPath+"/"+serviceOfferingID, nil)
	if err != nil {
		c.logger.Error("failed to send GET Service Offering request", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return c.serviceOfferingResponse(resp)
	default:
		return nil, c.errorResponse(resp)
	}
}

func (c *Client) serviceOfferingResponse(resp *http.Response) (*types.ServiceOffering, error) {
	body, err := c.readResponseBody(resp.Body)
	if err != nil {
		c.logger.Error("failed to read Service Offering response body", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	so := &types.ServiceOffering{}
	if err := json.Unmarshal(body, so); err != nil {
		c.logger.Error("failed to unmarshal Service Offering response body", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	return so, nil
}

func (c *Client) servicePlansForServiceOffering(serviceOfferingID string) (*types.ServicePlans, error) {
	req, err := http.NewRequest(http.MethodGet, c.smURL+ServicePlansPath, nil)
	if err != nil {
		c.logger.Error("failed to create GET Service Plans request", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}
	values := req.URL.Query()
	values.Add(URLFieldQueryKey, fmt.Sprintf(servicePlansForServiceOfferingQueryFormat, serviceOfferingID))
	req.URL.RawQuery = values.Encode()
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("failed to send GET Service Plans request", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return c.servicePlansResponse(resp)
	default:
		return nil, c.errorResponse(resp)
	}
}

func (c *Client) servicePlansResponse(resp *http.Response) (*types.ServicePlans, error) {
	body, err := c.readResponseBody(resp.Body)
	if err != nil {
		c.logger.Error("failed to read Service Plans response body", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	sps := &types.ServicePlans{}
	if err := json.Unmarshal(body, sps); err != nil {
		c.logger.Error("failed to unmarshal Service Plans response body", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	return sps, nil
}

func (c *Client) ServiceInstances() (*types.ServiceInstances, error) {
	resp, err := c.sendRequest(http.MethodGet, c.smURL+ServiceInstancesPath, nil)
	if err != nil {
		c.logger.Error("failed to send GET Service Instances request", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return c.serviceInstancesResponse(resp)
	default:
		return nil, c.errorResponse(resp)
	}
}

func (c *Client) serviceInstancesResponse(resp *http.Response) (*types.ServiceInstances, error) {
	body, err := c.readResponseBody(resp.Body)
	if err != nil {
		c.logger.Error("failed to read Service Instances response body", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	sis := &types.ServiceInstances{}
	if err := json.Unmarshal(body, sis); err != nil {
		c.logger.Error("failed to unmarshal Service Instances response body", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	return sis, nil
}

func (c *Client) ServiceInstance(serviceInstanceID string) (*types.ServiceInstance, error) {
	resp, err := c.sendRequest(http.MethodGet, c.smURL+ServiceInstancesPath+"/"+serviceInstanceID, nil)
	if err != nil {
		c.logger.Error("failed to send GET Service Instance request", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return c.serviceInstanceResponse(resp)
	default:
		return nil, c.errorResponse(resp)
	}
}

func (c *Client) CreateServiceInstance(si *types.ServiceInstance) (*types.ServiceInstance, error) {
	requestBody, err := json.Marshal(si)
	if err != nil {
		c.logger.Error("failed to marshal POST Service Instance request body", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	resp, err := c.sendRequest(http.MethodPost, c.smURL+ServiceInstancesPath, bytes.NewBuffer(requestBody))
	if err != nil {
		c.logger.Error("failed to send POST Service Instance request", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
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

func (c *Client) UpdateServiceInstance(si *types.ServiceInstanceUpdateRequest) (*types.ServiceInstance, error) {
	id := *si.ID
	si.ID = nil

	if !c.validServiceInstanceUpdateRequestBody(si) {
		c.logger.Warn("invalid PATCH Service Instance request body - share fields must be updated alone")
		return nil, types.NewServiceManagerClientError("invalid request body - share fields must be updated alone")
	}

	requestBody, err := json.Marshal(si)
	if err != nil {
		c.logger.Error("failed to marshal PATCH Service Instance request body", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	resp, err := c.sendRequest(http.MethodPatch, c.smURL+ServiceInstancesPath+"/"+id, bytes.NewBuffer(requestBody))
	if err != nil {
		c.logger.Error("failed to send PATCH Service Instance request", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
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

func (c *Client) validServiceInstanceUpdateRequestBody(si *types.ServiceInstanceUpdateRequest) bool {
	if si.Shared != nil {
		return si.ID == nil && si.Name == nil && si.ServicePlanID == nil && si.Parameters == nil && len(si.Labels) == 0
	}
	return true
}

func (c *Client) serviceInstanceResponse(resp *http.Response) (*types.ServiceInstance, error) {
	body, err := c.readResponseBody(resp.Body)
	if err != nil {
		c.logger.Error("failed to read Service Instance response body", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	si := &types.ServiceInstance{}
	if err := json.Unmarshal(body, si); err != nil {
		c.logger.Error("failed to unmarshal Service Instance response body", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	return si, nil
}

func (c *Client) ServiceInstanceParameters(serviceInstanceID string) (map[string]string, error) {
	resp, err := c.sendRequest(http.MethodGet, c.smURL+ServiceInstancesPath+"/"+serviceInstanceID+"/parameters", nil)
	if err != nil {
		c.logger.Error("failed to send GET Service Instance parameters request", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return c.paramsResponse(resp)
	default:
		return nil, c.errorResponse(resp)
	}
}

func (c *Client) ServiceInstanceWithPlanName(serviceInstanceID string) (*types.ServiceInstance, error) {
	si, err := c.ServiceInstance(serviceInstanceID)
	if err != nil {
		return nil, err
	}
	plan, err := c.ServicePlan(si.ServicePlanID)
	if err != nil {
		return nil, err
	}
	si.ServicePlanName = plan.Name
	return si, nil
}

func (c *Client) ServicePlan(servicePlanID string) (*types.ServicePlan, error) {
	resp, err := c.sendRequest(http.MethodGet, c.smURL+ServicePlansPath+"/"+servicePlanID, nil)
	if err != nil {
		c.logger.Error("failed to send GET Service Plan request", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return c.servicePlanResponse(resp)
	default:
		return nil, c.errorResponse(resp)
	}
}

func (c *Client) servicePlanResponse(resp *http.Response) (*types.ServicePlan, error) {
	body, err := c.readResponseBody(resp.Body)
	if err != nil {
		c.logger.Error("failed to read Service Plan response body", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	p := &types.ServicePlan{}
	if err := json.Unmarshal(body, p); err != nil {
		c.logger.Error("failed to unmarshal Service Plan response body", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	return p, nil
}

func (c *Client) DeleteServiceInstance(serviceInstanceID string) error {
	resp, err := c.sendRequest(http.MethodDelete, c.smURL+ServiceInstancesPath+"/"+serviceInstanceID, nil)
	if err != nil {
		c.logger.Error("failed to send DELETE Service Instance request", "error", err)
		return types.NewServiceManagerClientError(err.Error())
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

func (c *Client) ServiceBindings() (*types.ServiceBindings, error) {
	return c.ServiceBindingsFor("")
}

func (c *Client) ServiceBindingsFor(serviceInstanceId string) (*types.ServiceBindings, error) {
	req, err := http.NewRequest(http.MethodGet, c.smURL+ServiceBindingsPath, nil)
	if err != nil {
		c.logger.Error("failed to create GET Service Bindings request", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}
	req.Header.Add("Content-Type", "application/json")

	if serviceInstanceId != "" {
		values := req.URL.Query()
		values.Add("fieldQuery", "service_instance_id eq '"+serviceInstanceId+"'")
		req.URL.RawQuery = values.Encode()
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("failed to send GET Service Bindings request", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return c.serviceBindingsResponse(resp)
	default:
		return nil, c.errorResponse(resp)
	}
}

func (c *Client) serviceBindingsResponse(resp *http.Response) (*types.ServiceBindings, error) {
	body, err := c.readResponseBody(resp.Body)
	if err != nil {
		c.logger.Error("failed to read Service Bindings response body", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	sbs := &types.ServiceBindings{}
	if err := json.Unmarshal(body, sbs); err != nil {
		c.logger.Error("failed to unmarshal Service Bindings response body", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	return sbs, nil
}

func (c *Client) ServiceBinding(serviceBindingId string) (*types.ServiceBinding, error) {
	resp, err := c.sendRequest(http.MethodGet, c.smURL+ServiceBindingsPath+"/"+serviceBindingId, nil)
	if err != nil {
		c.logger.Error("failed to send GET Service Binding request", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return c.serviceBindingResponse(resp)
	default:
		return nil, c.errorResponse(resp)
	}
}

func (c *Client) CreateServiceBinding(sb *types.ServiceBinding) (*types.ServiceBinding, error) {
	reqBody, err := json.Marshal(sb)
	if err != nil {
		c.logger.Error("failed to marshal POST Service Binding request body", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	resp, err := c.sendRequest(http.MethodPost, c.smURL+ServiceBindingsPath, bytes.NewBuffer(reqBody))
	if err != nil {
		c.logger.Error("failed to send POST Service Binding request", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	switch resp.StatusCode {
	case http.StatusCreated:
		return c.serviceBindingResponse(resp)
	case http.StatusAccepted:
		return nil, nil
	default:
		return nil, c.errorResponse(resp)
	}
}

func (c *Client) serviceBindingResponse(resp *http.Response) (*types.ServiceBinding, error) {
	body, err := c.readResponseBody(resp.Body)
	if err != nil {
		c.logger.Error("failed to read Service Binding response body", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	var sb types.ServiceBinding
	if err := json.Unmarshal(body, &sb); err != nil {
		c.logger.Error("failed to unmarshal Service Binding response body", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	return &sb, nil
}

func (c *Client) DeleteServiceBinding(serviceBindingId string) error {
	resp, err := c.sendRequest(http.MethodDelete, c.smURL+ServiceBindingsPath+"/"+serviceBindingId, nil)
	if err != nil {
		c.logger.Error("failed to send DELETE Service Binding request", "error", err)
		return types.NewServiceManagerClientError(err.Error())
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

func (c *Client) ServiceBindingParameters(serviceBindingId string) (map[string]string, error) {
	resp, err := c.sendRequest(http.MethodGet, c.smURL+ServiceBindingsPath+"/"+serviceBindingId+"/parameters", nil)
	if err != nil {
		c.logger.Error("failed to send GET Service Binding parameters request", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return c.paramsResponse(resp)
	default:
		return nil, c.errorResponse(resp)
	}
}

func (c *Client) paramsResponse(resp *http.Response) (map[string]string, error) {
	body, err := c.readResponseBody(resp.Body)
	if err != nil {
		c.logger.Error("failed to read parameters response body", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	var params map[string]string
	if err := json.Unmarshal(body, params); err != nil {
		c.logger.Error("failed to unmarshal parameters response body", "error", err)
		return nil, types.NewServiceManagerClientError(err.Error())
	}

	return params, nil
}

func (c *Client) readResponseBody(respBody io.ReadCloser) ([]byte, error) {
	defer respBody.Close()
	bodyInBytes, err := io.ReadAll(respBody)
	if err != nil {
		return nil, err
	}
	return bodyInBytes, nil
}

func (c *Client) sendRequest(method string, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if method == http.MethodPost || method == http.MethodPatch || method == http.MethodPut {
		req.Header.Add("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) errorResponse(resp *http.Response) error {
	body, err := c.readResponseBody(resp.Body)
	if err != nil {
		return types.NewServiceManagerClientError(err.Error())
	}

	errResp := &types.ErrorResponse{}
	if err = json.Unmarshal(body, errResp); err != nil {
		return types.NewServiceManagerClientError(err.Error())
	}

	return errResp
}
