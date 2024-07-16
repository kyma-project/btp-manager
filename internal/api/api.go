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

	"github.com/kyma-project/btp-manager/internal/service-manager/types"

	"github.com/kyma-project/btp-manager/internal/api/responses"

	clusterobject "github.com/kyma-project/btp-manager/internal/cluster-object"
	servicemanager "github.com/kyma-project/btp-manager/internal/service-manager"
)

type Config struct {
	Port         string        `envconfig:"default=8080"`
	ReadTimeout  time.Duration `envconfig:"default=30s"`
	WriteTimeout time.Duration `envconfig:"default=90s"`
	IdleTimeout  time.Duration `envconfig:"default=120s"`
}

type API struct {
	server         *http.Server
	smClient       servicemanager.Client
	secretProvider clusterobject.SecretProvider
	frontendFS     http.FileSystem
	logger         *slog.Logger
}

func NewAPI(cfg Config, serviceManager servicemanager.Client, secretProvider clusterobject.SecretProvider, fs http.FileSystem) *API {
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
	router.HandleFunc("GET /api/service-instances", a.ListServiceInstances)
	router.HandleFunc("PUT /api/service-instance/{id}", a.CreateServiceInstance)
	router.HandleFunc("GET /api/service-instance/{id}", a.GetServiceInstance)
	router.HandleFunc("GET /api/service-offerings/{namespace}/{name}", a.ListServiceOfferings)
	router.HandleFunc("GET /api/service-offering/{id}", a.GetServiceOffering)
	router.HandleFunc("POST /api/service-bindings", a.CreateServiceBindings)
	router.HandleFunc("GET /api/service-binding/{id}", a.GetServiceBinding)
	router.Handle("GET /", http.FileServer(a.frontendFS))
	router.Handle("GET /", http.FileServer(a.frontendFS))
}

func (a *API) CreateServiceInstance(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
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
	si, err := a.smClient.ServiceInstance(id)
	if returnError(writer, err) {
		return
	}
	plan, err := a.smClient.ServicePlan(si.ServicePlanID)
	if returnError(writer, err) {
		return
	}
	response, err := json.Marshal(responses.ToServiceInstanceVM(si, plan))
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

func (a *API) CreateServiceBindings(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	var sb types.ServiceBinding
	err := json.NewDecoder(request.Body).Decode(&sb)
	if returnError(writer, err) {
		return
	}
	_, err = a.smClient.CreateServiceBinding(&sb)
	if returnError(writer, err) {
		return
	}
	returnResponse(writer, []byte{}, err)
}

func (a *API) GetServiceBinding(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	id := request.PathValue("id")
	details, err := a.smClient.ServiceBinding(id)
	if returnError(writer, err) {
		return
	}
	response, err := json.Marshal(responses.ToServiceBindingVM(details))
	returnResponse(writer, response, err)
}

func (a *API) setupCors(writer http.ResponseWriter, request *http.Request) {
	a.logger.Info(fmt.Sprintf("api call to -> %s as: %s", request.RequestURI, request.Method))
	origin := request.Header.Get("Origin")
	origin = strings.ReplaceAll(origin, "\r", "")
	origin = strings.ReplaceAll(origin, "\n", "")
	writer.Header().Set("Access-Control-Allow-Origin", origin)
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
