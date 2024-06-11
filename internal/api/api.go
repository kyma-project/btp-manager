package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"log/slog"

	"github.com/kyma-project/btp-manager/internal/api/vm"
	clusterobject "github.com/kyma-project/btp-manager/internal/cluster-object"
	servicemanager "github.com/kyma-project/btp-manager/internal/service-manager"
)

type API struct {
	serviceManager *servicemanager.Client
	secretProvider *clusterobject.SecretProvider
	slogger        *slog.Logger
}

func NewAPI(serviceManager *servicemanager.Client, secretProvider *clusterobject.SecretProvider) *API {
	slogger := slog.Default()
	return &API{serviceManager: serviceManager, secretProvider: secretProvider, slogger: slogger}
}

func (a *API) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/secrets", a.ListSecrets)
	mux.HandleFunc("GET /api/service-instances", a.ListServiceInstances)
	mux.HandleFunc("PUT /api/service-instance/{id}", a.CreateServiceInstance)
	mux.HandleFunc("GET /api/service-instance/{id}", a.GetServiceInstance)
	mux.HandleFunc("GET /api/service-offerings/{namespace}/{name}", a.ListServiceOfferings)
	mux.HandleFunc("GET /api/service-offering/{id}", a.GetServiceOffering)

	go func() {
		err := http.ListenAndServe(":3006", mux)
		if err != nil {
			a.slogger.Error("failed to Start listening", "error", err)
		}
	}()
}

func (a *API) CreateServiceInstance(writer http.ResponseWriter, request *http.Request) {
	return
}

func (a *API) GetServiceOffering(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	id := request.PathValue("id")
	details, err := a.serviceManager.ServiceOfferingDetails(id)
	if returnError(writer, err) {
		return
	}
	response, err := json.Marshal(vm.ToServiceOfferingDetailsVM(details))
	returnResponse(writer, response, err)
}

func (a *API) GetServiceOfferingDetails(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	serviceOfferingId := request.PathValue("id")
	details, err := a.serviceManager.ServiceOfferingDetails(serviceOfferingId)
	if returnError(writer, err) {
		return
	}
	response, err := json.Marshal(vm.ToServiceOfferingDetailsVM(details))
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
	response, err := json.Marshal(vm.ToServiceOfferingsVM(offerings))
	returnResponse(writer, response, err)
}

func (a *API) ListServiceOfferingsDetails(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	// not implemented in SM
}

func (a *API) ListSecrets(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	secrets, err := a.secretProvider.All(context.Background())
	if returnError(writer, err) {
		return
	}
	response, err := json.Marshal(vm.ToSecretVM(*secrets))
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

func (a *API) setupCors(writer http.ResponseWriter, request *http.Request) {
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
