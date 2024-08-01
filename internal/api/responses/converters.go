package responses

import (
	"encoding/json"

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

func ToServiceOfferingDetailsVM(details *types.ServiceOfferingDetails) ServiceOfferingDetails {
	toReturn := ServiceOfferingDetails{
		Plans: []ServiceOfferingPlan{},
	}

	toReturn.LongDescription, _ = details.MetadataValueByFieldName(types.ServiceOfferingLongDescription)

	for _, plan := range details.ServicePlans.Items {
		toReturn.Plans = append(toReturn.Plans, ServiceOfferingPlan{
			ID:          plan.ID,
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
		namespace, _ := instance.ContextValueByFieldName(types.ContextNamespace)
		subaccountID, _ := instance.ContextValueByFieldName(types.ContextSubaccountID)
		clusterID, _ := instance.ContextValueByFieldName(types.ContextClusterID)
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

func ToServiceInstanceVM(instance *types.ServiceInstance) ServiceInstance {
	namespace, _ := instance.ContextValueByFieldName(types.ContextNamespace)
	subaccountID, _ := instance.ContextValueByFieldName(types.ContextSubaccountID)
	clusterID, _ := instance.ContextValueByFieldName(types.ContextClusterID)

	return ServiceInstance{
		ID:              instance.ID,
		Name:            instance.Name,
		Namespace:       namespace,
		ServicePlanID:   instance.ServicePlanID,
		ServicePlanName: instance.ServicePlanName,
		SubaccountID:    subaccountID,
		ClusterID:       clusterID,
	}
}

func ToServiceBindingsVM(bindings *types.ServiceBindings) (ServiceBindings, error) {
	toReturn := ServiceBindings{
		NumItems: len(bindings.Items),
		Items:    []ServiceBinding{},
	}

	for _, binding := range bindings.Items {
		binding, err := ToServiceBindingVM(&binding)
		if err != nil {
			return ServiceBindings{}, err
		}
		toReturn.Items = append(toReturn.Items, binding)
	}
	return toReturn, nil
}

func ToServiceBindingVM(binding *types.ServiceBinding) (ServiceBinding, error) {
	var credentials map[string]interface{}
	err := json.Unmarshal(binding.Credentials, &credentials)
	if err != nil {
		return ServiceBinding{}, err
	}
	return ServiceBinding{
		ID:          binding.ID,
		Name:        binding.Name,
		Credentials: credentials,
	}, nil
}
