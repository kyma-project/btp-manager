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

export class CreateServiceInstance {
  id: string = "";
  name: string = "";
  service_plan_id: string = "";
  labels: { [key: string]: string[] } = {};
  parameters: string = "{}";
}

export class ApiError {
  message: string = "";
  name: string = "";
  code: string = "";
  config: string = "";
  request: string = "";
  response: Response = new Response();
}

export class Response {
  data: string = "";
  status: number = 0;
  statusText: string = "";
  headers: string = "";
  config: string = "";
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

export class ServiceInstanceBinding {
  id: string = "";
  serviceInstanceId: string = "";
  name: string = "";
  namespace: string = "";
}