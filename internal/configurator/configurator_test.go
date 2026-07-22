package configurator

import (
	"context"
	"errors"
	"testing"

	"github.com/kyma-project/btp-manager/internal/conditions"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type stubReader struct {
	credNs           string
	clusterId        string
	defaultSecret    *corev1.Secret
	configMap        *corev1.ConfigMap
	defaultSecretErr error
	configMapErr     error
}

func (s *stubReader) InitializeFromSecret(_ *corev1.Secret)   {}
func (s *stubReader) CredentialsNamespaceFromManager() string { return s.credNs }
func (s *stubReader) ClusterIdFromManager() string            { return s.clusterId }
func (s *stubReader) GetDefaultCredentialsSecret(_ context.Context) (*corev1.Secret, error) {
	return s.defaultSecret, s.defaultSecretErr
}
func (s *stubReader) GetSapBtpServiceOperatorConfigMap(_ context.Context) (*corev1.ConfigMap, error) {
	return s.configMap, s.configMapErr
}

func secret(ns string) *corev1.Secret {
	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: ns}}
}

func configMap(clusterId string) *corev1.ConfigMap {
	return &corev1.ConfigMap{Data: map[string]string{"CLUSTER_ID": clusterId}}
}

func TestCheck_NoChanges(t *testing.T) {
	c := NewConfigurator(&stubReader{
		credNs:        "kyma-system",
		clusterId:     "c1",
		defaultSecret: secret("kyma-system"),
		configMap:     configMap("c1"),
	})
	result := c.Check(context.Background(), &corev1.Secret{})
	if result.ReprocessReason != "" || result.ErrorReason != "" {
		t.Fatalf("expected empty result, got %+v", result)
	}
}

func TestCheck_NoDefaultSecret_NoConfigMap(t *testing.T) {
	c := NewConfigurator(&stubReader{credNs: "kyma-system", clusterId: "c1"})
	result := c.Check(context.Background(), &corev1.Secret{})
	if result.ReprocessReason != "" || result.ErrorReason != "" {
		t.Fatalf("expected empty result when no operand resources exist, got %+v", result)
	}
}

func TestCheck_CredentialsNamespaceDrift(t *testing.T) {
	c := NewConfigurator(&stubReader{
		credNs:        "new-ns",
		clusterId:     "c1",
		defaultSecret: secret("old-ns"),
	})
	result := c.Check(context.Background(), &corev1.Secret{})
	if result.ReprocessReason != conditions.CredentialsNamespaceChanged {
		t.Fatalf("expected CredentialsNamespaceChanged, got %v", result.ReprocessReason)
	}
	if result.ErrorReason != "" {
		t.Fatalf("expected no error reason, got %v", result.ErrorReason)
	}
}

func TestCheck_ClusterIdDrift(t *testing.T) {
	c := NewConfigurator(&stubReader{
		credNs:        "kyma-system",
		clusterId:     "new-id",
		defaultSecret: secret("kyma-system"),
		configMap:     configMap("old-id"),
	})
	result := c.Check(context.Background(), &corev1.Secret{})
	if result.ReprocessReason != conditions.ClusterIdChanged {
		t.Fatalf("expected ClusterIdChanged, got %v", result.ReprocessReason)
	}
	if result.ErrorReason != "" {
		t.Fatalf("expected no error reason, got %v", result.ErrorReason)
	}
}

func TestCheck_DefaultSecretError(t *testing.T) {
	c := NewConfigurator(&stubReader{defaultSecretErr: errors.New("api down")})
	result := c.Check(context.Background(), &corev1.Secret{})
	if result.ErrorReason != conditions.GettingDefaultCredentialsSecretFailed {
		t.Fatalf("expected GettingDefaultCredentialsSecretFailed, got %v", result.ErrorReason)
	}
}

func TestCheck_ConfigMapError(t *testing.T) {
	c := NewConfigurator(&stubReader{
		credNs:        "kyma-system",
		clusterId:     "c1",
		defaultSecret: secret("kyma-system"),
		configMapErr:  errors.New("api down"),
	})
	result := c.Check(context.Background(), &corev1.Secret{})
	if result.ErrorReason != conditions.GettingSapBtpServiceOperatorConfigMapFailed {
		t.Fatalf("expected GettingSapBtpServiceOperatorConfigMapFailed, got %v", result.ErrorReason)
	}
}
