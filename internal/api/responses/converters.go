package responses

import (
	"github.com/kyma-project/btp-manager/internal/service-manager/types"
	v1 "k8s.io/api/core/v1"
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
	toReturn := ServiceOfferings{
		NumItems: len(offerings.Items),
		Items:    []ServiceOffering{},
	}

	for _, offering := range offerings.Items {
		imageUrl, _ := offering.MetadataValueByFieldName(types.ServiceOfferingImageUrl)
		displayName, _ := offering.MetadataValueByFieldName(types.ServiceOfferingDisplayName)
		supportUrl, _ := offering.MetadataValueByFieldName(types.ServiceOfferingSupportURL)
		documentationUrl, _ := offering.MetadataValueByFieldName(types.ServiceOfferingDocumentationUrl)
		offering := ServiceOffering{
			ID:          offering.ID,
			Description: offering.Description,
			CatalogID:   offering.CatalogID,
			CatalogName: offering.CatalogName,
			Metadata: ServiceOfferingMetadata{
				ImageUrl:         imageUrl,
				DisplayName:      displayName,
				SupportUrl:       supportUrl,
				DocumentationUrl: documentationUrl,
			},
		}
		toReturn.Items = append(toReturn.Items, offering)
	}
	return toReturn
}

func ToServiceInstancesVM(instances *types.ServiceInstances) ServiceInstances {
	toReturn := ServiceInstances{
		NumItems: len(instances.Items),
		Items:    []ServiceInstance{},
	}

	for _, instance := range instances.Items {
		instance := ServiceInstance{
			ID:          instance.ID,
			Name:        instance.Name,
			Description: instance.Description,
		}
		toReturn.Items = append(toReturn.Items, instance)
	}
	return toReturn
}

func ToServiceOfferingDetailsVM(details *types.ServiceOfferingDetails) ServiceOfferingDetails {
	toReturn := ServiceOfferingDetails{
		Plans: []ServiceOfferingPlan{},
	}

	toReturn.LongDescription, _ = details.MetadataValueByFieldName(types.ServiceOfferingLongDescription)

	for _, plan := range details.ServicePlans.Items {
		toReturn.Plans = append(toReturn.Plans, ServiceOfferingPlan{
			Name:        plan.Name,
			Description: plan.Description,
		})
	}

	return toReturn
}

func ToServiceInstancesVM(instances *types.ServiceInstances) ServiceInstances {
	toReturn := ServiceInstances{
		NumItems: len(instances.Items),
		Items:    []ServiceInstance{},
	}

	for _, instance := range instances.Items {
		namespace, _ := instance.ContextValueByFieldName(types.ServiceInstanceNamespace)
		subaccountID, _ := instance.ContextValueByFieldName(types.ServiceInstanceSubaccountID)
		clusterID, _ := instance.ContextValueByFieldName(types.ServiceInstanceClusterID)
		instance := ServiceInstance{
			ID:           instance.ID,
			Name:         instance.Name,
			Namespace:    namespace,
			SubaccountID: subaccountID,
			ClusterID:    clusterID,
		}
		toReturn.Items = append(toReturn.Items, instance)
	}
	return toReturn
}

func ToServiceInstanceVM(instance *types.ServiceInstance, plan *types.ServicePlan) ServiceInstance {
	namespace, _ := instance.ContextValueByFieldName(types.ServiceInstanceNamespace)
	subaccountID, _ := instance.ContextValueByFieldName(types.ServiceInstanceSubaccountID)
	clusterID, _ := instance.ContextValueByFieldName(types.ServiceInstanceClusterID)

	return ServiceInstance{
		ID:              instance.ID,
		Name:            instance.Name,
		Namespace:       namespace,
		ServicePlanID:   instance.ServicePlanID,
		ServicePlanName: plan.Name,
		SubaccountID:    subaccountID,
		ClusterID:       clusterID,
	}
}

func ToServiceBindingsVM(bindings *types.ServiceBindings) ServiceBindings {
	toReturn := ServiceBindings{
		Items: []ServiceBinding{},
	}

	for _, _ = range bindings.Items {
		n := ServiceBinding{}
		toReturn.Items = append(toReturn.Items, n)
	}

	return toReturn
}

func ToServiceBindingVM(binding *types.ServiceBinding) ServiceBindings {
	return ServiceBindings{}
}
