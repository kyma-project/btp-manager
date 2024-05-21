package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/kyma-project/btp-manager/internal/api/vm"
	clusterobject "github.com/kyma-project/btp-manager/internal/cluster-object"
	servicemanager "github.com/kyma-project/btp-manager/internal/service-manager"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type API struct {
	cluster                 *client.Client
	namespaceProvider       *clusterobject.NamespaceProvider
	serviceInstanceProvider *clusterobject.ServiceInstanceProvider
	secretProvider          *clusterobject.SecretProvider
	serviceManager          *servicemanager.Client
}

func NewAPI() (*API, error) {
	k8sCfg, err := config.GetConfig()
	if err != nil {
		panic(err)
	}
	k8s, err := client.New(k8sCfg, client.Options{})
	if err != nil {
		return nil, err
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	namespaceProvider := clusterobject.NewNamespaceProvider(k8s, log)
	serviceInstanceProvider := clusterobject.NewServiceInstanceProvider(k8s, log)
	secretProvider := clusterobject.NewSecretProvider(k8s, namespaceProvider, serviceInstanceProvider, log)
	serviceManager := servicemanager.NewClient(context.Background(), log, secretProvider)

	return &API{
		cluster:                 &k8s,
		namespaceProvider:       namespaceProvider,
		serviceInstanceProvider: serviceInstanceProvider,
		secretProvider:          secretProvider,
		serviceManager:          serviceManager,
	}, nil
}

func (a *API) ListServiceOfferings(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(&writer, request)
	vars := mux.Vars(request)
	namespace := vars["namespace"]
	name := vars["name"]
	a.serviceManager.SetForGivenSecret(context.Background(), name, namespace)
	offerings, err := a.serviceManager.ServiceOfferings()
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	response, err := json.Marshal(vm.ToServiceOfferingsVM(offerings))
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.Write(response)
}

func (a *API) ListServiceInstances(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(&writer, request)
	serviceInstanceProvider, err := a.serviceInstanceProvider.AllWithSecretRef(context.Background())
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	instances := vm.ToServiceInstancesVM(serviceInstanceProvider.Items)
	response, err := json.Marshal(instances)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.Write(response)
}

func (a *API) ListSecrets(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(&writer, request)
	secrets, err := a.secretProvider.All(context.Background())
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	response, err := json.Marshal(vm.ToSecretVM(*secrets))
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.Write(response)
}

func (a *API) GetServiceInstance(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(&writer, request)
}

func (a *API) GetServiceOffering(writer http.ResponseWriter, request *http.Request) {
	a.setupCors(&writer, request)
}

func (a *API) setupCors(w *http.ResponseWriter, req *http.Request) {
	origin := req.Header.Get("Origin")
	origin = strings.ReplaceAll(origin, "\r", "")
	origin = strings.ReplaceAll(origin, "\n", "")
	(*w).Header().Set("Access-Control-Allow-Origin", origin)
}
