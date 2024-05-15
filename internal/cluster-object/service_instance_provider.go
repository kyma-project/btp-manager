package clusterobject

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/kyma-project/btp-manager/controllers"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	serviceInstanceProviderName = "ServiceInstanceProvider"
	secretRefKey                = "btpAccessCredentialsSecret"
)

type ServiceInstanceProvider struct {
	client.Reader
	logger *slog.Logger
}

func NewServiceInstanceProvider(reader client.Reader, logger *slog.Logger) *ServiceInstanceProvider {
	logger = logger.With(logComponentNameKey, serviceInstanceProviderName)

	return &ServiceInstanceProvider{
		Reader: reader,
		logger: logger,
	}
}

func (p *ServiceInstanceProvider) AllWithSecretRef(ctx context.Context) (*unstructured.UnstructuredList, error) {
	filtered, err := p.All(ctx)
	if err != nil {
		p.logger.Error("while fetching filtered service instances", "error", err)
		return nil, err
	}

	if filtered == nil || len(filtered.Items) == 0 {
		return nil, nil
	}

	if err := p.filterBySecretRef(filtered); err != nil {
		p.logger.Error("while filtering service instances by secret ref", "error", err)
		return nil, err
	}

	return filtered, nil
}

func (p *ServiceInstanceProvider) All(ctx context.Context) (*unstructured.UnstructuredList, error) {
	siCrdExists, err := p.crdExists(ctx, controllers.InstanceGvk)
	if err != nil {
		p.logger.Error("failed to check if ServiceInstance CRD exists", "error", err)
		return nil, err
	}
	if !siCrdExists {
		p.logger.Info("cannot fetch SAP BTP service operator secrets from ServiceInstances due to missing CRD")
		return nil, nil
	}

	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(controllers.InstanceGvk)
	if err := p.List(ctx, list); err != nil {
		p.logger.Error("failed to list all service instances", "error", err)
		return nil, err
	}
	if len(list.Items) == 0 {
		p.logger.Info("no service instances found")
		return list, nil
	}

	return list, nil
}

func (p *ServiceInstanceProvider) filterBySecretRef(all *unstructured.UnstructuredList) error {
	for i := 0; i < len(all.Items); {
		found, err := p.hasSecretRef(all.Items[i])
		if err != nil {
			return err
		}
		if !found {
			all.Items = append(all.Items[:i], all.Items[i+1:]...)
			continue
		}
		i++
	}

	return nil
}

func (p *ServiceInstanceProvider) hasSecretRef(item unstructured.Unstructured) (bool, error) {
	_, found, err := unstructured.NestedString(item.Object, "spec", secretRefKey)
	if err != nil {
		p.logger.Error(fmt.Sprintf("while traversing \"%s\" unstructured object to find \"%s\" key", item.GetName(), secretRefKey), "error", err)
		return false, err
	}

	return found, nil
}

func (p *ServiceInstanceProvider) crdExists(ctx context.Context, gvk schema.GroupVersionKind) (bool, error) {
	crdName := fmt.Sprintf("%ss.%s", strings.ToLower(gvk.Kind), gvk.Group)
	crd := &apiextensionsv1.CustomResourceDefinition{}

	if err := p.Get(ctx, client.ObjectKey{Name: crdName}, crd); err != nil {
		if k8serrors.IsNotFound(err) {
			p.logger.Info(fmt.Sprintf("%s CRD does not exist", crdName))
			return false, nil
		} else {
			p.logger.Error("failed to get CRD", "name", crdName, "error", err)
			return false, err
		}
	}

	return true, nil
}
