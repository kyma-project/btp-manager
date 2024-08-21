package clusterobject

import (
	"context"
	"fmt"
	"log/slog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/btp-manager/controllers"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	secretProviderName           = "SecretProvider"
	btpServiceOperatorSecretName = "sap-btp-service-operator"
)

type SecretProvider struct {
	client.Reader
	client.Writer
	namespaceProvider       *NamespaceProvider
	serviceInstanceProvider *ServiceInstanceProvider
	logger                  *slog.Logger
}

func NewSecretProvider(reader client.Reader, nsProvider *NamespaceProvider, siProvider *ServiceInstanceProvider, logger *slog.Logger) *SecretProvider {
	logger = logger.With(logComponentNameKey, secretProviderName)

	return &SecretProvider{
		Reader:                  reader,
		namespaceProvider:       nsProvider,
		serviceInstanceProvider: siProvider,
		logger:                  logger,
	}
}

func (p *SecretProvider) All(ctx context.Context) (*corev1.SecretList, error) {
	p.logger.Info("fetching all btp operator secrets")
	secrets := &corev1.SecretList{}
	if err := p.getAllSapBtpServiceOperatorNamedSecrets(ctx, secrets); err != nil {
		return nil, err
	}

	namespaces, err := p.namespaceProvider.All(ctx)
	if err != nil {
		p.logger.Error("while fetching namespaces", "error", err)
		return nil, err
	}

	nsnames := p.getNamespacesNames(namespaces)
	if err := p.getAllSecretsWithNamespaceNamePrefix(ctx, secrets, nsnames); err != nil {
		return nil, err
	}

	siList, err := p.serviceInstanceProvider.AllWithSecretRef(ctx)
	if err != nil {
		p.logger.Error("while fetching service instances with secret ref", "error", err)
		return nil, err
	}

	if siList != nil && len(siList.Items) > 0 {
		if err := p.getSecretsFromRefInServiceInstances(ctx, siList, secrets); err != nil {
			return nil, err
		}
	}

	if len(secrets.Items) == 0 {
		p.logger.Warn(fmt.Sprintf("no btp operator secrets found"))
		return nil, err
	}

	return secrets, err
}

func (p *SecretProvider) getAllSapBtpServiceOperatorNamedSecrets(ctx context.Context, secrets *corev1.SecretList) error {
	if err := p.Reader.List(ctx, secrets, client.MatchingFields{"metadata.name": btpServiceOperatorSecretName}); err != nil {
		p.logger.Error(fmt.Sprintf("failed to fetch all \"%s\" secrets", btpServiceOperatorSecretName), "error", err)
		return err
	}
	return nil
}

func (p *SecretProvider) getNamespacesNames(namespaces *corev1.NamespaceList) []string {
	names := make([]string, len(namespaces.Items))
	for i, ns := range namespaces.Items {
		names[i] = ns.Name
	}
	return names
}

func (p *SecretProvider) getAllSecretsWithNamespaceNamePrefix(ctx context.Context, secrets *corev1.SecretList, nsnames []string) error {
	for _, nsname := range nsnames {
		secret := &corev1.Secret{}
		secretName := fmt.Sprintf("%s-%s", nsname, btpServiceOperatorSecretName)
		if err := p.Get(ctx, client.ObjectKey{Namespace: controllers.ChartNamespace, Name: secretName}, secret); err != nil {
			if k8serrors.IsNotFound(err) {
				p.logger.Info(fmt.Sprintf("secret \"%s\" not found in \"%s\" namespace", secretName, controllers.ChartNamespace))
				continue
			}
			p.logger.Error(fmt.Sprintf("failed to fetch \"%s\" secret", secretName), "error", err)
			return err
		}
		secrets.Items = append(secrets.Items, *secret)
	}

	return nil
}

func (p *SecretProvider) getSecretsFromRefInServiceInstances(ctx context.Context, siList *unstructured.UnstructuredList, secrets *corev1.SecretList) error {
	for _, item := range siList.Items {
		secretRef, found, err := unstructured.NestedString(item.Object, "spec", secretRefKey)
		if err != nil {
			p.logger.Error(fmt.Sprintf("while traversing \"%s\" unstructured object to find \"%s\" key", item.GetName(), secretRefKey), "error", err)
			return err
		} else if !found {
			p.logger.Warn(fmt.Sprintf("expected secret ref not found in \"%s\" service instance", item.GetName()))
			continue
		}
		secret := &corev1.Secret{}
		if err := p.Get(ctx, client.ObjectKey{Namespace: controllers.ChartNamespace, Name: secretRef}, secret); err != nil {
			if k8serrors.IsNotFound(err) {
				p.logger.Warn(fmt.Sprintf("secret \"%s\" not found in \"%s\" namespace", secretRef, controllers.ChartNamespace))
				continue
			}
			p.logger.Error(fmt.Sprintf("failed to fetch \"%s\" secret", secretRef), "error", err)
			return err
		}
		if p.secretExistsInList(secret, secrets) {
			continue
		}
		secrets.Items = append(secrets.Items, *secret)
	}

	return nil
}

func (p *SecretProvider) secretExistsInList(secret *corev1.Secret, secrets *corev1.SecretList) bool {
	for _, s := range secrets.Items {
		if s.Name == secret.Name {
			return true
		}
	}
	return false
}

func (p *SecretProvider) GetByNameAndNamespace(ctx context.Context, name, namespace string) (*corev1.Secret, error) {
	p.logger.Info(fmt.Sprintf("fetching \"%s\" secret in \"%s\" namespace", name, namespace))
	secret := &corev1.Secret{}
	if err := p.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, secret); err != nil {
		if k8serrors.IsNotFound(err) {
			p.logger.Warn(fmt.Sprintf("secret \"%s\" not found in \"%s\" namespace", name, namespace))
			return nil, err
		}
		p.logger.Error(fmt.Sprintf("failed to fetch \"%s\" secret in \"%s\" namespace", name, namespace), "error", err)
		return nil, err
	}

	return secret, nil
}

func (p *SecretProvider) CreateSecret(ctx context.Context, name, namespace string) (*corev1.Secret, error) {
	p.logger.Info(fmt.Sprintf("creating \"%s\" secret in \"%s\" namespace", btpServiceOperatorSecretName, namespace))
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	if err := p.Writer.Create(ctx, secret); err != nil {
		p.logger.Error(fmt.Sprintf("failed to create \"%s\" secret in \"%s\" namespace", btpServiceOperatorSecretName, namespace), "error", err)
		return nil, err
	}

	return secret, nil
}
