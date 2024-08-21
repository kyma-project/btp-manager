export interface Secrets {
  items: Secret[];
}

export interface Secret {
  name: string;
  namespace: string;
}

export interface ServiceOfferings {
  numItems: number;
  items: ServiceOffering[];
}

export interface ServiceOffering {
  id: string;
  description: string;
  catalogId: string;
  catalog_name: string;
  metadata: ServiceOfferingMetadata;
}

export interface ServiceOfferingMetadata {
  imageUrl: string;
  displayName: string;
  supportUrl: string;
  documentationUrl: string;
}

export interface ServiceOfferingDetails {
  longDescription: string;
  plans: ServiceOfferingPlan[];
}

export interface ServiceOfferingPlan {
  name: string;
  description: string;
  supportUrl: string;
  documentationUrl: string;
}

export interface ServiceInstances {
  items: ServiceInstance[];
}

export interface ServiceInstance {
  id: string;
  name: string;
  context: string[];
  namespace: string;
  serviceBindings: ServiceInstanceBinding[];
}

export interface ServiceInstanceBindings {
  items: ServiceInstanceBinding[];
}

export interface ServiceInstanceBinding {
  id: string;
  name: string;
  namespace: string;
}