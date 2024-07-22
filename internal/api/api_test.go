package api_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/kyma-project/btp-manager/internal/api"
	"github.com/kyma-project/btp-manager/internal/api/responses"
	clusterobject "github.com/kyma-project/btp-manager/internal/cluster-object"
	servicemanager "github.com/kyma-project/btp-manager/internal/service-manager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	port         = "8080"
	readTimeout  = 1 * time.Second
	writeTimeout = 1 * time.Second
	idleTimeout  = 2 * time.Second
)

func TestAPI(t *testing.T) {
	// before all
	cfg := api.Config{
		Port:         port,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}
	defaultSIs := defaultServiceInstances()

	fakeSM, err := servicemanager.NewFakeServer()
	require.NoError(t, err)

	fakeSM.Start()
	defer fakeSM.Close()
	httpClient := fakeSM.Client()
	url := fakeSM.URL

	secretMgr := clusterobject.NewFakeSecretProvider()
	secretMgr.AddSecret(clusterobject.FakeDefaultSecret())

	fakeSMClient := servicemanager.NewClient(context.TODO(), slog.Default(), secretMgr)
	fakeSMClient.SetHTTPClient(httpClient)
	fakeSMClient.SetSMURL(url)

	btpMgrAPI := api.NewAPI(cfg, fakeSMClient, secretMgr, nil)
	apiAddr := "http://localhost" + btpMgrAPI.Address()
	go btpMgrAPI.Start()

	apiClient := http.Client{
		Timeout: 500 * time.Millisecond,
	}

	t.Run("GET Service Instances", func(t *testing.T) {
		// when
		req, err := http.NewRequest(http.MethodGet, apiAddr+"/api/service-instances", nil)
		resp, err := apiClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode)
		defer resp.Body.Close()

		var sis responses.ServiceInstances
		err = json.NewDecoder(resp.Body).Decode(&sis)
		require.NoError(t, err)

		// then
		assert.Equal(t, sis.NumItems, 4)
		assert.ElementsMatch(t, sis.Items, defaultSIs.Items)
	})

	t.Run("GET Service Instance by ID", func(t *testing.T) {
		// given
		siID := "a7e240d6-e348-4fc0-a54c-7b7bfe9b9da6"
		expectedSI := getServiceInstanceByID(defaultSIs, siID)
		expectedSI.ServicePlanID = "4036790e-5ef3-4cf7-bb16-476053477a9a"
		expectedSI.ServicePlanName = "service1-plan2"

		// when
		req, err := http.NewRequest(http.MethodGet, apiAddr+"/api/service-instances/"+siID, nil)
		resp, err := apiClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode)
		defer resp.Body.Close()

		var si responses.ServiceInstance
		err = json.NewDecoder(resp.Body).Decode(&si)
		require.NoError(t, err)

		// then
		assert.Equal(t, expectedSI, si)
	})
}

func defaultServiceInstances() responses.ServiceInstances {
	return responses.ServiceInstances{
		NumItems: 4,
		Items: []responses.ServiceInstance{
			{
				ID:           "f9ffbaa4-739a-4a16-ad02-6f2f17a830c5",
				Name:         "si-test-1",
				Namespace:    "kyma-system",
				SubaccountID: "a4bdee5b-2bc4-4a44-915b-196ae18c7f29",
				ClusterID:    "59c7efc0-d6bc-4d07-87cf-9bd049534afe",
			},
			{
				ID:           "df28885c-7c5f-46f0-bb75-0ae2dc85ac41",
				Name:         "si-test-2",
				Namespace:    "kyma-system",
				SubaccountID: "5ef574ba-5fb3-493f-839c-48b787f2b710",
				ClusterID:    "5dc40d3c-1839-4173-9743-d5b4f36d9d7b",
			},
			{
				ID:           "a7e240d6-e348-4fc0-a54c-7b7bfe9b9da6",
				Name:         "si-test-3",
				Namespace:    "kyma-system",
				SubaccountID: "73b7f0df-6376-4115-8e45-a0e005c0f5d2",
				ClusterID:    "4f6ee6a5-9c28-4e50-8b91-708345e1b607",
			},
			{
				ID:           "c7a604e8-f289-4f61-841f-c6519db8daf2",
				Name:         "si-test-4",
				Namespace:    "kyma-system",
				SubaccountID: "ad4e88f7-e9cc-4346-944a-d9e0dc42a038",
				ClusterID:    "8e0b4ad1-4fa0-4f7f-a6a7-3db2ac0779e2",
			},
		},
	}
}

func getServiceInstanceByID(serviceInstances responses.ServiceInstances, serviceInstanceID string) responses.ServiceInstance {
	for _, si := range serviceInstances.Items {
		if si.ID == serviceInstanceID {
			return si
		}
	}
	return responses.ServiceInstance{}
}
