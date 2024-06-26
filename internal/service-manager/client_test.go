package servicemanager_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	servicemanager "github.com/kyma-project/btp-manager/internal/service-manager"
	"github.com/kyma-project/btp-manager/internal/service-manager/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	serviceOfferingsJSONPath = "testdata/service_offerings.json"
	servicePlansJSONPath     = "testdata/service_plans.json"
	serviceInstancesJSONPath = "testdata/service_instances.json"
)

func TestClient(t *testing.T) {
	// given
	secretProvider := newFakeSecretProvider()
	secretProvider.AddSecret(defaultSecret())
	srv, err := initFakeServer()
	require.NoError(t, err)

	srv.Start()
	defer srv.Close()
	httpClient := srv.Client()
	url := srv.URL

	allServiceOfferings := getAllServiceOfferingsFromJSON(t)
	allServicePlans := getAllServicePlansFromJSON(t)
	allServiceInstances := getAllServiceInstancesFromJSON(t)

	t.Run("should get service offerings available for the default credentials", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)

		// when
		err = smClient.Defaults(ctx)

		// then
		require.NoError(t, err)

		// given
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)

		// when
		sos, err := smClient.ServiceOfferings()

		// then
		require.NoError(t, err)
		assert.Len(t, sos.Items, 4)
		assert.ElementsMatch(t, allServiceOfferings.Items, sos.Items)
	})

	t.Run("should get service offering details and plans for given service offering ID", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)
		soID := "fc26622b-aeb2-4f3c-95da-8eb337a26883"
		expectedServiceOffering := getServiceOfferingByID(allServiceOfferings, soID)
		filteredServicePlans := filterServicePlansByServiceOfferingID(allServicePlans, soID)

		// when
		sod, err := smClient.ServiceOfferingDetails(soID)

		// then
		require.NoError(t, err)
		assert.Len(t, sod.ServicePlans.Items, 3)
		assert.Equal(t, expectedServiceOffering, sod.ServiceOffering)
		assert.ElementsMatch(t, filteredServicePlans.Items, sod.ServicePlans.Items)
	})

	t.Run("should get all service instances", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)

		// when
		sis, err := smClient.ServiceInstances()

		// then
		require.NoError(t, err)
		assert.Len(t, sis.Items, 4)
		assert.ElementsMatch(t, allServiceInstances.Items, sis.Items)
	})

	t.Run("should get service instance for given service instance ID", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)
		siID := "c7a604e8-f289-4f61-841f-c6519db8daf2"
		expectedServiceInstance := getServiceInstanceByID(allServiceInstances, siID)

		// when
		si, err := smClient.ServiceInstance(siID)

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedServiceInstance, si)
	})

	t.Run("should create service instance", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)
		siCreateRequest := &types.ServiceInstance{
			Common: types.Common{
				Name:   "test-service-instance",
				Labels: types.Labels{"test-label": []string{"test-value"}},
			},
			ServicePlanID: "test-service-plan-id",
			Parameters:    json.RawMessage(`{"test-parameter": "test-value"}`),
		}

		// when
		si, err := smClient.CreateServiceInstance(siCreateRequest)

		// then
		require.NoError(t, err)
		assert.NotEmpty(t, si.ID)
		assert.Equal(t, siCreateRequest.Name, si.Name)
		assert.Equal(t, siCreateRequest.ServicePlanID, si.ServicePlanID)
		assert.Equal(t, siCreateRequest.Labels, si.Labels)

		var expectedParams, actualParams []byte
		require.NoError(t, siCreateRequest.Parameters.UnmarshalJSON(expectedParams))
		require.NoError(t, si.Parameters.UnmarshalJSON(actualParams))
		assert.Equal(t, expectedParams, actualParams)
	})
}

func initFakeServer() (*httptest.Server, error) {
	smHandler, err := newFakeSMHandler()
	if err != nil {
		return nil, fmt.Errorf("while creating new fake SM handler: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/service_offerings", smHandler.getServiceOfferings)
	mux.HandleFunc("GET /v1/service_offerings/{serviceOfferingID}", smHandler.getServiceOffering)
	mux.HandleFunc("GET /v1/service_plans", smHandler.getServicePlans)
	mux.HandleFunc("GET /v1/service_instances", smHandler.getServiceInstances)
	mux.HandleFunc("GET /v1/service_instances/{serviceInstanceID}", smHandler.getServiceInstance)
	mux.HandleFunc("POST /v1/service_instances", smHandler.createServiceInstance)

	srv := httptest.NewUnstartedServer(mux)

	return srv, nil
}

type fakeSMHandler struct {
	serviceOfferings map[string]interface{}
	servicePlans     map[string]interface{}
	serviceInstances map[string]interface{}
}

func newFakeSMHandler() (*fakeSMHandler, error) {
	sos, err := getResourcesFromJSONFile(serviceOfferingsJSONPath)
	if err != nil {
		return nil, fmt.Errorf("while getting service offerings from JSON file: %w", err)
	}
	plans, err := getResourcesFromJSONFile(servicePlansJSONPath)
	if err != nil {
		return nil, fmt.Errorf("while getting service plans from JSON file: %w", err)
	}
	sis, err := getResourcesFromJSONFile(serviceInstancesJSONPath)
	if err != nil {
		return nil, fmt.Errorf("while getting service instances from JSON file: %w", err)
	}

	return &fakeSMHandler{serviceOfferings: sos, servicePlans: plans, serviceInstances: sis}, nil
}

func getResourcesFromJSONFile(jsonFilePath string) (map[string]interface{}, error) {
	var buf map[string]interface{}
	f, err := os.Open(jsonFilePath)
	defer f.Close()
	if err != nil {
		return nil, fmt.Errorf("while reading resources from JSON file: %w", err)
	}

	d := json.NewDecoder(f)
	if err := d.Decode(&buf); err != nil {
		return nil, fmt.Errorf("while decoding resources JSON: %w", err)
	}
	return buf, nil
}

func (h *fakeSMHandler) getServiceOfferings(w http.ResponseWriter, r *http.Request) {
	data, err := json.Marshal(h.serviceOfferings)
	if err != nil {
		log.Println("error while marshalling service offerings data: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(data); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("error while writing service offerings data: %w", err)
		return
	}
}

func (h *fakeSMHandler) getServiceOffering(w http.ResponseWriter, r *http.Request) {
	soID := r.PathValue("serviceOfferingID")
	if len(soID) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	data, err := json.Marshal(h.serviceOfferings)
	if err != nil {
		log.Println("error while marshalling service offerings data: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var sos types.ServiceOfferings
	if err := json.Unmarshal(data, &sos); err != nil {
		log.Println("error while unmarshalling service offerings data to struct: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data = make([]byte, 0)
	for _, so := range sos.Items {
		if so.ID == soID {
			data, err = json.Marshal(so)
			if err != nil {
				log.Println("error while marshalling service offering data: %w", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			break
		}
		w.WriteHeader(http.StatusNotFound)
	}

	if _, err = w.Write(data); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("error while writing service offerings data: %w", err)
		return
	}
}

func (h *fakeSMHandler) getServicePlans(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	prefixedSoID := values.Get(servicemanager.URLFieldQueryKey)
	IDFilter := ""
	if len(prefixedSoID) != 0 {
		fields := strings.Fields(prefixedSoID)
		IDFilter = strings.Trim(fields[2], "'")
	}

	data, err := json.Marshal(h.servicePlans)
	if err != nil {
		log.Println("error while marshalling service plans data: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var responseSps types.ServicePlans
	if err := json.Unmarshal(data, &responseSps); err != nil {
		log.Println("error while unmarshalling service offerings data to struct: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(IDFilter) != 0 {
		var filteredSps types.ServicePlans
		for _, sp := range responseSps.Items {
			if sp.ServiceOfferingID == IDFilter {
				filteredSps.Items = append(filteredSps.Items, sp)
			}
		}
		responseSps = filteredSps
	}

	data = make([]byte, 0)
	data, err = json.Marshal(responseSps)
	if err != nil {
		log.Println("error while marshalling service plans data: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err = w.Write(data); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("error while writing service plans data: %w", err)
		return
	}
}

func (h *fakeSMHandler) getServiceInstances(w http.ResponseWriter, r *http.Request) {
	data, err := json.Marshal(h.serviceInstances)
	if err != nil {
		log.Println("error while marshalling service instances data: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(data); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("error while writing service instances data: %w", err)
		return
	}
}

func (h *fakeSMHandler) getServiceInstance(w http.ResponseWriter, r *http.Request) {
	siID := r.PathValue("serviceInstanceID")
	if len(siID) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	data, err := json.Marshal(h.serviceInstances)
	if err != nil {
		log.Println("error while marshalling service instances data: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var sis types.ServiceInstances
	if err := json.Unmarshal(data, &sis); err != nil {
		log.Println("error while unmarshalling service instances data to struct: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data = make([]byte, 0)
	for _, si := range sis.Items {
		if si.ID == siID {
			data, err = json.Marshal(si)
			if err != nil {
				log.Println("error while marshalling service instance data: %w", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			break
		}
		w.WriteHeader(http.StatusNotFound)
	}

	if _, err = w.Write(data); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("error while writing service instance data: %w", err)
		return
	}
}

func (h *fakeSMHandler) createServiceInstance(w http.ResponseWriter, r *http.Request) {
	var siCreateRequest types.ServiceInstance
	err := json.NewDecoder(r.Body).Decode(&siCreateRequest)
	if err != nil {
		log.Println("error while decoding request body into Service Instance struct: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	siCreateRequest.ID = uuid.New().String()

	data, err := json.Marshal(siCreateRequest)
	if err != nil {
		log.Println("error while marshalling service instance: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if _, err = w.Write(data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("error while writing service instance data: %w", err)
		return
	}
}

type fakeSecretProvider struct {
	secrets []*corev1.Secret
}

func newFakeSecretProvider() *fakeSecretProvider {
	return &fakeSecretProvider{secrets: make([]*corev1.Secret, 0)}
}

func (p *fakeSecretProvider) AddSecret(secret *corev1.Secret) {
	p.secrets = append(p.secrets, secret)
}

func (p *fakeSecretProvider) GetByNameAndNamespace(ctx context.Context, name, namespace string) (*corev1.Secret, error) {
	for _, secret := range p.secrets {
		if secret.Name == name && secret.Namespace == namespace {
			return secret, nil
		}
	}
	return nil, fmt.Errorf("secret not found")
}

func (p *fakeSecretProvider) clean() {
	p.secrets = make([]*corev1.Secret, 0)
}

func defaultSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sap-btp-service-operator",
			Namespace: "kyma-system",
		},
		StringData: map[string]string{
			"clientid":       "default-client-id",
			"clientsecret":   "default-client-secret",
			"sm_url":         "https://default-sm-url.local",
			"tokenurl":       "https://default-token-url.local",
			"tokenurlsuffix": "/oauth/token",
		},
	}
}

func getAllServiceOfferingsFromJSON(t *testing.T) types.ServiceOfferings {
	var allSos types.ServiceOfferings
	soJSON, err := getResourcesFromJSONFile(serviceOfferingsJSONPath)
	require.NoError(t, err)

	soBytes, err := json.Marshal(soJSON)
	require.NoError(t, err)

	err = json.Unmarshal(soBytes, &allSos)
	require.NoError(t, err)

	return allSos
}

func getAllServicePlansFromJSON(t *testing.T) types.ServicePlans {
	var allSps types.ServicePlans
	spsJSON, err := getResourcesFromJSONFile(servicePlansJSONPath)
	require.NoError(t, err)

	spBytes, err := json.Marshal(spsJSON)
	require.NoError(t, err)

	err = json.Unmarshal(spBytes, &allSps)
	require.NoError(t, err)

	return allSps
}

func getAllServiceInstancesFromJSON(t *testing.T) types.ServiceInstances {
	var allSis types.ServiceInstances
	sisJSON, err := getResourcesFromJSONFile(serviceInstancesJSONPath)
	require.NoError(t, err)

	sisBytes, err := json.Marshal(sisJSON)
	require.NoError(t, err)

	err = json.Unmarshal(sisBytes, &allSis)
	require.NoError(t, err)

	return allSis
}

func getServiceOfferingByID(serviceOfferings types.ServiceOfferings, serviceOfferingID string) types.ServiceOffering {
	for _, so := range serviceOfferings.Items {
		if so.ID == serviceOfferingID {
			return so
		}
	}
	return types.ServiceOffering{}
}

func getServiceInstanceByID(serviceInstances types.ServiceInstances, serviceInstanceID string) *types.ServiceInstance {
	for _, si := range serviceInstances.Items {
		if si.ID == serviceInstanceID {
			return &si
		}
	}
	return nil
}

func filterServicePlansByServiceOfferingID(servicePlans types.ServicePlans, serviceOfferingID string) types.ServicePlans {
	var filteredSp types.ServicePlans
	for _, sp := range servicePlans.Items {
		if sp.ServiceOfferingID == serviceOfferingID {
			filteredSp.Items = append(filteredSp.Items, sp)
		}
	}
	return filteredSp
}
