package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/kyma-project/btp-manager/internal/api/requests"
	"github.com/kyma-project/btp-manager/internal/api/responses"
	clusterobject "github.com/kyma-project/btp-manager/internal/cluster-object"
	servicemanager "github.com/kyma-project/btp-manager/internal/service-manager"
	"github.com/kyma-project/btp-manager/internal/service-manager/types"
)

type Config struct {
	Port         string        `envconfig:"default=8080"`
	ReadTimeout  time.Duration `envconfig:"default=30s"`
	WriteTimeout time.Duration `envconfig:"default=90s"`
	IdleTimeout  time.Duration `envconfig:"default=120s"`
}

type API struct {
	server         *http.Server
	smClient       *servicemanager.Client
	secretProvider *clusterobject.SecretProvider
	frontendFS     http.FileSystem
	logger         *slog.Logger
}

func NewAPI(cfg Config, serviceManager *servicemanager.Client, secretProvider *clusterobject.SecretProvider, fs http.FileSystem) *API {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
	return &API{
		server:         srv,
		smClient:       serviceManager,
		secretProvider: secretProvider,
		frontendFS:     fs,
		logger:         slog.Default()}
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

func (a *API) CreateServiceInstance(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	csiRequest, err := a.decodeCreateServiceInstanceRequest(request)
	if returnError(writer, err) {
		return
	}
	si := csiRequest.ConvertToServiceInstance()
	createdSI, err := a.smClient.CreateServiceInstance(si)
	if returnError(writer, err) {
		return
	}
	createdSI.ServicePlanName = si.ServicePlanName
	response, err := json.Marshal(responses.ToServiceInstanceVM(createdSI))
	returnResponse(writer, response, err)
}

func (a *API) GetServiceOffering(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	id := request.PathValue("id")
	details, err := a.smClient.ServiceOfferingDetails(id)
	if returnError(writer, err) {
		return
	}
	response, err := json.Marshal(responses.ToServiceOfferingDetailsVM(details))
	returnResponse(writer, response, err)
}

func (a *API) ListServiceOfferings(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	namespace := request.PathValue("namespace")
	name := request.PathValue("name")
	err := a.smClient.SetForGivenSecret(context.Background(), name, namespace)
	if returnError(writer, err) {
		return
	}
	offerings, err := a.smClient.ServiceOfferings()
	if returnError(writer, err) {
		return
	}
	response, err := json.Marshal(responses.ToServiceOfferingsVM(offerings))
	returnResponse(writer, response, err)
}

func (a *API) ListSecrets(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	secrets, err := a.secretProvider.All(context.Background())
	if returnError(writer, err) {
		return
	}
	response, err := json.Marshal(responses.ToSecretVM(*secrets))
	returnResponse(writer, response, err)
}

func (a *API) GetServiceInstance(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	id := request.PathValue("id")
	si, err := a.smClient.ServiceInstanceWithPlanName(id)
	if returnError(writer, err) {
		return
	}
	response, err := json.Marshal(responses.ToServiceInstanceVM(si))
	returnResponse(writer, response, err)
}

func (a *API) ListServiceInstances(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)

	sis, err := a.smClient.ServiceInstances()
	if returnError(writer, err) {
		return
	}
	response, err := json.Marshal(responses.ToServiceInstancesVM(sis))
	returnResponse(writer, response, err)
}

func (a *API) ListServiceBindings(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	sbs, err := a.smClient.ServiceBindings()
	if returnError(writer, err) {
		return
	}
	sbsVM, err := responses.ToServiceBindingsVM(sbs)
	if returnError(writer, err) {
		return
	}
	response, err := json.Marshal(sbsVM)
	returnResponse(writer, response, err)
}

func (a *API) CreateServiceBinding(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	var serviceBindingRequest requests.CreateServiceBinding
	err := json.NewDecoder(request.Body).Decode(&serviceBindingRequest)
	if returnError(writer, err) {
		return
	}
	si, err := a.smClient.ServiceInstance(serviceBindingRequest.ServiceInstanceId)
	if returnError(writer, err) {
		return
	}
	sb, err := requests.ToServiceBinding(serviceBindingRequest, si)
	if returnError(writer, err) {
		return
	}
	createdServiceBinding, err := a.smClient.CreateServiceBinding(&sb)
	if returnError(writer, err) {
		return
	}
	sbVM, err := responses.ToServiceBindingVM(createdServiceBinding)
	if returnError(writer, err) {
		return
	}
	response, err := json.Marshal(sbVM)
	returnResponse(writer, response, err)
}

func (a *API) GetServiceBinding(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	id := request.PathValue("id")
	sb, err := a.smClient.ServiceBinding(id)
	if returnError(writer, err) {
		return
	}
	sbVM, err := responses.ToServiceBindingVM(sb)
	if returnError(writer, err) {
		return
	}
	response, err := json.Marshal(sbVM)
	returnResponse(writer, response, err)
}

func (a *API) DeleteServiceBinding(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	id := request.PathValue("id")
	err := a.smClient.DeleteServiceBinding(id)
	if returnError(writer, err) {
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
	if returnError(writer, err) {
		return
	}
	siuReq.ID = &id
	updatedSI, err := a.smClient.UpdateServiceInstance(siuReq)
	if returnError(writer, err) {
		return
	}
	response, err := json.Marshal(updatedSI)
	returnResponse(writer, response, err)
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
	err := a.smClient.DeleteServiceInstance(id)
	if returnError(writer, err) {
		return
	}
}

func returnResponse(writer http.ResponseWriter, response []byte, err error) {
	if returnError(writer, err) {
		return
	}
	_, err = writer.Write(response)
	if returnError(writer, err) {
		return
	}
}

func returnError(writer http.ResponseWriter, err error) bool {
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, err := writer.Write([]byte(err.Error()))
		if err != nil {
			return true
		}
		return true
	}
	return false
}
