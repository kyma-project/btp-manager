package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	btpv1alpha1 "github.com/kyma-project/btp-manager/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	caBundlePath    = "/etc/ssl/certs/ca-certificates.crt"
	caBundleMntPath = "/etc/ssl/certs"
	mountInfoPath   = "/proc/self/mountinfo"
	tlsResultOK     = "ok"
	tlsResultX509   = "failed-x509"
	tlsResultOther  = "failed-other"
	signalOK        = "ok"
	signalAlert     = "alert"
	signalError     = "error"
)

type config struct {
	Namespace        string
	TLSSecret        string
	TokenURLOverride string
}

func loadConfig() config {
	return config{
		Namespace:        getEnv("PROBE_NAMESPACE", "kyma-system"),
		TLSSecret:        getEnv("PROBE_TLS_SECRET", "sap-btp-manager"),
		TokenURLOverride: getEnv("PROBE_TOKENURL_OVERRIDE", ""),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type mountSignal struct {
	Present bool
	Hash    string
	Content []byte
}

func collectMount() mountSignal {
	// PROBE_FORCE_HASH bypasses mount detection and file reading entirely.
	// Useful for testing the restart-on-hash-change path on distroless images
	// without modifying the image or filesystem. Set to any hex string; change
	// the value between probe cycles to simulate CA bundle rotation.
	// REMOVE before promoting to a stable release.
	if hash := os.Getenv("PROBE_FORCE_HASH"); hash != "" {
		return mountSignal{Present: true, Hash: hash}
	}
	return collectMountFromPath(caBundlePath, caBundleMntPath, mountInfoPath)
}

// collectMountFromPath is the testable inner implementation.
// It considers the CA bundle injected only when caBundleMntDir is an explicit
// mountpoint in mountInfoFile — distinguishing an injected volume from the CA
// bundle that ships in the base image.
func collectMountFromPath(caFile, caBundleMntDir, mountInfoFile string) mountSignal {
	if !isMountPoint(caBundleMntDir, mountInfoFile) {
		return mountSignal{}
	}
	data, err := os.ReadFile(caFile)
	if err != nil {
		return mountSignal{}
	}
	sum := sha256.Sum256(data)
	return mountSignal{Present: true, Hash: fmt.Sprintf("%x", sum), Content: data}
}

// isMountPoint reports whether path appears as a mount destination in mountInfoFile.
// Field 5 (0-indexed) of each /proc/self/mountinfo line is the mount point.
func isMountPoint(path, mountInfoFile string) bool {
	data, err := os.ReadFile(mountInfoFile)
	if err != nil {
		return false
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 5 && fields[4] == path {
			return true
		}
	}
	return false
}

func buildCertPool(m mountSignal) *x509.CertPool {
	if m.Present {
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(m.Content)
		return pool
	}
	pool, err := x509.SystemCertPool()
	if err != nil {
		return x509.NewCertPool()
	}
	return pool
}

func dialTLS(addr string, pool *x509.CertPool) string {
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 10 * time.Second},
		"tcp",
		addr,
		&tls.Config{RootCAs: pool},
	)
	if err != nil {
		var x509Err *tls.CertificateVerificationError
		if errors.As(err, &x509Err) {
			return tlsResultX509
		}
		if isTLSCertError(err.Error()) {
			return tlsResultX509
		}
		return tlsResultOther
	}
	conn.Close()
	return tlsResultOK
}

func isTLSCertError(errStr string) bool {
	return strings.Contains(errStr, "x509") || strings.Contains(errStr, "certificate")
}

func resolveTokenURL(ctx context.Context, cl client.Client, cfg config) (string, error) {
	if cfg.TokenURLOverride != "" {
		return cfg.TokenURLOverride, nil
	}
	secret := &corev1.Secret{}
	if err := cl.Get(ctx, types.NamespacedName{Namespace: cfg.Namespace, Name: cfg.TLSSecret}, secret); err != nil {
		return "", fmt.Errorf("reading secret %s: %w", cfg.TLSSecret, err)
	}
	tokenURL := string(secret.Data["tokenurl"])
	if tokenURL == "" {
		return "", fmt.Errorf("secret %s missing tokenurl key", cfg.TLSSecret)
	}
	return tokenURL, nil
}

func computeSignal(mountPresent bool, tlsResult string) string {
	switch tlsResult {
	case tlsResultOK:
		return signalOK
	case tlsResultX509:
		if mountPresent {
			return signalAlert
		}
		return signalError
	default: // failed-other
		return signalError
	}
}

func patchBtpOperatorAnnotations(ctx context.Context, cl client.Client, cfg config, annotations map[string]string) error {
	cr := &btpv1alpha1.BtpOperator{}
	if err := cl.Get(ctx, types.NamespacedName{Namespace: cfg.Namespace, Name: "btpoperator"}, cr); err != nil {
		return fmt.Errorf("get BtpOperator: %w", err)
	}
	patch := client.MergeFrom(cr.DeepCopy())
	if cr.Annotations == nil {
		cr.Annotations = map[string]string{}
	}
	for k, v := range annotations {
		cr.Annotations[k] = v
	}
	return cl.Patch(ctx, cr, patch)
}

var probeAnnotationKeys = []string{
	"tls-probe-status",
	"tls-probe-hash",
	"tls-probe-updated-at",
	"tls-probe-last-hash",
}

func clearBtpOperatorProbeAnnotations(ctx context.Context, cl client.Client, cfg config) error {
	cr := &btpv1alpha1.BtpOperator{}
	if err := cl.Get(ctx, types.NamespacedName{Namespace: cfg.Namespace, Name: "btpoperator"}, cr); err != nil {
		return fmt.Errorf("get BtpOperator: %w", err)
	}
	annotations := cr.GetAnnotations()
	hasProbeAnnotations := false
	for _, k := range probeAnnotationKeys {
		if _, ok := annotations[k]; ok {
			hasProbeAnnotations = true
			break
		}
	}
	if !hasProbeAnnotations {
		return nil
	}
	patch := client.MergeFrom(cr.DeepCopy())
	for _, k := range probeAnnotationKeys {
		delete(annotations, k)
	}
	return cl.Patch(ctx, cr, patch)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := loadConfig()
	ctx := context.Background()

	restCfg, err := rest.InClusterConfig()
	if err != nil {
		logger.Error("in-cluster config", "err", err)
		return
	}

	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = btpv1alpha1.AddToScheme(scheme)

	cl, err := client.New(restCfg, client.Options{Scheme: scheme})
	if err != nil {
		logger.Error("k8s client", "err", err)
		return
	}

	annotations, err := runProbe(ctx, cl, cfg)
	if annotations == nil && err == nil {
		logger.Info("no rt-bootstrapper mount detected — clearing stale probe annotations")
		if clearErr := clearBtpOperatorProbeAnnotations(ctx, cl, cfg); clearErr != nil {
			logger.Error("clear probe annotations", "err", clearErr)
		}
		return
	}
	if err != nil {
		logger.Error("probe error", "err", err)
		annotations = map[string]string{
			"tls-probe-error": err.Error(),
		}
	}

	if patchErr := patchBtpOperatorAnnotations(ctx, cl, cfg, annotations); patchErr != nil {
		logger.Error("patch BtpOperator annotations", "err", patchErr)
	}

	if err == nil && annotations != nil {
		logger.Info("probe run complete",
			"status", annotations["tls-probe-status"],
			"hash", annotations["tls-probe-hash"],
		)
	}
}

func runProbe(ctx context.Context, cl client.Client, cfg config) (map[string]string, error) {
	mount := collectMount()
	pool := buildCertPool(mount)

	tokenURL, err := resolveTokenURL(ctx, cl, cfg)
	if err != nil {
		return nil, fmt.Errorf("resolve token URL: %w", err)
	}

	u, err := url.Parse(tokenURL)
	if err != nil {
		return nil, fmt.Errorf("parse token URL: %w", err)
	}
	host := u.Host
	if u.Port() == "" {
		host = u.Hostname() + ":443"
	}

	tlsResult := dialTLS(host, pool)
	signal := computeSignal(mount.Present, tlsResult)

	// No mount and TLS healthy — rt-bootstrapper not active, system CA works fine.
	// No need to annotate the CR; probe exits silently.
	if !mount.Present && tlsResult == tlsResultOK {
		return nil, nil
	}

	return map[string]string{
		"tls-probe-status":     signal,
		"tls-probe-hash":       mount.Hash,
		"tls-probe-updated-at": time.Now().UTC().Format(time.RFC3339),
	}, nil
}
