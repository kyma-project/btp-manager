package servicemanager

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/kyma-project/btp-manager/internal/service-manager/types"
)

const (
	defaultDir               = "service-manager"
	rootDir                  = "btp-manager"
	serviceOfferingsJSONPath = "testdata/service_offerings.json"
	servicePlansJSONPath     = "testdata/service_plans.json"
	serviceInstancesJSONPath = "testdata/service_instances.json"
	serviceBindingsJSONPath  = "testdata/service_bindings.json"
)

func NewFakeServer() (*httptest.Server, error) {
	smHandler, err := newFakeSMHandler()
	if err != nil {
		return nil, fmt.Errorf("while creating new fake SM handler: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/service_offerings", smHandler.getServiceOfferings)
	mux.HandleFunc("GET /v1/service_offerings/{serviceOfferingID}", smHandler.getServiceOffering)
	mux.HandleFunc("GET /v1/service_plans", smHandler.getServicePlans)
	mux.HandleFunc("GET /v1/service_plans/{servicePlanID}", smHandler.getServicePlan)
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
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	currentDir := filepath.Base(wd)
	if currentDir != defaultDir {
		// change working directory to service manager directory to read JSON files
		if err := setDefaultDir(); err != nil {
			return nil, fmt.Errorf("while setting default fake service manager directory: %w", err)
		}
	}

	sos, err := GetServiceOfferingsFromJSON()
	if err != nil {
		return nil, fmt.Errorf("while getting service offerings from JSON: %w", err)

	}
	plans, err := GetServicePlansFromJSON()
	if err != nil {
		return nil, fmt.Errorf("while getting service plans from JSON: %w", err)
	}
	sis, err := GetServiceInstancesFromJSON()
	if err != nil {
		return nil, fmt.Errorf("while getting service instances from JSON: %w", err)

	}
	sbs, err := GetServiceBindingsFromJSON()
	if err != nil {
		return nil, fmt.Errorf("while getting service bindings from JSON: %w", err)

	}
	// restore working directory
	if err = os.Chdir(wd); err != nil {
		return nil, err
	}
	return &fakeSMHandler{serviceOfferings: sos, servicePlans: plans, serviceInstances: sis, serviceBindings: sbs}, nil
}

func setDefaultDir() error {
	if err := setRepoRootDir(); err != nil {
		return err
	}
	if err := os.Chdir("internal/" + defaultDir); err != nil {
		return err
	}
	return nil
}

func setRepoRootDir() error {
	currentPath, err := os.Getwd()
	if err != nil {
		return err
	}
	currentDir := filepath.Base(currentPath)
	if currentDir != rootDir {
		err = os.Chdir("..")
		if err != nil {
			return err

		}
		err := setRepoRootDir()
		if err != nil {
			return err
		}
	}
	return nil
}

func GetServiceOfferingsFromJSON() (*types.ServiceOfferings, error) {
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

func GetServicePlansFromJSON() (*types.ServicePlans, error) {
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

func GetServiceInstancesFromJSON() (*types.ServiceInstances, error) {
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

func GetServiceBindingsFromJSON() (*types.ServiceBindings, error) {
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
	prefixedSoID := values.Get(URLFieldQueryKey)
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

func (h *fakeSMHandler) getServicePlan(w http.ResponseWriter, r *http.Request) {
	planID := r.PathValue("servicePlanID")
	if len(planID) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var err error
	data := make([]byte, 0)
	for _, p := range h.servicePlans.Items {
		if p.ID == planID {
			data, err = json.Marshal(p)
			if err != nil {
				log.Println("error while marshalling plan data: %w", err)
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
			Description: "could not find such plan",
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
		log.Println("error while writing plan data: %w", err)
		return
	}
}
