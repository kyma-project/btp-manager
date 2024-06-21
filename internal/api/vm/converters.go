package vm

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
		NumItems: len(offerings.ServiceOfferings),
		Items:    []ServiceOffering{},
	}

	for _, offering := range offerings.ServiceOfferings {
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

	for _, plan := range details.ServicePlans.ServicePlans {
		toReturn.Plans = append(toReturn.Plans, ServiceOfferingPlan{
			Name:        plan.Name,
			Description: plan.Description,
		})
	}

	return toReturn
}
