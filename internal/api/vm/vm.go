package vm

type Secrets struct {
	Items []Secret `json:"items"`
}

type Secret struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type ServiceOfferings struct {
	NumItems int               `json:"numItems"`
	Items    []ServiceOffering `json:"items"`
}

type ServiceOffering struct {
	ID          string                  `json:"id"`
	Description string                  `json:"description"`
	CatalogID   string                  `json:"catalogID"`
	CatalogName string                  `json:"catalogName"`
	Metadata    ServiceOfferingMetadata `json:"metadata"`
}

type ServiceOfferingMetadata struct {
	ImageUrl    string `json:"imageUrl"`
	DisplayName string `json:"displayName"`
}

type ServiceInstances struct {
	Items []ServiceInstance `json:"items"`
}

type ServiceInstance struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type ServiceOfferingDetails struct {
	LongDescription string                `json:"longDescription"`
	Plans           []ServiceOfferingPlan `json:"plans"`
}

type ServiceOfferingPlan struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	DocumentationUrl string `json:"documentationUrl"`
	SupportUrl       string `json:"supportUrl"`
}
