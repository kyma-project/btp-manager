package provisioning

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestVerifySecret_AllKeysPresent(t *testing.T) {
	s := &corev1.Secret{
		Data: map[string][]byte{
			"clientid":     []byte("id"),
			"clientsecret": []byte("secret"),
			"sm_url":       []byte("url"),
			"tokenurl":     []byte("turl"),
			"cluster_id":   []byte("cid"),
		},
	}
	if err := verifySecret(s); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestVerifySecret_MissingKey(t *testing.T) {
	s := &corev1.Secret{
		Data: map[string][]byte{
			"clientid": []byte("id"),
		},
	}
	if err := verifySecret(s); err == nil {
		t.Fatal("expected error for missing keys, got nil")
	}
}

func TestVerifySecret_EmptyValue(t *testing.T) {
	s := &corev1.Secret{
		Data: map[string][]byte{
			"clientid":     []byte(""),
			"clientsecret": []byte("s"),
			"sm_url":       []byte("u"),
			"tokenurl":     []byte("t"),
			"cluster_id":   []byte("c"),
		},
	}
	if err := verifySecret(s); err == nil {
		t.Fatal("expected error for empty value, got nil")
	}
}
