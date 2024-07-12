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
	serviceBindingsJSONPath  = "testdata/service_bindings.json"
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

	defaultServiceOfferings, err := getServiceOfferingsFromJSON()
	require.NoError(t, err)
	defaultServicePlans, err := getServicePlansFromJSON()
	require.NoError(t, err)
	defaultServiceInstances, err := getServiceInstancesFromJSON()
	require.NoError(t, err)
	defaultServiceBindings, err := getServiceBindingsFromJSON()
	require.NoError(t, err)

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
		assertEqualServiceOfferings(t, defaultServiceOfferings, sos)
	})

	t.Run("should get service offering details and plans for given service offering ID", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)
		soID := "fc26622b-aeb2-4f3c-95da-8eb337a26883"
		expectedServiceOffering := getServiceOfferingByID(defaultServiceOfferings, soID)
		filteredServicePlans := filterServicePlansByServiceOfferingID(defaultServicePlans, soID)

		// when
		sod, err := smClient.ServiceOfferingDetails(soID)

		// then
		require.NoError(t, err)
		assert.Len(t, sod.ServicePlans.Items, 3)
		assertEqualServiceOffering(t, *expectedServiceOffering, sod.ServiceOffering)
		assertEqualServicePlans(t, &filteredServicePlans, &sod.ServicePlans)
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
		assertEqualServiceInstances(t, defaultServiceInstances, sis)
	})

	t.Run("should get service instance for given service instance ID", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)
		siID := "c7a604e8-f289-4f61-841f-c6519db8daf2"
		expectedServiceInstance := getServiceInstanceByID(defaultServiceInstances, siID)

		// when
		si, err := smClient.ServiceInstance(siID)

		// then
		require.NoError(t, err)
		assertEqualServiceInstance(t, *expectedServiceInstance, *si)
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

	t.Run("should update service instance except shared field", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)
		siID := "f9ffbaa4-739a-4a16-ad02-6f2f17a830c5"
		siUpdatedName := "updated-service-instance"
		siUpdatedServicePlanID := "updated-service-plan-id"
		siUpdatedParameters := json.RawMessage(`{"updated-parameter": "updated-value"}`)
		siUpdatedLabels := []types.LabelChange{{Operation: types.AddLabelOperation, Key: "updated-label", Values: []string{"updated-value"}}}
		siUpdateRequest := &types.ServiceInstanceUpdateRequest{
			ID:            &siID,
			Name:          &siUpdatedName,
			ServicePlanID: &siUpdatedServicePlanID,
			Parameters:    &siUpdatedParameters,
			Labels:        siUpdatedLabels,
		}

		// when
		si, err := smClient.UpdateServiceInstance(siUpdateRequest)

		// then
		require.NoError(t, err)
		assert.Equal(t, siID, si.ID)
		assert.Equal(t, siUpdatedName, si.Name)
		assert.Equal(t, siUpdatedServicePlanID, si.ServicePlanID)
		assert.Contains(t, si.Labels, siUpdatedLabels[0].Key)

		var expectedParams, actualParams []byte
		require.NoError(t, siUpdatedParameters.UnmarshalJSON(expectedParams))
		require.NoError(t, si.Parameters.UnmarshalJSON(actualParams))
		assert.Equal(t, expectedParams, actualParams)

		// revert server to default state
		srv, err = initFakeServer()
		require.NoError(t, err)
	})

	t.Run("should update only the shared field in a service instance", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)
		siID := "df28885c-7c5f-46f0-bb75-0ae2dc85ac41"
		siUpdatedShared := true
		siUpdateRequest := &types.ServiceInstanceUpdateRequest{
			ID:     &siID,
			Shared: &siUpdatedShared,
		}

		// when
		si, err := smClient.UpdateServiceInstance(siUpdateRequest)

		// then
		require.NoError(t, err)
		assert.Equal(t, siID, si.ID)
		assert.Equal(t, siUpdatedShared, si.Shared)

		// revert server to default state
		srv, err = initFakeServer()
		require.NoError(t, err)
	})

	t.Run("delete service instance", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)
		siID := "a7e240d6-e348-4fc0-a54c-7b7bfe9b9da6"

		// when
		err := smClient.DeleteServiceInstance(siID)

		// then
		require.NoError(t, err)

		// when
		_, err = smClient.ServiceInstance(siID)

		// then
		require.Error(t, err)

		// revert server to default state
		srv, err = initFakeServer()
		require.NoError(t, err)
	})

	t.Run("should get all service bindings", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)

		// when
		sbs, err := smClient.ServiceBindings()

		// then
		require.NoError(t, err)
		assertEqualServiceBindings(t, defaultServiceBindings, sbs)
	})

	t.Run("should get service binding for given service binding ID", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)
		sbID := "550e8400-e29b-41d4-a716-446655440003"
		expectedServiceBinding := getServiceBindingByID(defaultServiceBindings, sbID)

		// when
		sb, err := smClient.ServiceBinding(sbID)

		// then
		require.NoError(t, err)
		assertEqualServiceBinding(t, *expectedServiceBinding, *sb)
	})

	t.Run("should create service binding", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)
		sbCreateRequest := &types.ServiceBinding{
			Common: types.Common{
				Name:   "test-service-binding",
				Labels: types.Labels{"test-label": []string{"test-value"}},
			},
			ServiceInstanceID: "test-service-instance-id",
			Parameters:        json.RawMessage(`{"test-parameter": "test-value"}`),
			BindResource:      json.RawMessage(`{"test-bind-resource": "test-value"}`),
		}

		// when
		sb, err := smClient.CreateServiceBinding(sbCreateRequest)

		// then
		require.NoError(t, err)
		assert.NotEmpty(t, sb.ID)
		assert.Equal(t, sbCreateRequest.Name, sb.Name)
		assert.Equal(t, sbCreateRequest.ServiceInstanceID, sb.ServiceInstanceID)
		assert.Equal(t, sbCreateRequest.Labels, sb.Labels)

		var expectedParams, actualParams []byte
		require.NoError(t, sbCreateRequest.Parameters.UnmarshalJSON(expectedParams))
		require.NoError(t, sb.Parameters.UnmarshalJSON(actualParams))
		assert.Equal(t, expectedParams, actualParams)

		var expectedBindResource, actualBindResource []byte
		require.NoError(t, sbCreateRequest.Parameters.UnmarshalJSON(expectedBindResource))
		require.NoError(t, sb.Parameters.UnmarshalJSON(actualBindResource))
		assert.Equal(t, expectedBindResource, actualBindResource)
	})

	t.Run("should delete service binding", func(t *testing.T) {
		// given
		ctx := context.TODO()
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)
		sbID := "318a16c3-7c80-485f-b55c-918629012c9a"

		// when
		err := smClient.DeleteServiceBinding(sbID)

		// then
		require.NoError(t, err)

		// when
		_, err = smClient.ServiceBinding(sbID)

		// then
		require.Error(t, err)

		// revert server to default state
		srv, err = initFakeServer()
		require.NoError(t, err)
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
	mux.HandleFunc("PATCH /v1/service_instances/{serviceInstanceID}", smHandler.updateServiceInstance)
	mux.HandleFunc("DELETE /v1/service_instances/{serviceInstanceID}", smHandler.deleteServiceInstance)
	mux.HandleFunc("GET /v1/service_bindings", smHandler.getServiceBindings)
	mux.HandleFunc("GET /v1/service_bindings/{serviceBindingID}", smHandler.getServiceBinding)
	mux.HandleFunc("POST /v1/service_bindings", smHandler.createServiceBinding)
	mux.HandleFunc("DELETE /v1/service_bindings/{serviceBindingID}", smHandler.deleteServiceBinding)

	srv := httptest.NewUnstartedServer(mux)

	return srv, nil
}

type fakeSMHandler struct {
	serviceOfferings *types.ServiceOfferings
	servicePlans     *types.ServicePlans
	serviceInstances *types.ServiceInstances
	serviceBindings  *types.ServiceBindings
}

func newFakeSMHandler() (*fakeSMHandler, error) {
	sos, err := getServiceOfferingsFromJSON()
	if err != nil {
		return nil, fmt.Errorf("while getting service offerings from JSON: %w", err)

	}
	plans, err := getServicePlansFromJSON()
	if err != nil {
		return nil, fmt.Errorf("while getting service plans from JSON: %w", err)
	}
	sis, err := getServiceInstancesFromJSON()
	if err != nil {
		return nil, fmt.Errorf("while getting service instances from JSON: %w", err)

	}
	sbs, err := getServiceBindingsFromJSON()
	if err != nil {
		return nil, fmt.Errorf("while getting service bindings from JSON: %w", err)

	}
	return &fakeSMHandler{serviceOfferings: sos, servicePlans: plans, serviceInstances: sis, serviceBindings: sbs}, nil
}

func getServiceOfferingsFromJSON() (*types.ServiceOfferings, error) {
	var sos types.ServiceOfferings
	f, err := os.Open(serviceOfferingsJSONPath)
	defer f.Close()
	if err != nil {
		return nil, fmt.Errorf("while reading resources from JSON file: %w", err)
	}

	d := json.NewDecoder(f)
	if err := d.Decode(&sos); err != nil {
		return nil, fmt.Errorf("while decoding resources JSON: %w", err)
	}
	return &sos, nil
}

func getServicePlansFromJSON() (*types.ServicePlans, error) {
	var sps types.ServicePlans
	f, err := os.Open(servicePlansJSONPath)
	defer f.Close()
	if err != nil {
		return nil, fmt.Errorf("while reading resources from JSON file: %w", err)
	}

	d := json.NewDecoder(f)
	if err := d.Decode(&sps); err != nil {
		return nil, fmt.Errorf("while decoding resources JSON: %w", err)
	}

	return &sps, nil
}

func getServiceInstancesFromJSON() (*types.ServiceInstances, error) {
	var sis types.ServiceInstances
	f, err := os.Open(serviceInstancesJSONPath)
	defer f.Close()
	if err != nil {
		return nil, fmt.Errorf("while reading resources from JSON file: %w", err)
	}

	d := json.NewDecoder(f)
	if err := d.Decode(&sis); err != nil {
		return nil, fmt.Errorf("while decoding resources JSON: %w", err)
	}

	return &sis, nil
}

func getServiceBindingsFromJSON() (*types.ServiceBindings, error) {
	var sbs types.ServiceBindings
	f, err := os.Open(serviceBindingsJSONPath)
	defer f.Close()
	if err != nil {
		return nil, fmt.Errorf("while reading resources from JSON file: %w", err)
	}

	d := json.NewDecoder(f)
	if err := d.Decode(&sbs); err != nil {
		return nil, fmt.Errorf("while decoding resources JSON: %w", err)
	}

	return &sbs, nil
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
		w.WriteHeader(http.StatusInternalServerError)
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

	var err error
	data := make([]byte, 0)
	for _, so := range h.serviceOfferings.Items {
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
		w.WriteHeader(http.StatusInternalServerError)
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

	var responseSps types.ServicePlans
	if len(IDFilter) != 0 {
		var filteredSps types.ServicePlans
		for _, sp := range h.servicePlans.Items {
			if sp.ServiceOfferingID == IDFilter {
				filteredSps.Items = append(filteredSps.Items, sp)
			}
		}
		responseSps = filteredSps
	}

	data, err := json.Marshal(responseSps)
	if err != nil {
		log.Println("error while marshalling service plans data: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err = w.Write(data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
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
		w.WriteHeader(http.StatusInternalServerError)
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

	var err error
	data := make([]byte, 0)
	for _, si := range h.serviceInstances.Items {
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
	}
	if len(data) == 0 {
		errResp := types.ErrorResponse{
			ErrorType:   "NotFound",
			Description: "could not find such service_instance",
		}
		data, err = json.Marshal(errResp)
		if err != nil {
			log.Println("error while marshalling error response: %w", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}

	if _, err = w.Write(data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
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
	h.serviceInstances.Items = append(h.serviceInstances.Items, siCreateRequest)

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

func (h *fakeSMHandler) updateServiceInstance(w http.ResponseWriter, r *http.Request) {
	siID := r.PathValue("serviceInstanceID")
	if len(siID) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var siUpdateRequest types.ServiceInstanceUpdateRequest
	err := json.NewDecoder(r.Body).Decode(&siUpdateRequest)
	if err != nil {
		log.Println("error while decoding request body into ServiceInstanceUpdateRequest struct: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var siUpdateResponse *types.ServiceInstance
	for i, si := range h.serviceInstances.Items {
		if si.ID == siID {
			siUpdateResponse = &h.serviceInstances.Items[i]
			if siUpdateRequest.Name != nil {
				h.serviceInstances.Items[i].Name = *siUpdateRequest.Name
			}
			if siUpdateRequest.ServicePlanID != nil {
				h.serviceInstances.Items[i].ServicePlanID = *siUpdateRequest.ServicePlanID
			}
			if siUpdateRequest.Shared != nil {
				h.serviceInstances.Items[i].Shared = *siUpdateRequest.Shared
			}
			if siUpdateRequest.Parameters != nil {
				h.serviceInstances.Items[i].Parameters = *siUpdateRequest.Parameters
			}
			if len(siUpdateRequest.Labels) != 0 {
				for _, labelChange := range siUpdateRequest.Labels {
					if labelChange.Operation == types.AddLabelOperation {
						h.serviceInstances.Items[i].Labels[labelChange.Key] = labelChange.Values
					} else if labelChange.Operation == types.RemoveLabelOperation {
						delete(h.serviceInstances.Items[i].Labels, labelChange.Key)
					}
				}
			}
			break
		}
	}

	data, err := json.Marshal(siUpdateResponse)
	if err != nil {
		log.Println("error while marshalling service instance: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("error while writing service instance data: %w", err)
		return
	}
}

func (h *fakeSMHandler) deleteServiceInstance(w http.ResponseWriter, r *http.Request) {
	siID := r.PathValue("serviceInstanceID")
	if len(siID) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for i, si := range h.serviceInstances.Items {
		if si.ID == siID {
			h.serviceInstances.Items = append(h.serviceInstances.Items[:i], h.serviceInstances.Items[i+1:]...)
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)

	errResp := types.ErrorResponse{
		ErrorType:   "NotFound",
		Description: "could not find such service_instance",
	}

	data, err := json.Marshal(errResp)
	if err != nil {
		log.Println("error while marshalling error response: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err = w.Write(data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("error while writing error response data: %w", err)
		return
	}
}

func (h *fakeSMHandler) getServiceBindings(w http.ResponseWriter, r *http.Request) {
	data, err := json.Marshal(h.serviceBindings)
	if err != nil {
		log.Println("error while marshalling service bindings data: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("error while writing service bindings data: %w", err)
		return
	}
}

func (h *fakeSMHandler) getServiceBinding(w http.ResponseWriter, r *http.Request) {
	sbID := r.PathValue("serviceBindingID")
	if len(sbID) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var err error
	data := make([]byte, 0)
	for _, sb := range h.serviceBindings.Items {
		if sb.ID == sbID {
			data, err = json.Marshal(sb)
			if err != nil {
				log.Println("error while marshalling service binding data: %w", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			break
		}
	}
	if len(data) == 0 {
		errResp := types.ErrorResponse{
			ErrorType:   "NotFound",
			Description: "could not find such service_binding",
		}
		data, err = json.Marshal(errResp)
		if err != nil {
			log.Println("error while marshalling error response: %w", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}

	if _, err = w.Write(data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("error while writing service binding data: %w", err)
		return
	}
}

func (h *fakeSMHandler) createServiceBinding(w http.ResponseWriter, r *http.Request) {
	var sbCreateRequest types.ServiceBinding
	err := json.NewDecoder(r.Body).Decode(&sbCreateRequest)
	if err != nil {
		log.Println("error while decoding request body into Service Binding struct: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sbCreateRequest.ID = uuid.New().String()
	h.serviceBindings.Items = append(h.serviceBindings.Items, sbCreateRequest)

	data, err := json.Marshal(sbCreateRequest)
	if err != nil {
		log.Println("error while marshalling service binding: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if _, err = w.Write(data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("error while writing service binding data: %w", err)
		return
	}
}

func (h *fakeSMHandler) deleteServiceBinding(w http.ResponseWriter, r *http.Request) {
	sbID := r.PathValue("serviceBindingID")
	if len(sbID) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for i, sb := range h.serviceBindings.Items {
		if sb.ID == sbID {
			h.serviceBindings.Items = append(h.serviceBindings.Items[:i], h.serviceBindings.Items[i+1:]...)
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)

	errResp := types.ErrorResponse{
		ErrorType:   "NotFound",
		Description: "could not find such service_binding",
	}

	data, err := json.Marshal(errResp)
	if err != nil {
		log.Println("error while marshalling error response: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err = w.Write(data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("error while writing error response data: %w", err)
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

func getServiceOfferingByID(serviceOfferings *types.ServiceOfferings, serviceOfferingID string) *types.ServiceOffering {
	for _, so := range serviceOfferings.Items {
		if so.ID == serviceOfferingID {
			return &so
		}
	}
	return nil
}

func getServiceInstanceByID(serviceInstances *types.ServiceInstances, serviceInstanceID string) *types.ServiceInstance {
	for _, si := range serviceInstances.Items {
		if si.ID == serviceInstanceID {
			return &si
		}
	}
	return nil
}

func getServiceBindingByID(serviceBindings *types.ServiceBindings, serviceBindingID string) *types.ServiceBinding {
	for _, sb := range serviceBindings.Items {
		if sb.ID == serviceBindingID {
			return &sb
		}
	}
	return nil
}

func filterServicePlansByServiceOfferingID(servicePlans *types.ServicePlans, serviceOfferingID string) types.ServicePlans {
	var filteredSp types.ServicePlans
	for _, sp := range servicePlans.Items {
		if sp.ServiceOfferingID == serviceOfferingID {
			filteredSp.Items = append(filteredSp.Items, sp)
		}
	}
	return filteredSp
}

func assertEqualServiceOfferings(t *testing.T, expected, actual *types.ServiceOfferings) {
	assert.Len(t, actual.Items, len(expected.Items))
	for i := 0; i < len(expected.Items); i++ {
		expectedToCompare, actualToCompare := expected.Items[i], actual.Items[i]
		assertEqualServiceOffering(t, expectedToCompare, actualToCompare)
	}
}

func assertEqualServiceOffering(t *testing.T, expectedToCompare types.ServiceOffering, actualToCompare types.ServiceOffering) {
	var expectedBuff, actualBuff []byte
	require.NoError(t, expectedToCompare.Metadata.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.Metadata.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.Metadata, actualToCompare.Metadata = nil, nil

	require.NoError(t, expectedToCompare.Tags.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.Tags.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.Tags, actualToCompare.Tags = nil, nil

	assert.Equal(t, expectedToCompare, actualToCompare)
}

func assertEqualServicePlans(t *testing.T, expected, actual *types.ServicePlans) {
	assert.Len(t, actual.Items, len(expected.Items))
	for i := 0; i < len(expected.Items); i++ {
		expectedToCompare, actualToCompare := expected.Items[i], actual.Items[i]
		assertEqualServicePlan(t, expectedToCompare, actualToCompare)
	}
}

func assertEqualServicePlan(t *testing.T, expectedToCompare types.ServicePlan, actualToCompare types.ServicePlan) {
	var expectedBuff, actualBuff []byte
	require.NoError(t, expectedToCompare.Metadata.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.Metadata.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.Metadata, actualToCompare.Metadata = nil, nil

	require.NoError(t, expectedToCompare.Schemas.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.Schemas.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.Schemas, actualToCompare.Schemas = nil, nil

	assert.Equal(t, expectedToCompare, actualToCompare)
}

func assertEqualServiceInstances(t *testing.T, expected, actual *types.ServiceInstances) {
	assert.Len(t, actual.Items, len(expected.Items))
	for i := 0; i < len(expected.Items); i++ {
		expectedToCompare, actualToCompare := expected.Items[i], actual.Items[i]
		assertEqualServiceInstance(t, expectedToCompare, actualToCompare)
	}
}

func assertEqualServiceInstance(t *testing.T, expectedToCompare types.ServiceInstance, actualToCompare types.ServiceInstance) {
	var expectedBuff, actualBuff []byte
	require.NoError(t, expectedToCompare.Parameters.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.Parameters.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.Parameters, actualToCompare.Parameters = nil, nil

	require.NoError(t, expectedToCompare.MaintenanceInfo.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.MaintenanceInfo.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.MaintenanceInfo, actualToCompare.MaintenanceInfo = nil, nil

	require.NoError(t, expectedToCompare.Context.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.Context.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.Context, actualToCompare.Context = nil, nil

	require.NoError(t, expectedToCompare.PreviousValues.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.PreviousValues.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.PreviousValues, actualToCompare.PreviousValues = nil, nil

	assert.Equal(t, expectedToCompare, actualToCompare)
}

func assertEqualServiceBindings(t *testing.T, expected, actual *types.ServiceBindings) {
	assert.Len(t, actual.Items, len(expected.Items))
	for i := 0; i < len(expected.Items); i++ {
		expectedToCompare, actualToCompare := expected.Items[i], actual.Items[i]
		assertEqualServiceBinding(t, expectedToCompare, actualToCompare)
	}
}

func assertEqualServiceBinding(t *testing.T, expectedToCompare, actualToCompare types.ServiceBinding) {
	var expectedBuff, actualBuff []byte
	require.NoError(t, expectedToCompare.Credentials.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.Credentials.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.Credentials, actualToCompare.Credentials = nil, nil

	require.NoError(t, expectedToCompare.VolumeMounts.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.VolumeMounts.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.VolumeMounts, actualToCompare.VolumeMounts = nil, nil

	require.NoError(t, expectedToCompare.Endpoints.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.Endpoints.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.Endpoints, actualToCompare.Endpoints = nil, nil

	require.NoError(t, expectedToCompare.Context.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.Context.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.Context, actualToCompare.Context = nil, nil

	require.NoError(t, expectedToCompare.Parameters.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.Parameters.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.Parameters, actualToCompare.Parameters = nil, nil

	require.NoError(t, expectedToCompare.BindResource.UnmarshalJSON(expectedBuff))
	require.NoError(t, actualToCompare.BindResource.UnmarshalJSON(actualBuff))
	assert.Equal(t, expectedBuff, actualBuff)
	expectedToCompare.BindResource, actualToCompare.BindResource = nil, nil

	assert.Equal(t, expectedToCompare, actualToCompare)
}
