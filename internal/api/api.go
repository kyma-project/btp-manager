package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/btp-manager/internal/api/requests"
	"github.com/kyma-project/btp-manager/internal/api/responses"
	clusterobject "github.com/kyma-project/btp-manager/internal/cluster-object"
	servicemanager "github.com/kyma-project/btp-manager/internal/service-manager"
	"github.com/kyma-project/btp-manager/internal/service-manager/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Config struct {
	Port         string        `envconfig:"default=8080"`
	ReadTimeout  time.Duration `envconfig:"default=30s"`
	WriteTimeout time.Duration `envconfig:"default=90s"`
	IdleTimeout  time.Duration `envconfig:"default=120s"`
}

type API struct {
	server        *http.Server
	smClient      *servicemanager.Client
	secretManager clusterobject.Manager[*corev1.SecretList, *corev1.Secret]
	frontendFS    http.FileSystem
	logger        *slog.Logger
}

func NewAPI(cfg Config, serviceManagerClient *servicemanager.Client, secretManager clusterobject.Manager[*corev1.SecretList, *corev1.Secret], fs http.FileSystem) *API {
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
	router.HandleFunc("GET /api/service-offerings/{namespace}/{name}", a.ListServiceOfferings)
	router.HandleFunc("GET /api/service-offerings/{id}", a.GetServiceOffering)
	router.HandleFunc("GET /api/service-instances", a.ListServiceInstances)
	router.HandleFunc("GET /api/service-instances/{id}", a.GetServiceInstance)
	router.HandleFunc("POST /api/service-instances", a.CreateServiceInstance)
	router.HandleFunc("PATCH /api/service-instances/{id}", a.UpdateServiceInstance)
	router.HandleFunc("DELETE /api/service-instances/{id}", a.DeleteServiceInstance)
	router.HandleFunc("GET /api/service-bindings", a.ListServiceBindings)
	router.HandleFunc("GET /api/service-bindings/{id}", a.GetServiceBinding)
	router.HandleFunc("POST /api/service-bindings", a.CreateServiceBinding)
	router.HandleFunc("DELETE /api/service-bindings/{id}", a.DeleteServiceBinding)
	router.Handle("GET /", http.FileServer(a.frontendFS))
}

func (a *API) Address() string {
	return a.server.Addr
}

func (a *API) CreateServiceInstance(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
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

func (a *API) GetServiceOffering(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	id := request.PathValue("id")
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
	a.setupCors(writer, request)
	namespace := request.PathValue("namespace")
	name := request.PathValue("name")
	err := a.smClient.SetForGivenSecret(context.Background(), name, namespace)
	if err != nil {
		a.handleError(writer, err)
		return
	}
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

func (a *API) GetServiceInstance(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	id := request.PathValue("id")
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
	a.setupCors(writer, request)
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

func (a *API) ListServiceBindings(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	queryParams := request.URL.Query()
	serviceInstanceId := queryParams.Get("service_instance_id")
	sbs, err := a.smClient.ServiceBindingsFor(serviceInstanceId)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	sbsVM, err := responses.ToServiceBindingsVM(sbs)
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
	var serviceBindingRequest requests.CreateServiceBinding
	err := json.NewDecoder(request.Body).Decode(&serviceBindingRequest)
	if err != nil {
		a.handleError(writer, err)
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
	createdServiceBinding, err := a.smClient.CreateServiceBinding(&sb)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	secret, err := generateSecretFromSISBData(si, createdServiceBinding)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	err = a.secretManager.Create(context.Background(), secret)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	sbVM, err := responses.ToServiceBindingVM(createdServiceBinding)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	response, err := json.Marshal(sbVM)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	a.sendResponse(writer, response, http.StatusCreated)
}

func (a *API) GetServiceBinding(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	id := request.PathValue("id")
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
	response, err := json.Marshal(sbVM)
	if err != nil {
		a.handleError(writer, err)
		return
	}
	a.sendResponse(writer, response)
}

func (a *API) DeleteServiceBinding(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	id := request.PathValue("id")
	filterLabels := map[string]string{
		clusterobject.ManagedByLabelKey:     clusterobject.OperatorName,
		clusterobject.ServiceBindingIDLabel: id,
	}
	secrets, err := a.secretManager.GetAllByLabels(context.Background(), filterLabels)
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
	id := request.PathValue("id")
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
	id := request.PathValue("id")
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

func generateSecretFromSISBData(si *types.ServiceInstance, sb *types.ServiceBinding) (*corev1.Secret, error) {
	slicedUUID := strings.Split(uuid.NewString(), "-")
	suffix := strings.Join(slicedUUID[:2], "-")
	secretName := fmt.Sprintf("%s-%s", sb.Name, suffix)

	namespace, err := sb.ContextValueByFieldName(types.ContextNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace from service binding context: %w", err)
	}

	labels := map[string]string{
		clusterobject.ManagedByLabelKey:        clusterobject.OperatorName,
		clusterobject.ServiceBindingIDLabel:    sb.ID,
		clusterobject.ServiceInstanceIDLabel:   si.ID,
		clusterobject.ServiceInstanceNameLabel: si.Name,
	}

	data := map[string]string{}
	if err := json.Unmarshal(sb.Credentials, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials from service binding: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels:    labels,
		},
		StringData: data,
	}

	return secret, nil
}
