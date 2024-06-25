package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kyma-project/btp-manager/internal/api/requests"
	"github.com/kyma-project/btp-manager/internal/api/responses"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

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
	serviceManager *servicemanager.Client
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
		serviceManager: serviceManager,
		secretProvider: secretProvider,
		frontendFS:     fs,
		logger:         slog.Default()}
}

func (a *API) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/secrets", a.ListSecrets)
	mux.HandleFunc("GET /api/service-instances", a.ListServiceInstances)
	mux.HandleFunc("PUT /api/service-instance/{id}", a.CreateServiceInstance)
	mux.HandleFunc("GET /api/service-instance/{id}", a.GetServiceInstance)
	mux.HandleFunc("GET /api/service-offerings/{namespace}/{name}", a.ListServiceOfferings)
	mux.HandleFunc("GET /api/service-offering/{id}", a.GetServiceOffering)
	mux.HandleFunc("POST /api/service-bindings", a.CreateServiceBindings)
	mux.HandleFunc("GET /api/service-binding/{id}", a.GetServiceBinding)
	mux.Handle("GET /", http.FileServer(a.frontendFS))
	a.server.Handler = mux

	log.Fatal(a.server.ListenAndServe())
}

func (a *API) CreateServiceInstance(writer http.ResponseWriter, request *http.Request) {
	// not implemented in SM
}

func (a *API) GetServiceOffering(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	id := request.PathValue("id")
	details, err := a.serviceManager.ServiceOfferingDetails(id)
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
	err := a.serviceManager.SetForGivenSecret(context.Background(), name, namespace)
	if returnError(writer, err) {
		return
	}
	offerings, err := a.serviceManager.ServiceOfferings()
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
	// not implemented in SM
}

func (a *API) ListServiceInstances(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	// will be taken from SM
}

func (a *API) CreateServiceBindings(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	var sb requests.CreateServiceBinding
	err := json.NewDecoder(request.Body).Decode(&sb)
	if returnError(writer, err) {
		return
	}
	dto := requests.CreateServiceBindingVM(sb)
	err = a.serviceManager.CreateServiceBinding(&dto)
	if returnError(writer, err) {
		return
	}
	returnResponse(writer, []byte{}, err)
}

func (a *API) GetServiceBinding(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	id := request.PathValue("id")
	details, err := a.serviceManager.ServiceBinding(id)
	if returnError(writer, err) {
		return
	}
	response, err := json.Marshal(responses.ToServiceBindingsVM(details))
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
