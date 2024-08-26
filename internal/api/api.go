package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/btp-manager/internal/api/requests"
	"github.com/kyma-project/btp-manager/internal/api/responses"
	clusterobject "github.com/kyma-project/btp-manager/internal/cluster-object"
	"github.com/kyma-project/btp-manager/internal/service-manager/types"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SMClient interface {
	SetForGivenSecret(ctx context.Context, name, namespace string) error
	ServiceInstances() (*types.ServiceInstances, error)
	CreateServiceInstance(*types.ServiceInstance) (*types.ServiceInstance, error)
	ServiceOfferingDetails(id string) (*types.ServiceOfferingDetails, error)
	ServiceOfferings() (*types.ServiceOfferings, error)
	ServiceInstanceWithPlanName(id string) (*types.ServiceInstance, error)
	UpdateServiceInstance(*types.ServiceInstanceUpdateRequest) (*types.ServiceInstance, error)
	DeleteServiceInstance(id string) error
	ServiceBindingsFor(serviceInstanceID string) (*types.ServiceBindings, error)
	CreateServiceBinding(*types.ServiceBinding) (*types.ServiceBinding, error)
	ServiceBinding(id string) (*types.ServiceBinding, error)
	DeleteServiceBinding(id string) error
	ServiceInstance(id string) (*types.ServiceInstance, error)
}
type MockSMClient struct {
	Original SMClient
}

func (m *MockSMClient) SetForGivenSecret(ctx context.Context, name, namespace string) error {
	return nil
}

func (m *MockSMClient) ServiceInstances() (*types.ServiceInstances, error) {
	return m.Original.ServiceInstances()
}

func (m *MockSMClient) CreateServiceInstance(si *types.ServiceInstance) (*types.ServiceInstance, error) {
	return m.Original.CreateServiceInstance(si)
}

func (m *MockSMClient) ServiceOfferingDetails(id string) (*types.ServiceOfferingDetails, error) {
	return m.Original.ServiceOfferingDetails(id)
}

func (m *MockSMClient) ServiceOfferings() (*types.ServiceOfferings, error) {
	return m.Original.ServiceOfferings()
}

func (m *MockSMClient) ServiceInstanceWithPlanName(id string) (*types.ServiceInstance, error) {
	return m.Original.ServiceInstanceWithPlanName(id)
}

func (m *MockSMClient) UpdateServiceInstance(siuReq *types.ServiceInstanceUpdateRequest) (*types.ServiceInstance, error) {
	return m.Original.UpdateServiceInstance(siuReq)
}

func (m *MockSMClient) DeleteServiceInstance(id string) error {
	return m.Original.DeleteServiceInstance(id)
}

func (m *MockSMClient) ServiceBindingsFor(serviceInstanceID string) (*types.ServiceBindings, error) {
	return m.Original.ServiceBindingsFor(serviceInstanceID)
}

func (m *MockSMClient) CreateServiceBinding(sb *types.ServiceBinding) (*types.ServiceBinding, error) {
	return m.Original.CreateServiceBinding(sb)
}

func (m *MockSMClient) ServiceBinding(id string) (*types.ServiceBinding, error) {
	return m.Original.ServiceBinding(id)
}

func (m *MockSMClient) DeleteServiceBinding(id string) error {
	return m.Original.DeleteServiceBinding(id)
}

func (m *MockSMClient) ServiceInstance(id string) (*types.ServiceInstance, error) {
	return m.Original.ServiceInstance(id)
}

type Config struct {
	Port         string        `envconfig:"default=8080"`
	ReadTimeout  time.Duration `envconfig:"default=30s"`
	WriteTimeout time.Duration `envconfig:"default=90s"`
	IdleTimeout  time.Duration `envconfig:"default=120s"`
}

type API struct {
	server        *http.Server
	smClient      SMClient
	secretManager clusterobject.Manager[*corev1.SecretList, *corev1.Secret]
	frontendFS    http.FileSystem
	logger        *slog.Logger
}

func NewAPI(cfg Config, serviceManagerClient SMClient, secretManager clusterobject.Manager[*corev1.SecretList, *corev1.Secret], fs http.FileSystem) *API {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
	return &API{
		server:        srv,
		smClient:      serviceManagerClient,
		secretManager: secretManager,
		frontendFS:    fs,
		logger:        slog.Default()}
}

func (a *API) Start() {
	router := http.NewServeMux()

	a.AttachRoutes(router)
	a.server.Handler = router

	log.Fatal(a.server.ListenAndServe())
}

func (a *API) AttachRoutes(router *http.ServeMux) {
	router.HandleFunc("GET /api/secrets", a.ListSecrets)
	router.HandleFunc("GET /api/service-offerings", a.HandleGetServiceOffering)
	router.HandleFunc("GET /api/service-instances", a.HandleGetServiceInstances)
	router.HandleFunc("POST /api/service-instances", a.CreateServiceInstance)
	router.HandleFunc("PATCH /api/service-instances", a.UpdateServiceInstance)
	router.HandleFunc("DELETE /api/service-instances", a.DeleteServiceInstance)
	router.HandleFunc("GET /api/service-bindings", a.HandleGetServiceBinding)
	router.HandleFunc("POST /api/service-bindings", a.CreateServiceBinding)
	router.HandleFunc("DELETE /api/service-bindings", a.DeleteServiceBinding)
	router.HandleFunc("PUT /api/service-bindings/{id}", a.RestoreSecret)

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			fullPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
			_, err := a.frontendFS.Open(fullPath)
			if err != nil {
				if !os.IsNotExist(err) {
					panic(err)
				}
				r.URL.Path = "/"
			}
		}
		http.FileServer(a.frontendFS).ServeHTTP(w, r)
	})
}

func (a *API) Address() string {
	return a.server.Addr
}

func (a *API) CreateServiceInstance(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	namespace := request.URL.Query().Get("sm_secret_namespace")
	name := request.URL.Query().Get("sm_secret_name")
	err := a.smClient.SetForGivenSecret(context.Background(), name, namespace)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	csiRequest, err := a.decodeCreateServiceInstanceRequest(request)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	si := csiRequest.ConvertToServiceInstance()
	createdSI, err := a.smClient.CreateServiceInstance(si)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	createdSI.ServicePlanName = si.ServicePlanName
	response, err := json.Marshal(responses.ToServiceInstanceVM(createdSI))
	if err != nil {
		a.handleError(writer, err)
		return
	}
	a.sendResponse(writer, response, http.StatusCreated)
}

func (a *API) GetServiceOffering(writer http.ResponseWriter, request *http.Request, id string) {
	details, err := a.smClient.ServiceOfferingDetails(id)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	response, err := json.Marshal(responses.ToServiceOfferingDetailsVM(details))
	if err != nil {
		a.handleError(writer, err)
		return
	}
	a.sendResponse(writer, response)
}

func (a *API) ListServiceOfferings(writer http.ResponseWriter, request *http.Request) {
	offerings, err := a.smClient.ServiceOfferings()
	if err != nil {
		a.handleError(writer, err)
		return
	}
	response, err := json.Marshal(responses.ToServiceOfferingsVM(offerings))
	if err != nil {
		a.handleError(writer, err)
		return
	}
	a.sendResponse(writer, response)
}

func (a *API) HandleGetServiceOffering(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	namespace := request.URL.Query().Get("sm_secret_namespace")
	name := request.URL.Query().Get("sm_secret_name")
	err := a.smClient.SetForGivenSecret(context.Background(), name, namespace)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	id := request.URL.Query().Get("id")
	if id == "" {
		a.ListServiceOfferings(writer, request)
		return
	}
	a.GetServiceOffering(writer, request, id)
}

func (a *API) ListSecrets(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	secrets, err := a.secretManager.GetAll(context.Background())
	if err != nil {
		a.handleError(writer, err)
		return
	}
	response, err := json.Marshal(responses.ToSecretVM(*secrets))
	if err != nil {
		a.handleError(writer, err)
		return
	}
	a.sendResponse(writer, response)
}

func (a *API) GetServiceInstance(writer http.ResponseWriter, request *http.Request, id string) {
	si, err := a.smClient.ServiceInstanceWithPlanName(id)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	response, err := json.Marshal(responses.ToServiceInstanceVM(si))
	if err != nil {
		a.handleError(writer, err)
		return
	}
	a.sendResponse(writer, response)
}

func (a *API) ListServiceInstances(writer http.ResponseWriter, request *http.Request) {
	sis, err := a.smClient.ServiceInstances()
	if err != nil {
		a.handleError(writer, err)
		return
	}
	response, err := json.Marshal(responses.ToServiceInstancesVM(sis))
	if err != nil {
		a.handleError(writer, err)
		return
	}
	a.sendResponse(writer, response)
}

func (a *API) HandleGetServiceInstances(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	namespace := request.URL.Query().Get("sm_secret_namespace")
	name := request.URL.Query().Get("sm_secret_name")
	err := a.smClient.SetForGivenSecret(context.Background(), name, namespace)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	id := request.URL.Query().Get("id")
	if id == "" {
		a.ListServiceInstances(writer, request)
		return
	}
	a.GetServiceInstance(writer, request, id)
}

func (a *API) ListServiceBindings(writer http.ResponseWriter, request *http.Request) {
	queryParams := request.URL.Query()
	serviceInstanceId := queryParams.Get("service_instance_id")
	sbs, err := a.smClient.ServiceBindingsFor(serviceInstanceId)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	if sbs == nil || len(sbs.Items) == 0 {
		a.sendResponse(writer, nil, http.StatusNoContent)
		return
	}
	sbSecrets := a.ServiceBindingsSecrets(sbs)
	sbsVM, err := responses.ToServiceBindingsVM(sbs, sbSecrets)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	response, err := json.Marshal(sbsVM)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	a.sendResponse(writer, response)
}

func (a *API) CreateServiceBinding(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	namespace := request.URL.Query().Get("sm_secret_namespace")
	name := request.URL.Query().Get("sm_secret_name")
	err := a.smClient.SetForGivenSecret(context.Background(), name, namespace)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	serviceBindingRequest := &requests.CreateServiceBinding{}
	err = json.NewDecoder(request.Body).Decode(serviceBindingRequest)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	secretExists, err := a.secretExists(serviceBindingRequest.SecretName, serviceBindingRequest.SecretNamespace)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	if secretExists {
		secretExistsErr := fmt.Errorf("secret \"%s\" in \"%s\" namespace already exists", serviceBindingRequest.SecretName, serviceBindingRequest.SecretNamespace)
		a.handleError(writer, secretExistsErr, http.StatusConflict)
		return
	}
	si, err := a.smClient.ServiceInstance(serviceBindingRequest.ServiceInstanceID)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	sb, err := requests.ToServiceBinding(serviceBindingRequest, si)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	createdServiceBinding, err := a.smClient.CreateServiceBinding(sb)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	secret, err := generateSecretFromSISBData(si, createdServiceBinding, serviceBindingRequest)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	sbVM, err := responses.ToServiceBindingVM(createdServiceBinding)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	if secret.Name != "" && secret.Namespace != "" {
		err = a.secretManager.Create(context.Background(), secret)
		if err != nil {
			a.handleError(writer, err)
			return
		}
		sbVM.SecretName = secret.Name
		sbVM.SecretNamespace = secret.Namespace
	}
	response, err := json.Marshal(sbVM)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	a.sendResponse(writer, response, http.StatusCreated)
}

func (a *API) GetServiceBinding(writer http.ResponseWriter, request *http.Request, id string) {
	sb, err := a.smClient.ServiceBinding(id)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	sbVM, err := responses.ToServiceBindingVM(sb)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	secrets, err := a.secretsForGivenServiceBindingID(sb.ID)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	if len(secrets.Items) > 0 {
		sbVM.SecretName = secrets.Items[0].Name
		sbVM.SecretNamespace = secrets.Items[0].Namespace
	}
	response, err := json.Marshal(sbVM)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	a.sendResponse(writer, response)
}

func (a *API) HandleGetServiceBinding(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	namespace := request.URL.Query().Get("sm_secret_namespace")
	name := request.URL.Query().Get("sm_secret_name")
	err := a.smClient.SetForGivenSecret(context.Background(), name, namespace)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	id := request.URL.Query().Get("id")
	if id == "" {
		a.ListServiceBindings(writer, request)
		return
	}
	a.GetServiceBinding(writer, request, id)
}

func (a *API) DeleteServiceBinding(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	namespace := request.URL.Query().Get("sm_secret_namespace")
	name := request.URL.Query().Get("sm_secret_name")
	err := a.smClient.SetForGivenSecret(context.Background(), name, namespace)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	id := request.URL.Query().Get("id")
	secrets, err := a.secretsForGivenServiceBindingID(id)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	if err := a.secretManager.DeleteList(context.Background(), secrets); err != nil {
		a.handleError(writer, err)
		return
	}
	if err := a.smClient.DeleteServiceBinding(id); err != nil {
		a.handleError(writer, err)
		return
	}
}

func (a *API) secretsForGivenServiceBindingID(sbID string) (*corev1.SecretList, error) {
	filterLabels := map[string]string{
		clusterobject.ManagedByLabelKey:     clusterobject.OperatorName,
		clusterobject.ServiceBindingIDLabel: sbID,
	}
	secrets, err := a.secretManager.GetAllByLabels(context.Background(), filterLabels)
	return secrets, err
}

func (a *API) setupCors(writer http.ResponseWriter, request *http.Request) {
	a.logger.Info(fmt.Sprintf("api call to -> %s as: %s", request.RequestURI, request.Method))
	origin := request.Header.Get("Origin")
	origin = strings.ReplaceAll(origin, "\r", "")
	origin = strings.ReplaceAll(origin, "\n", "")
	writer.Header().Set("Access-Control-Allow-Origin", origin)
}

func (a *API) decodeCreateServiceInstanceRequest(request *http.Request) (*requests.CreateServiceInstance, error) {
	var csiRequest requests.CreateServiceInstance
	err := json.NewDecoder(request.Body).Decode(&csiRequest)
	if err != nil {
		return nil, err
	}
	return &csiRequest, nil
}

func (a *API) UpdateServiceInstance(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	namespace := request.URL.Query().Get("sm_secret_namespace")
	name := request.URL.Query().Get("sm_secret_name")
	err := a.smClient.SetForGivenSecret(context.Background(), name, namespace)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	id := request.URL.Query().Get("id")
	siuReq, err := a.decodeServiceInstanceUpdateRequest(request)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	siuReq.ID = &id
	updatedSI, err := a.smClient.UpdateServiceInstance(siuReq)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	response, err := json.Marshal(responses.ToServiceInstanceVM(updatedSI))
	if err != nil {
		a.handleError(writer, err)
		return
	}
	a.sendResponse(writer, response)
}

func (a *API) decodeServiceInstanceUpdateRequest(request *http.Request) (*types.ServiceInstanceUpdateRequest, error) {
	var siuRequest types.ServiceInstanceUpdateRequest
	err := json.NewDecoder(request.Body).Decode(&siuRequest)
	if err != nil {
		return nil, err
	}
	return &siuRequest, nil
}

func (a *API) DeleteServiceInstance(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	namespace := request.URL.Query().Get("sm_secret_namespace")
	name := request.URL.Query().Get("sm_secret_name")
	err := a.smClient.SetForGivenSecret(context.Background(), name, namespace)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	id := request.URL.Query().Get("id")
	if err := a.smClient.DeleteServiceInstance(id); err != nil {
		a.handleError(writer, err)
		return
	}
}

func (a *API) sendResponse(writer http.ResponseWriter, response []byte, optionalHTTPStatusCode ...int) {
	if len(optionalHTTPStatusCode) > 0 {
		writer.WriteHeader(optionalHTTPStatusCode[0])
	}
	if len(response) > 0 {
		writer.Header().Set("Content-Type", "application/json")
		if _, err := writer.Write(response); err != nil {
			a.logger.Error(err.Error())
		}
	}
}

func (a *API) handleError(writer http.ResponseWriter, errToHandle error, fallbackHTTPStatusCode ...int) {
	httpStatusCode := http.StatusInternalServerError
	if len(fallbackHTTPStatusCode) > 0 {
		httpStatusCode = fallbackHTTPStatusCode[0]
	}
	var smError *types.ErrorResponse
	if errors.As(errToHandle, &smError) {
		if smError.BrokerError != nil {
			writer.WriteHeader(smError.BrokerError.StatusCode)
			_, err := writer.Write([]byte(smError.Error()))
			if err != nil {
				a.logger.Error(err.Error())
				return
			}
			return
		}
		writer.WriteHeader(smError.StatusCode)
		_, err := writer.Write([]byte(smError.Error()))
		if err != nil {
			a.logger.Error(err.Error())
			return
		}
		return
	}
	a.logger.Error(errToHandle.Error())
	writer.WriteHeader(httpStatusCode)
	if _, err := writer.Write([]byte(errToHandle.Error())); err != nil {
		a.logger.Error(err.Error())
	}
	return
}

func (a *API) ServiceBindingsSecrets(sbs *types.ServiceBindings) responses.ServiceBindingSecret {
	serviceBindingsSecrets := make(responses.ServiceBindingSecret)
	for _, sb := range sbs.Items {
		secrets, err := a.secretsForGivenServiceBindingID(sb.ID)
		if err != nil {
			a.logger.Error("failed to get secrets for service binding", "service binding id", sb.ID, "error", err)
			continue
		}
		if secrets != nil && len(secrets.Items) > 0 {
			serviceBindingsSecrets[sb.ID] = &secrets.Items[0]
		}
	}

	return serviceBindingsSecrets
}

func (a *API) secretExists(secretName, secretNamespace string) (bool, error) {
	existingSecret, err := a.secretManager.GetByNameAndNamespace(context.Background(), secretName, secretNamespace)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return existingSecret != nil && existingSecret.Name == secretName && existingSecret.Namespace == secretNamespace, nil
}

func (a *API) RestoreSecret(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	sbID := request.PathValue("id")
	sb, err := a.smClient.ServiceBinding(sbID)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	serviceBindingRequest := &requests.CreateServiceBinding{}
	err = json.NewDecoder(request.Body).Decode(serviceBindingRequest)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	secretExists, err := a.secretExists(serviceBindingRequest.SecretName, serviceBindingRequest.SecretNamespace)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	if secretExists {
		secretExistsErr := fmt.Errorf("secret \"%s\" in \"%s\" namespace already exists", serviceBindingRequest.SecretName, serviceBindingRequest.SecretNamespace)
		a.handleError(writer, secretExistsErr, http.StatusConflict)
		return
	}
	si, err := a.smClient.ServiceInstance(serviceBindingRequest.ServiceInstanceID)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	secret, err := generateSecretFromSISBData(si, sb, serviceBindingRequest)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	sbVM, err := responses.ToServiceBindingVM(sb)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	err = a.secretManager.Create(context.Background(), secret)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	sbVM.SecretName = secret.Name
	sbVM.SecretNamespace = secret.Namespace

	response, err := json.Marshal(sbVM)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	a.sendResponse(writer, response, http.StatusCreated)
}

func generateSecretFromSISBData(si *types.ServiceInstance, sb *types.ServiceBinding, createSBRequest *requests.CreateServiceBinding) (*corev1.Secret, error) {
	var secretName, secretNamespace string
	var err error
	if createSBRequest.SecretName != "" {
		secretName = createSBRequest.SecretName
	} else {
		slicedUUID := strings.Split(uuid.NewString(), "-")
		suffix := strings.Join(slicedUUID[:2], "-")
		secretName = fmt.Sprintf("%s-%s", sb.Name, suffix)
	}
	if createSBRequest.SecretNamespace != "" {
		secretNamespace = createSBRequest.SecretNamespace
	} else {
		secretNamespace, err = sb.ContextValueByFieldName(types.ContextNamespace)
		if err != nil {
			return nil, fmt.Errorf("failed to get namespace from service binding context: %w", err)
		}
	}
	labels := map[string]string{
		clusterobject.ManagedByLabelKey:        clusterobject.OperatorName,
		clusterobject.ServiceBindingIDLabel:    sb.ID,
		clusterobject.ServiceInstanceIDLabel:   si.ID,
		clusterobject.ServiceInstanceNameLabel: si.Name,
	}
	if sb.Labels != nil {
		existingClusterIDLabels, exists := sb.Labels[types.ClusterIDLabel]
		if exists && len(existingClusterIDLabels) > 0 {
			labels[clusterobject.ClusterIDLabel] = existingClusterIDLabels[0]
		}
	}
	creds, err := normalizeCredentials(sb.Credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize credentials for secret's data: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: secretNamespace,
			Labels:    labels,
		},
		Data: creds,
	}

	return secret, nil
}

func normalizeCredentials(sbCredentials json.RawMessage) (map[string][]byte, error) {
	data := make(map[string]interface{})
	if err := json.Unmarshal(sbCredentials, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials from service binding: %w", err)
	}
	normalized := make(map[string][]byte)
	for k, v := range data {
		keyString := strings.Replace(k, " ", "_", -1)
		normalizedValue, err := serialize(v)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize value for key %s: %w", k, err)
		}
		normalized[keyString] = normalizedValue
	}

	return normalized, nil
}

func serialize(value interface{}) ([]byte, error) {
	if byteArrayVal, ok := value.([]byte); ok {
		return byteArrayVal, nil
	}
	if strVal, ok := value.(string); ok {
		return []byte(strVal), nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return data, nil
}
