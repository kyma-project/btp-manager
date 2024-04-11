package clusterobject

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kyma-project/btp-manager/controllers"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	secretProviderName           = "SecretProvider"
	btpServiceOperatorSecretName = "sap-btp-service-operator"
)

type SecretProvider struct {
	client.Reader
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

func (p *SecretProvider) All(ctx context.Context) (*v1.SecretList, error) {
	p.logger.Info("fetching all btp operator secrets")
	secrets := &v1.SecretList{}
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

	return secrets, err
}

func (p *SecretProvider) getAllSapBtpServiceOperatorNamedSecrets(ctx context.Context, secrets *v1.SecretList) error {
	if err := p.Reader.List(ctx, secrets, client.MatchingFields{"metadata.name": btpServiceOperatorSecretName}); err != nil {
		p.logger.Error(fmt.Sprintf("failed to fetch all \"%s\" secrets", btpServiceOperatorSecretName), "error", err)
		return err
	}
	return nil
}

func (p *SecretProvider) getNamespacesNames(namespaces *v1.NamespaceList) []string {
	names := make([]string, len(namespaces.Items))
	for i, ns := range namespaces.Items {
		names[i] = ns.Name
	}
	return names
}

func (p *SecretProvider) getAllSecretsWithNamespaceNamePrefix(ctx context.Context, secrets *v1.SecretList, nsnames []string) error {
	for _, nsname := range nsnames {
		secret := &v1.Secret{}
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
