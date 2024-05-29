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
	"testing"

	servicemanager "github.com/kyma-project/btp-manager/internal/service-manager"
	"github.com/kyma-project/btp-manager/internal/service-manager/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	serviceOfferingsJSONPath = "testdata/service_offerings.json"
)

func TestClient(t *testing.T) {
	// given
	secretProvider := newFakeSecretProvider()
	srv, err := initFakeServer()
	require.NoError(t, err)

	srv.Start()
	defer srv.Close()
	httpClient := srv.Client()
	url := srv.URL

	t.Run("should get service offerings available for the default credentials", func(t *testing.T) {
		// given
		ctx := context.TODO()
		secretProvider.AddSecret(defaultSecret())
		smClient := servicemanager.NewClient(ctx, slog.Default(), secretProvider)

		var expectedServiceOfferings types.ServiceOfferings
		soJSON, err := getResourcesFromJSONFile(serviceOfferingsJSONPath)
		require.NoError(t, err)

		soBytes, err := json.Marshal(soJSON)
		require.NoError(t, err)

		err = json.Unmarshal(soBytes, &expectedServiceOfferings)
		require.NoError(t, err)

		// when
		err = smClient.Defaults(ctx)

		// then
		require.NoError(t, err)

		// given
		smClient.SetHTTPClient(httpClient)
		smClient.SetSMURL(url)

		// when
		so, err := smClient.ServiceOfferings()

		// then
		require.NoError(t, err)
		assert.Len(t, so.ServiceOfferings, 4)
		assert.ElementsMatch(t, expectedServiceOfferings.ServiceOfferings, so.ServiceOfferings)
	})
}

func initFakeServer() (*httptest.Server, error) {
	smHandler, err := newFakeSMHandler()
	if err != nil {
		return nil, fmt.Errorf("while creating new fake SM handler: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/service_offerings", smHandler.getServiceOfferings)

	srv := httptest.NewUnstartedServer(mux)

	return srv, nil
}

type fakeSMHandler struct {
	serviceOfferings map[string]interface{}
}

func newFakeSMHandler() (*fakeSMHandler, error) {
	so, err := getResourcesFromJSONFile(serviceOfferingsJSONPath)
	if err != nil {
		return nil, fmt.Errorf("while getting service offerings from JSON file: %w", err)
	}

	return &fakeSMHandler{serviceOfferings: so}, nil
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
