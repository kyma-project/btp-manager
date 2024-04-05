export default interface ServiceOfferings {
  num_items: number;
  items: ServiceOffering[];
}

export interface ServiceOffering {
  id: string;
  ready: boolean;
  name: string;
  description: string;
  bindable: boolean;
  instances_retrievable: boolean;
  bindings_retrievable: boolean;
  plan_updateable: boolean;
  allow_context_updates: boolean;
  metadata: ServieOfferingMetadata;
  broker_id: string;
  catalog_id: string;
  catalog_name: string;
  created_at: string;
  updated_at: string;
}

interface ServieOfferingMetadata {
  createBindingDocumentationUrl: string;
  discoveryCenterUrl: string;
  displayName: string;
  documentationUrl: string;
  imageUrl: string;
  longDescription: string;
  serviceInventoryId: string;
  shareable: boolean;
  supportUrl: string;
}