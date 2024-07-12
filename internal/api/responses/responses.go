package responses

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
	ImageUrl         string `json:"imageUrl"`
	DisplayName      string `json:"displayName"`
	DocumentationUrl string `json:"documentationUrl"`
	SupportUrl       string `json:"supportUrl"`
}

type ServiceInstances struct {
	NumItems int               `json:"numItems"`
	Items    []ServiceInstance `json:"items"`
}

type ServiceInstance struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	ServicePlanID   string `json:"servicePlanID"`
	ServicePlanName string `json:"servicePlanName"`
	SubaccountID    string `json:"subaccountID"`
	ClusterID       string `json:"clusterID"`
}

type ServiceOfferingDetails struct {
	LongDescription string                `json:"longDescription"`
	Plans           []ServiceOfferingPlan `json:"plans"`
}

type ServiceOfferingPlan struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ServiceBinding struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type ServiceBindings struct {
	Items []ServiceBinding `json:"items"`
}
