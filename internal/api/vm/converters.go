package vm

import (
	"github.com/kyma-project/btp-manager/internal/service-manager/types"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func ToSecretVM(list v1.SecretList) Secrets {
	secrets := Secrets{
		Items: []Secret{},
	}
	for _, secret := range list.Items {
		secrets.Items = append(
			secrets.Items, Secret{
				Name:      secret.Name,
				Namespace: secret.Namespace,
			},
		)
	}
	return secrets
}

func ToServiceOfferingsVM(offerings *types.ServiceOfferings) ServiceOfferings {
	serviceOfferings := ServiceOfferings{
		NumItems: len(offerings.ServiceOfferings),
		Items:    []ServiceOffering{},
	}

	for _, offering := range offerings.ServiceOfferings {
		imageUrl, _ := offering.MetadataValueByFieldName(types.ServiceOfferingImageUrl)
		displayName, _ := offering.MetadataValueByFieldName(types.ServiceOfferingDisplayName)
		offering := ServiceOffering{
			ID:          offering.ID,
			Description: offering.Description,
			CatalogID:   offering.CatalogID,
			CatalogName: offering.CatalogName,
			Metadata: ServiceOfferingMetadata{
				ImageUrl:    imageUrl,
				DisplayName: displayName,
			},
		}
		serviceOfferings.Items = append(serviceOfferings.Items, offering)
	}
	return serviceOfferings
}

func ToServiceInstancesVM(serviceInstances []unstructured.Unstructured) ServiceInstances {
	instances := ServiceInstances{
		Items: []ServiceInstance{},
	}

	for _, serviceInstance := range serviceInstances {
		name, found, err := unstructured.NestedString(serviceInstance.Object, "metadata", "name")
		if err != nil || !found {
			name = "not found"
		}
		namespace, found, err := unstructured.NestedString(serviceInstance.Object, "metadata", "namespace")
		if err != nil || !found {
			namespace = "not found"
		}
		instances.Items = append(
			instances.Items, ServiceInstance{
				Name:      name,
				Namespace: namespace,
			},
		)
	}

	return instances
}
