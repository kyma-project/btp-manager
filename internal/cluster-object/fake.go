package clusterobject

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FakeSecretManager struct {
	secrets []*corev1.Secret
}

func NewFakeSecretManager() *FakeSecretManager {
	return &FakeSecretManager{secrets: make([]*corev1.Secret, 0)}
}

func (p *FakeSecretManager) Create(ctx context.Context, secret *corev1.Secret) error {
	p.secrets = append(p.secrets, secret)
	return nil
}

func (p *FakeSecretManager) GetAll(ctx context.Context) (*corev1.SecretList, error) {
	items := make([]corev1.Secret, 0)
	for _, secret := range p.secrets {
		items = append(items, *secret)
	}
	return &corev1.SecretList{
		Items: items,
	}, nil
}

func (p *FakeSecretManager) GetByNameAndNamespace(ctx context.Context, name, namespace string) (*corev1.Secret, error) {
	for _, secret := range p.secrets {
		if secret.Name == name && secret.Namespace == namespace {
			return secret, nil
		}
	}
	return nil, errors.NewNotFound(corev1.Resource("secret"), name)
}

func (p *FakeSecretManager) Clean() {
	p.secrets = make([]*corev1.Secret, 0)
}

func (p *FakeSecretManager) GetAllByLabels(ctx context.Context, labels map[string]string) (*corev1.SecretList, error) {
	items := make([]corev1.Secret, 0)
	mustMatchLen := len(labels)
	for _, secret := range p.secrets {
		if secret.Labels == nil {
			continue
		}
		matchingLabelsNum := 0
		secretLabels := secret.Labels
		for key, value := range labels {
			if secretLabels[key] == value {
				matchingLabelsNum++
				continue
			}
		}
		if matchingLabelsNum == mustMatchLen {
			items = append(items, *secret)
		}
	}
	return &corev1.SecretList{
		Items: items,
	}, nil
}

func (p *FakeSecretManager) Delete(ctx context.Context, secret *corev1.Secret) error {
	for i, s := range p.secrets {
		if s.Name == secret.Name && s.Namespace == secret.Namespace {
			p.secrets = append(p.secrets[:i], p.secrets[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("secret not found")
}

func (p *FakeSecretManager) DeleteList(ctx context.Context, secrets *corev1.SecretList) error {
	for _, secret := range secrets.Items {
		if err := p.Delete(ctx, &secret); err != nil {
			return err
		}
	}
	return nil
}

func (p *FakeSecretManager) DeleteAllByLabels(ctx context.Context, labels map[string]string) error {
	secrets, err := p.GetAllByLabels(ctx, labels)
	if err != nil {
		return err
	}
	return p.DeleteList(ctx, secrets)
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
