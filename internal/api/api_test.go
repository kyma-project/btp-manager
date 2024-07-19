package api_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/kyma-project/btp-manager/internal/api"
	clusterobject "github.com/kyma-project/btp-manager/internal/cluster-object"
	servicemanager "github.com/kyma-project/btp-manager/internal/service-manager"
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

	btpManagerAPI := api.NewAPI(cfg, fakeSMClient, secretMgr, nil)
	go btpManagerAPI.Start()
}
