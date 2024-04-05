// ** Kubernetes related */

export interface Secrets {
  items: Secret[];
}

export interface Secret {
  name: string;
  namespace: string;
}

// ** Service Offerings related */

export interface ServiceOfferings {
  num_items: number;
  items: ServiceOfferingBrief[];
}

export interface ServiceOfferingBrief {
  id: string;
  description: string;
  catalog_id: string;
  catalog_name: string;
  metadata: ServieOfferingMetadataBrief;
}

export interface ServieOfferingMetadataBrief {
  imageUrl: string;
  displayName: string;
}

//** Service Instances related */

export interface ServiceInstancesBrief {
  items: ServiceInstanceBrief[];
}

export interface ServiceInstanceBrief {
  id: string;
  name: string;
  context: string[];
  namespace: string;
  service_bindings: ServiceInstaceBindingsBrief[];
}

export interface ServiceInstaceBindingsBrief {
  id: string;
  name: string;
  namespace: string;
}

export interface ServiceInstanceDetails {
  id: string;
  name: string;
  context: string;
  namespace: string;
}
