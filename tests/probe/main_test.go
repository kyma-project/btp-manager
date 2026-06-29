package main

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	btpv1alpha1 "github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func testScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(s)
	_ = btpv1alpha1.AddToScheme(s)
	return s
}

func TestLoadConfig_Defaults(t *testing.T) {
	cfg := loadConfig()
	assert.Equal(t, "kyma-system", cfg.Namespace)
	assert.Equal(t, "btpoperator", cfg.BtpOperatorName)
	assert.Equal(t, "sap-btp-manager", cfg.TLSSecret)
	assert.Equal(t, "", cfg.TokenURLOverride)
}

func TestLoadConfig_EnvOverride(t *testing.T) {
	os.Setenv("PROBE_NAMESPACE", "test-ns")
	os.Setenv("PROBE_TOKENURL_OVERRIDE", "https://fake.local/health")
	defer func() {
		os.Unsetenv("PROBE_NAMESPACE")
		os.Unsetenv("PROBE_TOKENURL_OVERRIDE")
	}()
	cfg := loadConfig()
	assert.Equal(t, "test-ns", cfg.Namespace)
	assert.Equal(t, "https://fake.local/health", cfg.TokenURLOverride)
}

func TestCollectMount_Present(t *testing.T) {
	caFile, err := os.CreateTemp("", "ca-*.crt")
	require.NoError(t, err)
	content := []byte("-----BEGIN CERTIFICATE-----\nfake\n-----END CERTIFICATE-----\n")
	_, err = caFile.Write(content)
	require.NoError(t, err)
	caFile.Close()
	defer os.Remove(caFile.Name())

	// Write a fake mountinfo that lists the CA file's directory as a mountpoint
	mntDir := strings.TrimSuffix(caFile.Name(), "/"+strings.Split(caFile.Name(), "/")[len(strings.Split(caFile.Name(), "/"))-1])
	mountInfo, err := os.CreateTemp("", "mountinfo-*")
	require.NoError(t, err)
	defer os.Remove(mountInfo.Name())
	fmt.Fprintf(mountInfo, "100 99 0:1 / %s rw - tmpfs tmpfs rw\n", mntDir)
	mountInfo.Close()

	m := collectMountFromPath(caFile.Name(), mntDir, mountInfo.Name())
	assert.True(t, m.Present)
	assert.NotEmpty(t, m.Hash)
	assert.Equal(t, content, m.Content)
}

func TestCollectMount_Absent_NoMount(t *testing.T) {
	// mountinfo does not list /etc/ssl/certs → no injected bundle
	mountInfo, err := os.CreateTemp("", "mountinfo-*")
	require.NoError(t, err)
	defer os.Remove(mountInfo.Name())
	fmt.Fprintf(mountInfo, "100 99 0:1 / / rw - overlay overlay rw\n")
	mountInfo.Close()

	m := collectMountFromPath("/etc/ssl/certs/ca-certificates.crt", "/etc/ssl/certs", mountInfo.Name())
	assert.False(t, m.Present)
	assert.Empty(t, m.Hash)
	assert.Nil(t, m.Content)
}

func TestDialTLS_OK(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()
	pool := srv.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs
	assert.Equal(t, tlsResultOK, dialTLS(srv.Listener.Addr().String(), pool))
}

func TestDialTLS_X509(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	// Empty pool → certificate signed by unknown authority
	assert.Equal(t, tlsResultX509, dialTLS(srv.Listener.Addr().String(), x509.NewCertPool()))
}

func TestDialTLS_Other(t *testing.T) {
	// Nothing listening on this port → connection refused → failed-other
	assert.Equal(t, tlsResultOther, dialTLS("127.0.0.1:19999", x509.NewCertPool()))
}

func TestBuildCertPool_NoMount(t *testing.T) {
	pool := buildCertPool(mountSignal{Present: false})
	assert.NotNil(t, pool)
}

func TestBuildCertPool_WithMount_CustomOnly(t *testing.T) {
	pem := []byte("-----BEGIN CERTIFICATE-----\nfake\n-----END CERTIFICATE-----\n")
	pool := buildCertPool(mountSignal{Present: true, Content: pem})
	assert.NotNil(t, pool)
}

func TestComputeSignal(t *testing.T) {
	cases := []struct {
		mount    bool
		tls      string
		expected string
	}{
		{false, tlsResultOK, signalNone},
		{false, tlsResultX509, signalWarning},
		{false, tlsResultOther, signalWarning},
		{true, tlsResultOK, signalNone},
		{true, tlsResultX509, signalAlert},
		{true, tlsResultOther, signalWarning},
	}
	for _, c := range cases {
		assert.Equal(t, c.expected, computeSignal(c.mount, c.tls), "mount=%v tls=%s", c.mount, c.tls)
	}
}

func TestPatchBtpOperatorAnnotations(t *testing.T) {
	cr := &btpv1alpha1.BtpOperator{
		ObjectMeta: metav1.ObjectMeta{Name: "btpoperator", Namespace: "kyma-system"},
		Status:     btpv1alpha1.Status{State: btpv1alpha1.StateReady},
	}
	cl := fake.NewClientBuilder().WithScheme(testScheme()).WithObjects(cr).Build()

	cfg := config{Namespace: "kyma-system", BtpOperatorName: "btpoperator"}
	err := patchBtpOperatorAnnotations(context.Background(), cl, cfg, map[string]string{
		"tls-probe-mount":      "true",
		"tls-probe-tls-result": tlsResultOK,
		"tls-probe-hash":       "abc123",
		"tls-probe-signal":     signalNone,
	})
	require.NoError(t, err)

	updated := &btpv1alpha1.BtpOperator{}
	require.NoError(t, cl.Get(context.Background(), types.NamespacedName{Namespace: "kyma-system", Name: "btpoperator"}, updated))
	assert.Equal(t, "true", updated.Annotations["tls-probe-mount"])
	assert.Equal(t, tlsResultOK, updated.Annotations["tls-probe-tls-result"])
	assert.Equal(t, "abc123", updated.Annotations["tls-probe-hash"])
	assert.Equal(t, signalNone, updated.Annotations["tls-probe-signal"])
}
