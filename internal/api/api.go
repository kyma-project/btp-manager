package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"log/slog"

	"github.com/kyma-project/btp-manager/internal/api/vm"
	servicemanager "github.com/kyma-project/btp-manager/internal/service-manager"
)

type API struct {
	serviceManager *servicemanager.Client
	slogger        *slog.Logger
}

func NewAPI(serviceManager *servicemanager.Client) *API {
	slogger := slog.Default()
	return &API{serviceManager: serviceManager, slogger: slogger}
}

func (a *API) Start() {
	mux := http.ServeMux{}
	mux.HandleFunc("GET /api/secrets", a.ListSecrets)
	mux.HandleFunc("GET /api/service-instances", a.ListServiceInstances)
	mux.HandleFunc("GET /api/service-instance/{id}", a.GetServiceInstance)
	mux.HandleFunc("GET /api/service-offerings/{namespace}/{name}", a.ListServiceOfferings)
	mux.HandleFunc("GET /api/service-offering/{id}", a.GetServiceOffering)
	go func() {
		err := http.ListenAndServe(":3006", nil)
		if err != nil {
			a.slogger.Error("failed to Start listening", "error", err)
		}
	}()
}

func (a *API) ListServiceOfferings(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	namespace := request.URL.Query().Get("namespace")
	name := request.URL.Query().Get("name")
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

func (a *API) ListSecrets(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(writer, request)
	secrets, err := a.serviceManager.SecretProvider.All(context.Background())
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

func (a *API) GetServiceOffering(writer http.ResponseWriter, request *http.Request) {
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
