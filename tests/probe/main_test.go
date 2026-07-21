package main

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	btpv1alpha1 "github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// certA and certB are real self-signed test certificates generated once for these tests.
const certA = `-----BEGIN CERTIFICATE-----
MIIBDjCBtaADAgECAgEBMAoGCCqGSM49BAMCMBExDzANBgNVBAMTBmNlcnQtQTAe
Fw0yNjA3MjExNjQ0MjFaFw0yNjA3MjIxNjQ0MjFaMBExDzANBgNVBAMTBmNlcnQt
QTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABB+lo3VAL3L7VG3Kuyt2W9jxxZcG
DGb35cngmqrfeEmQQ7p9Kz3FPeIbWB1hwAHQmjZZwYip11vcVi3MWXpPH2owCgYI
KoZIzj0EAwIDSAAwRQIhAM2P/q7brtxIrTpAgHf4h7R4O9Wk0pb9vCReK0rgIHAn
AiAJFUxHr4wMzZetkNMlopEqTv4vA2aYvxrIm0qvz6741w==
-----END CERTIFICATE-----
`

const certB = `-----BEGIN CERTIFICATE-----
MIIBDTCBtaADAgECAgEBMAoGCCqGSM49BAMCMBExDzANBgNVBAMTBmNlcnQtQjAe
Fw0yNjA3MjExNjQ0MjFaFw0yNjA3MjIxNjQ0MjFaMBExDzANBgNVBAMTBmNlcnQt
QjBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABEeVJBibdup224H6BuoKm1q+0SWX
hNZyiNO4/Xrnp7ctq69tyfkI5dbDC6VgydgukDif3vhEoeMzU4FGS0T4WxcwCgYI
KoZIzj0EAwIDRwAwRAIgBl0wahzcqJEsdpTFDusRCKK4TJE4egl7Si6kWa/O8c4C
ICpl+vSkNJzbvs8fPhWQyqaZ4BpENpJBSigrJvqdQz5N
-----END CERTIFICATE-----
`

func testScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(s)
	_ = btpv1alpha1.AddToScheme(s)
	return s
}

func TestLoadConfig_Defaults(t *testing.T) {
	cfg := loadConfig()
	assert.Equal(t, "kyma-system", cfg.Namespace)
	assert.Equal(t, "sap-btp-manager", cfg.TLSSecret)
	assert.Equal(t, "", cfg.TokenURLOverride)
}

func TestLoadConfig_EnvOverride(t *testing.T) {
	t.Setenv("PROBE_NAMESPACE", "test-ns")
	t.Setenv("PROBE_TOKENURL_OVERRIDE", "https://fake.local/health")
	cfg := loadConfig()
	assert.Equal(t, "test-ns", cfg.Namespace)
	assert.Equal(t, "https://fake.local/health", cfg.TokenURLOverride)
}

func TestCollectMount_Present(t *testing.T) {
	caFile, err := os.CreateTemp("", "ca-*.crt")
	require.NoError(t, err)
	content := []byte(certA)
	_, err = caFile.Write(content)
	require.NoError(t, err)
	caFile.Close()
	defer os.Remove(caFile.Name())

	// Write a fake mountinfo that lists the CA file's directory as a mountpoint
	mntDir := filepath.Dir(caFile.Name())
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
		{false, tlsResultOK, signalOK},
		{false, tlsResultX509, signalError},
		{false, tlsResultOther, signalError},
		{true, tlsResultOK, signalOK},
		{true, tlsResultX509, signalAlert},
		{true, tlsResultOther, signalError},
	}
	for _, c := range cases {
		assert.Equal(t, c.expected, computeSignal(c.mount, c.tls), "mount=%v tls=%s", c.mount, c.tls)
	}
}

func TestRunProbe_UpdatedAt(t *testing.T) {
	// No mount in test environment, TLS will fail (nothing listening) → signal=error → non-silent path.
	// Verifies that tls-probe-updated-at is written whenever the probe annotates the CR.
	cr := &btpv1alpha1.BtpOperator{
		ObjectMeta: metav1.ObjectMeta{Name: "btpoperator", Namespace: "kyma-system"},
		Status:     btpv1alpha1.Status{State: btpv1alpha1.StateReady},
	}
	cl := fake.NewClientBuilder().WithScheme(testScheme()).WithObjects(cr).Build()
	cfg := config{Namespace: "kyma-system", TokenURLOverride: "https://127.0.0.1:19999/token"}

	before := time.Now().UTC().Truncate(time.Second)
	annotations, err := runProbe(context.Background(), cl, cfg)
	require.NoError(t, err)
	require.NotNil(t, annotations)

	ts, ok := annotations["tls-probe-updated-at"]
	require.True(t, ok, "tls-probe-updated-at missing")
	parsed, err := time.Parse(time.RFC3339, ts)
	require.NoError(t, err)
	assert.False(t, parsed.Before(before), "updated-at should be >= test start time")
}

func TestPatchBtpOperatorAnnotations(t *testing.T) {
	cr := &btpv1alpha1.BtpOperator{
		ObjectMeta: metav1.ObjectMeta{Name: "btpoperator", Namespace: "kyma-system"},
		Status:     btpv1alpha1.Status{State: btpv1alpha1.StateReady},
	}
	cl := fake.NewClientBuilder().WithScheme(testScheme()).WithObjects(cr).Build()

	cfg := config{Namespace: "kyma-system"}
	err := patchBtpOperatorAnnotations(context.Background(), cl, cfg, map[string]string{
		"tls-probe-status": signalOK,
		"tls-probe-hash":   "abc123",
	})
	require.NoError(t, err)

	updated := &btpv1alpha1.BtpOperator{}
	require.NoError(t, cl.Get(context.Background(), types.NamespacedName{Namespace: "kyma-system", Name: "btpoperator"}, updated))
	assert.Equal(t, signalOK, updated.Annotations["tls-probe-status"])
	assert.Equal(t, "abc123", updated.Annotations["tls-probe-hash"])
}

func TestClearBtpOperatorProbeAnnotations_ClearsAll(t *testing.T) {
	cr := &btpv1alpha1.BtpOperator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "btpoperator",
			Namespace: "kyma-system",
			Annotations: map[string]string{
				"tls-probe-status":      "error",
				"tls-probe-hash":        "abc123",
				"tls-probe-updated-at":  "2026-07-09T06:52:48Z",
				"tls-probe-last-hash":   "abc123",
				"some-other-annotation": "keep-me",
			},
		},
		Status: btpv1alpha1.Status{State: btpv1alpha1.StateReady},
	}
	cl := fake.NewClientBuilder().WithScheme(testScheme()).WithObjects(cr).Build()

	cfg := config{Namespace: "kyma-system"}
	require.NoError(t, clearBtpOperatorProbeAnnotations(context.Background(), cl, cfg))

	updated := &btpv1alpha1.BtpOperator{}
	require.NoError(t, cl.Get(context.Background(), types.NamespacedName{Namespace: "kyma-system", Name: "btpoperator"}, updated))
	assert.NotContains(t, updated.Annotations, "tls-probe-status")
	assert.NotContains(t, updated.Annotations, "tls-probe-hash")
	assert.NotContains(t, updated.Annotations, "tls-probe-updated-at")
	assert.NotContains(t, updated.Annotations, "tls-probe-last-hash")
	assert.Equal(t, "keep-me", updated.Annotations["some-other-annotation"])
}

func TestClearBtpOperatorProbeAnnotations_NoOp(t *testing.T) {
	cr := &btpv1alpha1.BtpOperator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "btpoperator",
			Namespace: "kyma-system",
			Annotations: map[string]string{
				"some-other-annotation": "keep-me",
			},
		},
		Status: btpv1alpha1.Status{State: btpv1alpha1.StateReady},
	}
	cl := fake.NewClientBuilder().WithScheme(testScheme()).WithObjects(cr).Build()

	cfg := config{Namespace: "kyma-system"}
	require.NoError(t, clearBtpOperatorProbeAnnotations(context.Background(), cl, cfg))

	updated := &btpv1alpha1.BtpOperator{}
	require.NoError(t, cl.Get(context.Background(), types.NamespacedName{Namespace: "kyma-system", Name: "btpoperator"}, updated))
	assert.Equal(t, "keep-me", updated.Annotations["some-other-annotation"])
}

func TestHashCertBundle_EmptyInput(t *testing.T) {
	assert.Empty(t, hashCertBundle(nil))
	assert.Empty(t, hashCertBundle([]byte{}))
	assert.Empty(t, hashCertBundle([]byte("not a cert")))
}

func TestHashCertBundle_SingleCert_Stable(t *testing.T) {
	data := []byte(certA)
	h1 := hashCertBundle(data)
	h2 := hashCertBundle(data)
	require.NotEmpty(t, h1)
	assert.Equal(t, h1, h2)
}

func TestHashCertBundle_OrderIndependent(t *testing.T) {
	// Same two certs, different order — hash must be identical.
	bundleAB := []byte(certA + certB)
	bundleBA := []byte(certB + certA)
	hAB := hashCertBundle(bundleAB)
	hBA := hashCertBundle(bundleBA)
	require.NotEmpty(t, hAB)
	assert.Equal(t, hAB, hBA, "hash must not depend on cert order in the file")
}

func TestHashCertBundle_DifferentCerts_DifferentHash(t *testing.T) {
	hA := hashCertBundle([]byte(certA))
	hB := hashCertBundle([]byte(certB))
	require.NotEmpty(t, hA)
	require.NotEmpty(t, hB)
	assert.NotEqual(t, hA, hB)
}

func TestHashCertBundle_DuplicateCerts_SameAsUnique(t *testing.T) {
	// A bundle with A+A should hash the same as a bundle with just A.
	hOnce := hashCertBundle([]byte(certA))
	hTwice := hashCertBundle([]byte(certA + certA))
	assert.Equal(t, hOnce, hTwice, "duplicated cert must not change the hash")
}

func TestCollectMount_HashIsOrderIndependent(t *testing.T) {
	// Integration: collectMountFromPath must produce the same hash regardless of cert order.
	bundleAB := []byte(certA + certB)
	bundleBA := []byte(certB + certA)

	dir := t.TempDir()
	mountInfo, err := os.CreateTemp("", "mountinfo-*")
	require.NoError(t, err)
	defer os.Remove(mountInfo.Name())
	fmt.Fprintf(mountInfo, "100 99 0:1 / %s rw - tmpfs tmpfs rw\n", dir)
	mountInfo.Close()

	caAB := filepath.Join(dir, "ca-ab.crt")
	caBA := filepath.Join(dir, "ca-ba.crt")
	require.NoError(t, os.WriteFile(caAB, bundleAB, 0600))
	require.NoError(t, os.WriteFile(caBA, bundleBA, 0600))

	mAB := collectMountFromPath(caAB, dir, mountInfo.Name())
	mBA := collectMountFromPath(caBA, dir, mountInfo.Name())

	require.True(t, mAB.Present)
	require.True(t, mBA.Present)
	assert.Equal(t, mAB.Hash, mBA.Hash, "mount hash must be order-independent")
}
