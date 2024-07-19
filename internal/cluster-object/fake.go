package clusterobject

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FakeSecretProvider struct {
	secrets []*corev1.Secret
}

func NewFakeSecretProvider() *FakeSecretProvider {
	return &FakeSecretProvider{secrets: make([]*corev1.Secret, 0)}
}

func (p *FakeSecretProvider) AddSecret(secret *corev1.Secret) {
	p.secrets = append(p.secrets, secret)
}

func (p *FakeSecretProvider) GetByNameAndNamespace(ctx context.Context, name, namespace string) (*corev1.Secret, error) {
	for _, secret := range p.secrets {
		if secret.Name == name && secret.Namespace == namespace {
			return secret, nil
		}
	}
	return nil, fmt.Errorf("secret not found")
}

func (p *FakeSecretProvider) Clean() {
	p.secrets = make([]*corev1.Secret, 0)
}

func FakeDefaultSecret() *corev1.Secret {
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
