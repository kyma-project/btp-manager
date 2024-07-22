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
		req, err := http.NewRequest(http.MethodGet, apiAddr+"/api/service-instances", nil)
		resp, err := apiClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode)
		defer resp.Body.Close()

		var sis responses.ServiceInstances
		err = json.NewDecoder(resp.Body).Decode(&sis)
		require.NoError(t, err)

		assert.Equal(t, sis.NumItems, 4)
	})
}
