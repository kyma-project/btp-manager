package clusterobject

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kyma-project/btp-manager/controllers"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OperatorName      = "btp-manager"
	ManagedByLabelKey = "app.kubernetes.io/managed-by"

	secretProviderName           = "SecretManager"
	btpServiceOperatorSecretName = "sap-btp-service-operator"
)

// SecretManager provides functionality to manage secrets in the cluster related to SAP BTP service operator and Service Manager
type SecretManager struct {
	client.Client
	namespaceProvider       *NamespaceProvider
	serviceInstanceProvider *ServiceInstanceProvider
	logger                  *slog.Logger
}

func NewSecretManager(k8sClient client.Client, nsProvider *NamespaceProvider, siProvider *ServiceInstanceProvider, logger *slog.Logger) *SecretManager {
	logger = logger.With(logComponentNameKey, secretProviderName)

	return &SecretManager{
		Client:                  k8sClient,
		namespaceProvider:       nsProvider,
		serviceInstanceProvider: siProvider,
		logger:                  logger,
	}
}

func (p *SecretManager) GetAll(ctx context.Context) (*corev1.SecretList, error) {
	p.logger.Info("fetching all btp operator secrets")
	secrets := &corev1.SecretList{}
	if err := p.getAllSapBtpServiceOperatorNamedSecrets(ctx, secrets); err != nil {
		return nil, err
	}

	namespaces, err := p.namespaceProvider.GetAll(ctx)
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

func (p *SecretManager) getAllSapBtpServiceOperatorNamedSecrets(ctx context.Context, secrets *corev1.SecretList) error {
	if err := p.List(ctx, secrets, client.MatchingFields{"metadata.name": btpServiceOperatorSecretName}); err != nil {
		p.logger.Error(fmt.Sprintf("failed to fetch all \"%s\" secrets", btpServiceOperatorSecretName), "error", err)
		return err
	}
	return nil
}

func (p *SecretManager) getNamespacesNames(namespaces *corev1.NamespaceList) []string {
	names := make([]string, len(namespaces.Items))
	for i, ns := range namespaces.Items {
		names[i] = ns.Name
	}
	return names
}

func (p *SecretManager) getAllSecretsWithNamespaceNamePrefix(ctx context.Context, secrets *corev1.SecretList, nsnames []string) error {
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

func (p *SecretManager) getSecretsFromRefInServiceInstances(ctx context.Context, siList *unstructured.UnstructuredList, secrets *corev1.SecretList) error {
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

func (p *SecretManager) secretExistsInList(secret *corev1.Secret, secrets *corev1.SecretList) bool {
	for _, s := range secrets.Items {
		if s.Name == secret.Name {
			return true
		}
	}
	return false
}

func (p *SecretManager) GetAllByLabels(ctx context.Context, labels map[string]string) (*corev1.SecretList, error) {
	p.logger.Info("fetching secrets by labels")
	secrets := &corev1.SecretList{}
	err := p.List(ctx, secrets, client.MatchingLabels(labels))
	if err != nil {
		p.logger.Error("while fetching secrets by labels", "error", err)
		return nil, err
	}

	if len(secrets.Items) == 0 {
		p.logger.Warn(fmt.Sprintf("no secrets found with labels: %v", labels))
		return nil, err
	}

	return secrets, err
}

func (p *SecretManager) GetByNameAndNamespace(ctx context.Context, name, namespace string) (*corev1.Secret, error) {
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

func (p *SecretManager) Create(ctx context.Context, secret *corev1.Secret) error {
	p.logger.Info(fmt.Sprintf("creating \"%s\" secret in \"%s\" namespace", secret.Name, secret.Namespace))
	if err := p.Client.Create(ctx, secret); err != nil {
		p.logger.Error(fmt.Sprintf("failed to create \"%s\" secret in \"%s\" namespace", secret.Name, secret.Namespace), "error", err)
		return err
	}

	return nil
}

func (p *SecretManager) Delete(ctx context.Context, secret *corev1.Secret) error {
	p.logger.Info(fmt.Sprintf("deleting \"%s\" secret in \"%s\" namespace", secret.Name, secret.Namespace))
	if err := p.Client.Delete(ctx, secret); err != nil {
		p.logger.Error(fmt.Sprintf("failed to delete \"%s\" secret in \"%s\" namespace", secret.Name, secret.Namespace), "error", err)
		return err
	}

	return nil
}

func (p *SecretManager) DeleteAll(ctx context.Context, secrets *corev1.SecretList) error {
	p.logger.Info(fmt.Sprintf("deleting %d secrets", len(secrets.Items)))
	for _, secret := range secrets.Items {
		if err := p.Client.Delete(ctx, &secret); err != nil {
			p.logger.Error(fmt.Sprintf("failed to delete \"%s\" secret in \"%s\" namespace", secret.Name, secret.Namespace), "error", err)
			return err
		}
	}

	return nil
}

func (p *SecretManager) DeleteAllByLabels(ctx context.Context, labels map[string]string) error {
	p.logger.Info(fmt.Sprintf("deleting secrets with labels: %v", labels))
	if err := p.DeleteAllOf(ctx, &corev1.Secret{}, client.MatchingLabels(labels)); err != nil {
		p.logger.Error("while deleting secrets by labels", "error", err, "labels", labels)
		return err
	}

	return nil
}
