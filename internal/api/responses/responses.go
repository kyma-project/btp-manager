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
	CatalogID   string                  `json:"catalog_id"`
	CatalogName string                  `json:"catalog_name"`
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
	ServicePlanID   string `json:"service_plan_id"`
	ServicePlanName string `json:"service_plan_name"`
	SubaccountID    string `json:"subaccount_id"`
	ClusterID       string `json:"cluster_id"`
}

type ServiceOfferingDetails struct {
	LongDescription string                `json:"longDescription"`
	Plans           []ServiceOfferingPlan `json:"plans"`
}

type ServiceOfferingPlan struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ServiceBinding struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Credentials map[string]interface{} `json:"credentials"`
}

type ServiceBindings struct {
	NumItems int              `json:"numItems"`
	Items    []ServiceBinding `json:"items"`
}
