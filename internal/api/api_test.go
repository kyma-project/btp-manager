package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/kyma-project/btp-manager/internal/cluster-object/automock"
	"github.com/kyma-project/btp-manager/internal/service-manager/automock"
	"github.com/kyma-project/btp-manager/internal/service-manager/types"
	"github.com/kyma-project/btp-manager/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/require"
)

func TestApiResponses(t *testing.T) {

	tests := []struct {
		name           string
		path           string
		body           string
		method         string
		expectedStatus int
		file           string
		items          *types.ServiceInstances
	}{
		{
			name:           "list instances should return all its services",
			file:           "list-instances-happy-expected.json",
			path:           "api/service-instances",
			expectedStatus: http.StatusOK,
			items: &types.ServiceInstances{
				Items: []types.ServiceInstance{
					{
						Common: types.Common{
							ID:          "1",
							Name:        "service-1",
							Description: "",
						},
					},
					{
						Common: types.Common{
							ID:          "2",
							Name:        "service-2",
							Description: "",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			router := http.NewServeMux()

			sm := servicemanager.NewServiceManager(t)
			sm.On("Defaults", mock.Anything).Return(nil)
			sm.On("ServiceInstances").Return(tt.items, nil)

			provider := clusterojbect.NewProvider(t)

			api := NewAPI(Config{}, sm, provider, ui.NewUIStaticFS())
			api.AttachRoutes(router)

			httpServer := httptest.NewServer(router)
			defer httpServer.Close()

			// when
			resp := callAPI(t, httpServer, tt.method, tt.path, tt.body)

			// then
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			got, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			validateJSON(t, got, tt.file)
		})
	}
}

func validateJSON(t *testing.T, got []byte, file string) {
	var prettyWant bytes.Buffer
	want := readJsonFile(t, file)
	if len(want) > 0 {
		err := json.Indent(&prettyWant, []byte(want), "", "  ")
		if err != nil {
			t.Error(err)
			t.Fail()
		}
	}

	var prettyGot bytes.Buffer
	if len(got) > 0 {
		err := json.Indent(&prettyGot, got, "", "  ")
		if err != nil {
			t.Error(err)
			t.Fail()
		}
	}
	if !assert.JSONEq(t, prettyGot.String(), prettyWant.String()) {
		t.Errorf("%v Schema() = \n######### GOT ###########%v\n######### ENDGOT ########, want \n##### WANT #####%v\n##### ENDWANT #####", file, prettyGot.String(), prettyWant.String())
	}
}

func readJsonFile(t *testing.T, file string) string {
	t.Helper()

	filename := path.Join("testdata", file)
	jsonFile, err := os.ReadFile(filename)
	require.NoError(t, err)

	return string(jsonFile)
}

func callAPI(t *testing.T, httpServer *httptest.Server, method string, path string, body string) *http.Response {
	cli := httpServer.Client()
	req, err := http.NewRequest(method, fmt.Sprintf("%s/%s", httpServer.URL, path), bytes.NewBuffer([]byte(body)))
	req.Header.Set("X-Broker-API-Version", "2.15")
	require.NoError(t, err)

	resp, err := cli.Do(req)
	require.NoError(t, err)
	return resp
}
