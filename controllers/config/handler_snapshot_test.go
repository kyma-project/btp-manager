package config

import (
	"reflect"
	"testing"
	"time"

	"github.com/kyma-project/btp-manager/internal/certs"
)

type configState struct {
	chartNamespace                 string
	chartPath                      string
	secretName                     string
	configName                     string
	deploymentName                 string
	processingStateRequeueInterval time.Duration
	readyStateRequeueInterval      time.Duration
	readyTimeout                   time.Duration
	hardDeleteCheckInterval        time.Duration
	hardDeleteTimeout              time.Duration
	resourcesPath                  string
	readyCheckInterval             time.Duration
	deleteRequestTimeout           time.Duration
	caCertificateExpiration        time.Duration
	webhookCertificateExpiration   time.Duration
	expirationBoundary             time.Duration
	rsaKeyBits                     int
	enableLimitedCache             string
	statusUpdateTimeout            time.Duration
	statusUpdateCheckInterval      time.Duration
	managerResourcesPath           string
}

func captureConfigState() configState {
	return configState{
		chartNamespace:                 ChartNamespace,
		chartPath:                      ChartPath,
		secretName:                     SecretName,
		configName:                     ConfigName,
		deploymentName:                 DeploymentName,
		processingStateRequeueInterval: ProcessingStateRequeueInterval,
		readyStateRequeueInterval:      ReadyStateRequeueInterval,
		readyTimeout:                   ReadyTimeout,
		hardDeleteCheckInterval:        HardDeleteCheckInterval,
		hardDeleteTimeout:              HardDeleteTimeout,
		resourcesPath:                  ResourcesPath,
		readyCheckInterval:             ReadyCheckInterval,
		deleteRequestTimeout:           DeleteRequestTimeout,
		caCertificateExpiration:        CaCertificateExpiration,
		webhookCertificateExpiration:   WebhookCertificateExpiration,
		expirationBoundary:             ExpirationBoundary,
		rsaKeyBits:                     certs.RsaKeyBits(),
		enableLimitedCache:             EnableLimitedCache,
		statusUpdateTimeout:            StatusUpdateTimeout,
		statusUpdateCheckInterval:      StatusUpdateCheckInterval,
		managerResourcesPath:           ManagerResourcesPath,
	}
}

func restoreConfigState(state configState) {
	ChartNamespace = state.chartNamespace
	ChartPath = state.chartPath
	SecretName = state.secretName
	ConfigName = state.configName
	DeploymentName = state.deploymentName
	ProcessingStateRequeueInterval = state.processingStateRequeueInterval
	ReadyStateRequeueInterval = state.readyStateRequeueInterval
	ReadyTimeout = state.readyTimeout
	HardDeleteCheckInterval = state.hardDeleteCheckInterval
	HardDeleteTimeout = state.hardDeleteTimeout
	ResourcesPath = state.resourcesPath
	ReadyCheckInterval = state.readyCheckInterval
	DeleteRequestTimeout = state.deleteRequestTimeout
	CaCertificateExpiration = state.caCertificateExpiration
	WebhookCertificateExpiration = state.webhookCertificateExpiration
	ExpirationBoundary = state.expirationBoundary
	certs.SetRsaKeyBits(state.rsaKeyBits)
	EnableLimitedCache = state.enableLimitedCache
	StatusUpdateTimeout = state.statusUpdateTimeout
	StatusUpdateCheckInterval = state.statusUpdateCheckInterval
	ManagerResourcesPath = state.managerResourcesPath
}

func TestConfigSnapshot(t *testing.T) {
	original := captureConfigState()
	t.Cleanup(func() {
		restoreConfigState(original)
	})

	ChartNamespace = "custom-ns"
	ChartPath = "./custom-chart"
	SecretName = "custom-secret"
	ConfigName = "custom-config"
	DeploymentName = "custom-deployment"
	ProcessingStateRequeueInterval = 11 * time.Minute
	ReadyStateRequeueInterval = 12 * time.Minute
	ReadyTimeout = 13 * time.Minute
	HardDeleteCheckInterval = 14 * time.Second
	HardDeleteTimeout = 15 * time.Minute
	ResourcesPath = "./custom-resources"
	ReadyCheckInterval = 16 * time.Second
	DeleteRequestTimeout = 17 * time.Minute
	CaCertificateExpiration = 18 * time.Hour
	WebhookCertificateExpiration = 19 * time.Hour
	ExpirationBoundary = -20 * time.Hour
	certs.SetRsaKeyBits(3072)
	EnableLimitedCache = "false"
	StatusUpdateTimeout = 21 * time.Second
	StatusUpdateCheckInterval = 22 * time.Millisecond
	ManagerResourcesPath = "./custom-manager-resources"

	got := configSnapshot()
	want := map[string]any{
		"ChartNamespace":                 "custom-ns",
		"ChartPath":                      "./custom-chart",
		"SecretName":                     "custom-secret",
		"ConfigName":                     "custom-config",
		"DeploymentName":                 "custom-deployment",
		"ProcessingStateRequeueInterval": 11 * time.Minute,
		"ReadyStateRequeueInterval":      12 * time.Minute,
		"ReadyTimeout":                   13 * time.Minute,
		"HardDeleteCheckInterval":        14 * time.Second,
		"HardDeleteTimeout":              15 * time.Minute,
		"ResourcesPath":                  "./custom-resources",
		"ReadyCheckInterval":             16 * time.Second,
		"DeleteRequestTimeout":           17 * time.Minute,
		"CaCertificateExpiration":        18 * time.Hour,
		"WebhookCertificateExpiration":   19 * time.Hour,
		"ExpirationBoundary":             -20 * time.Hour,
		"RsaKeyBits":                     3072,
		"EnableLimitedCache":             "false",
		"StatusUpdateTimeout":            21 * time.Second,
		"StatusUpdateCheckInterval":      22 * time.Millisecond,
		"ManagerResourcesPath":           "./custom-manager-resources",
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("configSnapshot mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestChangedSnapshotKeys(t *testing.T) {
	before := map[string]any{
		"ChartNamespace": "kyma-system",
		"ReadyTimeout":   5 * time.Minute,
		"RsaKeyBits":     4096,
	}
	after := map[string]any{
		"ChartNamespace": "custom-ns",
		"ReadyTimeout":   5 * time.Minute,
		"EnableFeatureX": true,
	}

	got := changedSnapshotKeys(before, after)
	want := []string{"ChartNamespace", "EnableFeatureX", "RsaKeyBits"}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("changedSnapshotKeys mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestChangedSnapshotKeysNoChanges(t *testing.T) {
	before := map[string]any{"A": 1, "B": "x"}
	after := map[string]any{"A": 1, "B": "x"}

	got := changedSnapshotKeys(before, after)
	if len(got) != 0 {
		t.Fatalf("expected no changed keys, got: %#v", got)
	}
}

